package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeGroupElectionRepo struct {
	members []SupplierGroupSchedulingElectionMember
}

func (f *fakeGroupElectionRepo) ListGroupSchedulingElectionMembers(_ context.Context, _ int) ([]SupplierGroupSchedulingElectionMember, error) {
	return f.members, nil
}

type fakeGroupElectionStore struct {
	mu    sync.Mutex
	calls map[int64]bool
	// extraUpdates 记录「连续失败轮次」等 extra 回写，用于验证计数确实落库了。
	extraUpdates map[int64]map[string]any
	// extraErr 非 nil 时让 extra 回写失败，用于验证记账失败不会静默吞掉。
	extraErr error
	// schedulableErr 非 nil 时让「调度开关写库」失败，用于验证明细会回滚而不是记成一次切换。
	schedulableErr error
}

func newFakeGroupElectionStore() *fakeGroupElectionStore {
	return &fakeGroupElectionStore{calls: map[int64]bool{}, extraUpdates: map[int64]map[string]any{}}
}

func (f *fakeGroupElectionStore) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.schedulableErr != nil {
		return f.schedulableErr
	}
	f.calls[id] = schedulable
	return nil
}

func (f *fakeGroupElectionStore) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.extraErr != nil {
		return f.extraErr
	}
	if f.extraUpdates[id] == nil {
		f.extraUpdates[id] = map[string]any{}
	}
	for key, value := range updates {
		f.extraUpdates[id][key] = value
	}
	return nil
}

func TestGroupElectionNormalizeConfigDefaults(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{
		TopN:             0,
		DisabledGroupIDs: []int64{5, 0, -1, 5, 3},
	})
	require.Equal(t, DefaultSupplierGroupSchedulingElectionTopN, cfg.TopN)
	require.Equal(t, []int64{3, 5}, cfg.DisabledGroupIDs)

	capped := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 9999})
	require.Equal(t, MaxSupplierGroupSchedulingElectionTopN, capped.TopN)
}

// 单组单活：失败且开着的要过闸门（默认阈值 2，首次失败不关）；成功里连续成功最多的开启；
// 成功落选的关掉；未测过的不动。
func TestGroupElectionSingleGroupTopOne(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 10, Schedulable: true, LastTestStatus: "failed", HealthyCount: 0},
		{GroupID: 1, GroupName: "G1", AccountID: 11, Schedulable: false, LastTestStatus: "success", HealthyCount: 7},
		{GroupID: 1, GroupName: "G1", AccountID: 12, Schedulable: true, LastTestStatus: "success", HealthyCount: 3},
		{GroupID: 1, GroupName: "G1", AccountID: 13, Schedulable: false, LastTestStatus: "", HealthyCount: 0},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	// 10 首次失败未达阈值（1/2）→ 本轮不关，但要记上一笔；11 最优→开；
	// 12 成功但落选→关；13 未测→跳过（不写库）。
	_, touched10 := store.calls[10]
	require.False(t, touched10, "默认阈值 2，第一次失败不应立刻关")
	require.Equal(t, 1, store.extraUpdates[10][supplierGroupElectionFailedCountExtraKey], "失败轮次必须落库，下一轮判定要用")
	require.Equal(t, true, store.calls[11])
	require.Equal(t, false, store.calls[12])
	_, touched13 := store.calls[13]
	require.False(t, touched13, "未测过的账号不应被改动")

	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 1, result.DisabledCount) // 只有 12
	require.Equal(t, 1, result.SkippedCount)  // 13
	require.Equal(t, 1, result.PendingCount, "被闸门拦住的账号要单独计数，运行摘要才看得到")
	// 「故意不关」的账号必须出现在明细里，否则运维只看到"关闭 1 个"，不知道还有个失败账号在等阈值。
	require.Len(t, result.Items, 3)
	require.Equal(t, int64(10), result.Items[0].AccountID)
	require.Equal(t, "连续失败 1/2 次，未达阈值，暂不关闭", result.Items[0].Reason)
	require.Equal(t, SupplierGroupSchedulingElectionActionNone, result.Items[0].Action)
}

// TopN=2：连续成功前两名保持/开启，第三名落选被关。
func TestGroupElectionTopN(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 21, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
		{GroupID: 1, AccountID: 22, Schedulable: false, LastTestStatus: "success", HealthyCount: 5},
		{GroupID: 1, AccountID: 23, Schedulable: true, LastTestStatus: "success", HealthyCount: 1},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 2}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[21])
	require.Equal(t, true, store.calls[22])
	require.Equal(t, false, store.calls[23])
}

// 多分组 union-enable：账号在 G1 落选但在 G2 是最优 → 应保持/开启，不被 G1 关掉。
func TestGroupElectionMultiGroupUnionEnable(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// G1：31 更优，32 落选
		{GroupID: 1, AccountID: 31, Schedulable: true, LastTestStatus: "success", HealthyCount: 8},
		{GroupID: 1, AccountID: 32, Schedulable: false, LastTestStatus: "success", HealthyCount: 2},
		// G2：32 是唯一成功账号（最优）
		{GroupID: 2, AccountID: 32, Schedulable: false, LastTestStatus: "success", HealthyCount: 2},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	// 31 在 G1 最优且本就开着 → 保持不变，不写库。
	_, touched31 := store.calls[31]
	require.False(t, touched31, "31 已开且最优，无需写库")
	require.Equal(t, true, store.calls[32], "32 在 G2 最优应开启（union-enable），不被 G1 落选关掉")
}

// disabled 分组整组跳过：即便有失败开着的账号也不动。
func TestGroupElectionDisabledGroupSkipped(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 9, AccountID: 90, Schedulable: true, LastTestStatus: "failed", HealthyCount: 0},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, DisabledGroupIDs: []int64{9}}, time.Now())
	require.NoError(t, err)
	require.Empty(t, store.calls)
	require.Equal(t, 0, result.GroupCount)
	require.Equal(t, 0, result.AccountCount)
}

// 并列打破：连续成功次数相同，最近测试者优先。
func TestGroupElectionTieBreakByLastTested(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 41, Schedulable: false, LastTestStatus: "success", HealthyCount: 4, LastTestedAt: older},
		{GroupID: 1, AccountID: 42, Schedulable: false, LastTestStatus: "success", HealthyCount: 4, LastTestedAt: newer},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[42], "并列时最近测试者(42)胜出")
	_, touched41 := store.calls[41]
	require.False(t, touched41, "41 本就未开且落选，无需写库")
}

// 黏性 tie-break：次数并列时，优先保留当前已开启调度的账号，避免赢家在两者间横跳。
// 这里 61 已开、62 未开，两者次数都是 5、且 62 的 last_tested_at 更新（若无黏性会被 62 抢走）。
func TestGroupElectionTieBreakPrefersSchedulable(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 61, Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestedAt: older},
		{GroupID: 1, AccountID: 62, Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestedAt: newer},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	// 61 黏住名额：本就开着且未被严格超过 → 保持不变、不写库。
	_, touched61 := store.calls[61]
	require.False(t, touched61, "并列时应保留当前已开启的 61，不写库")
	// 62 虽然测试更近，但没超过 61 → 落选，本就未开 → 也不写库。
	_, touched62 := store.calls[62]
	require.False(t, touched62, "62 并列落选且本就未开，无需写库")
}

// 严格超过才换人：当前赢家次数被反超时，正常切换。
func TestGroupElectionSwitchesWhenStrictlyBeaten(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 71, Schedulable: true, LastTestStatus: "success", HealthyCount: 5},
		{GroupID: 1, AccountID: 72, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, false, store.calls[71], "71 被严格超过 → 关闭")
	require.Equal(t, true, store.calls[72], "72 次数更高 → 开启")
}

// 闸门一：全组无成功账号时，失败账号一个都不能关 —— 关掉最后一个就是把分组关成空组，
// 落到调度上就是请求全量失败；留着一个坏账号至少还有恢复的可能，所以交给人工确认。
// 注意这条优先于阈值：51 的失败轮次早已超过阈值，仍然不能关。
func TestGroupElectionKeepsAccountWhenNoAlternative(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 51, Schedulable: true, LastTestStatus: "failed", FailedCount: 99},
		{GroupID: 1, AccountID: 52, Schedulable: true, LastTestStatus: "failed", FailedCount: 99},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Empty(t, store.calls, "没有备选账号时不应关闭任何一个")
	require.Equal(t, 0, result.DisabledCount)
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 2, result.UnchangedCount)
	require.Equal(t, 2, result.KeptCount, "两个都是分组里最后的账号，都要计数")
	require.Equal(t, 0, result.PendingCount, "保底优先于阈值，不该被算成待观察")

	require.Len(t, result.Items, 2)
	require.Equal(t, int64(51), result.Items[0].AccountID)
	require.Equal(t, SupplierGroupSchedulingElectionReasonKeepLastOne, result.Items[0].Reason)
	require.Equal(t, int64(52), result.Items[1].AccountID)
	require.Equal(t, SupplierGroupSchedulingElectionReasonKeepLastOne, result.Items[1].Reason)
}

// 闸门二：连续失败两轮（默认阈值 2）才真正关闭，第一轮只记账。
func TestGroupElectionClosesAfterTwoConsecutiveFailures(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 61, Schedulable: true, LastTestStatus: "failed"},
		{GroupID: 1, AccountID: 62, Schedulable: false, LastTestStatus: "success", HealthyCount: 3},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	// 第一轮：只记账，不动调度。
	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	_, touched := store.calls[61]
	require.False(t, touched, "第一次失败不应关闭")
	require.Equal(t, 1, store.extraUpdates[61][supplierGroupElectionFailedCountExtraKey])
	require.Equal(t, 1, result.UnchangedCount)

	// 第二轮：把上一轮回写的计数灌回仓储（模拟下一轮读到的数据），此时才关。
	repo.members[0].FailedCount = 1
	result, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, false, store.calls[61], "连续失败 2/2 达到阈值，应当关闭")
	require.Equal(t, 1, result.DisabledCount)
	// 计数封顶到阈值，不会无限往上加。
	require.Equal(t, DefaultSupplierGroupSchedulingElectionFailureThreshold, store.extraUpdates[61][supplierGroupElectionFailedCountExtraKey])
}

// 阈值配成 1 即退回「一次失败立刻关」的旧行为，保证想要严格策略的人有得选。
func TestGroupElectionFailureThresholdOneClosesImmediately(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 71, Schedulable: true, LastTestStatus: "failed"},
		{GroupID: 1, AccountID: 72, Schedulable: false, LastTestStatus: "success", HealthyCount: 3},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, FailureThreshold: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, false, store.calls[71], "阈值 1 时首次失败即关")
	require.Equal(t, 1, result.DisabledCount)
	require.Equal(t, SupplierGroupSchedulingElectionReasonFailed, result.Items[0].Reason)
}

// 成功必须清零计数：否则账号一次失败后攒下的"欠账"会让它下次刚失败就被立刻关掉。
func TestGroupElectionSuccessResetsFailedCount(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 81, Schedulable: true, LastTestStatus: "success", HealthyCount: 4, FailedCount: 3},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, store.extraUpdates[81][supplierGroupElectionFailedCountExtraKey], "成功时连续失败计数必须清零")
	require.Empty(t, store.calls, "唯一候选且已开着，无需改调度")
	// 计数本身就是 0 的账号不该产生无意义的写库。
	store2 := newFakeGroupElectionStore()
	svc2 := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 82, Schedulable: true, LastTestStatus: "success", HealthyCount: 4},
	}}, store2)
	_, err = svc2.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Empty(t, store2.extraUpdates, "计数没有变化时不写库")
}

// 记账失败不能静默：调度裁决照常，但必须在明细里留下错误信息。
func TestGroupElectionFailedCountWriteErrorSurfaces(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 91, Schedulable: true, LastTestStatus: "failed"},
		{GroupID: 1, AccountID: 92, Schedulable: true, LastTestStatus: "success", HealthyCount: 3},
	}}
	store := newFakeGroupElectionStore()
	store.extraErr = errors.New("extra 写库失败")
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	// 92 已开着且是最优 → 无变更不入明细；91 因记账失败必须出现在明细里。
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(91), result.Items[0].AccountID)
	require.Contains(t, result.Items[0].ErrorMessage, "连续失败计数回写失败")
	require.Equal(t, 0, result.FailedWriteCount, "记账失败不改变调度裁决，也不计入写库失败数")
	_, touched := store.calls[91]
	require.False(t, touched)
}

// 调度开关写库失败时，明细里的 after 必须**回滚成 before**。
// 这条是「调度切换日志」的读侧契约：那份日志只取 schedulable_before <> schedulable_after 的条目，
// 一旦这里留下"想关但没关成"的 after=false，日志里就会多出一条根本没发生过的切换，
// 而且没有任何报错 —— 是最难被发现的那类假数据。
func TestGroupElectionSchedulableWriteFailureRollsBackDetail(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 71, Schedulable: true, LastTestStatus: "success", HealthyCount: 5},
		{GroupID: 1, AccountID: 72, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
	}}
	store := newFakeGroupElectionStore()
	store.schedulableErr = errors.New("schedulable 写库失败")
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, 2, result.FailedWriteCount, "两个账号的开关都没写成")
	require.Len(t, result.Items, 2)
	for _, item := range result.Items {
		require.Equal(t, item.SchedulableBefore, item.SchedulableAfter, "写库失败必须回滚 after，否则会被切换日志当成一次真实切换")
		require.Equal(t, SupplierGroupSchedulingElectionActionNone, item.Action)
		require.Equal(t, SupplierGroupSchedulingElectionReasonWriteFailed, item.Reason)
		require.NotEmpty(t, item.ErrorMessage)
	}
}

// 同平台、连续成功次数相同：用时短的当选。
func TestGroupElectionPrefersFasterLatencySamePlatform(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 81, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 1000},
		{GroupID: 1, AccountID: 82, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 200},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[82], "同平台同次数，用时更短的 82 应当选")
	_, touched81 := store.calls[81]
	require.False(t, touched81, "81 落选且本就未开，无需写库")
}

// 明细存在的意义是解释"为什么是它当选"，所以必须把参与打分的用时带出来；
// 没有耗时数据的账号（0）不能输出该字段，否则前端会把"未知"显示成 0 ms（读起来像极快）。
func TestGroupElectionDetailItemsCarryLatency(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 111, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 7, LastTestLatencyMs: 800},
		{GroupID: 1, AccountID: 112, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 3, LastTestLatencyMs: 120},
		{GroupID: 1, AccountID: 113, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 0},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	// 112 次数少但明显更快：即便在任者 111 的延迟按迟滞折算(800×0.85=680)仍远慢于 112(120)，
	// 112 = 0.3 + 0.5×1 = 0.8 > 111 = 0.7 + 0.5×0 = 0.7，靠用时翻盘。
	require.Equal(t, []int64{112}, result.Groups[0].WinnerIDs)

	require.Len(t, result.Items, 3)
	require.Equal(t, int64(111), result.Items[0].AccountID)
	require.Equal(t, int64(800), result.Items[0].LatencyMs)
	require.Equal(t, int64(112), result.Items[1].AccountID)
	require.Equal(t, int64(120), result.Items[1].LatencyMs)
	require.Equal(t, int64(113), result.Items[2].AccountID)
	require.Equal(t, int64(0), result.Items[2].LatencyMs)

	raw, err := json.Marshal(result.Items[2])
	require.NoError(t, err)
	require.NotContains(t, string(raw), "latency_ms", "无耗时数据时必须省略该字段")
}

// 跨平台不比用时：慢但次数多的赢，避免低延迟平台长期垄断赢家。
func TestGroupElectionCrossPlatformIgnoresLatency(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 91, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 100},
		{GroupID: 1, AccountID: 92, Platform: "gemini", Schedulable: false, LastTestStatus: "success", HealthyCount: 6, LastTestLatencyMs: 5000},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[92], "跨平台时用时不应参与比较，次数多的 92 当选")
	_, touched91 := store.calls[91]
	require.False(t, touched91)
}

// 防抖：次数差超过用时能抵的上限时，即使慢很多也应由次数多的保持当选，
// 避免一次网络抖动就换掉长期稳定的赢家。
func TestGroupElectionLatencyCannotOvercomeLargeCountGap(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 101, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 8, LastTestLatencyMs: 3000},
		{GroupID: 1, AccountID: 102, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 2, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[101], "次数差 6 已超过用时上限，101 应保住名额")
	_, touched102 := store.calls[102]
	require.False(t, touched102)
}

// 次数差在用时可抵范围内：快的一方可以翻盘。
func TestGroupElectionLatencyCanOvercomeSmallCountGap(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 111, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000},
		{GroupID: 1, AccountID: 112, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 3, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[112], "次数差 2 在用时可抵范围内，更快的 112 应翻盘")
	_, touched111 := store.calls[111]
	require.False(t, touched111)
}

// 次数封顶：连续成功只增不减，不封顶会让现任永久固化。
// 50 次与 12 次都已封顶，此时应由用时决胜。
func TestGroupElectionCountScoreCapSticksWinningSeat(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 121, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 50, LastTestLatencyMs: 3000},
		{GroupID: 1, AccountID: 122, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 12, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[122], "次数封顶后由用时决胜，更快的 122 当选")
	_, touched121 := store.calls[121]
	require.False(t, touched121)
}

// 同平台只有自己一个样本时无从比较"谁更快"，应回落中性分，
// 不能因为 132 的绝对值更小就让它赢。
func TestGroupElectionSinglePlatformSampleFallsBackToNeutral(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 131, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 9000},
		{GroupID: 1, AccountID: 132, Platform: "gemini", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[131], "各平台均只有一个样本，用时不可比，按账号 ID 兜底")
	_, touched132 := store.calls[132]
	require.False(t, touched132)
}

// 直接钉住综合分的归一化语义：两项各自归一到 [0,1] 再按权重合成。
func TestGroupElectionScoresNormalizeWithinPlatform(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{AccountID: 141, Platform: "openai", HealthyCount: 5, LastTestLatencyMs: 100},
		{AccountID: 142, Platform: "openai", HealthyCount: 5, LastTestLatencyMs: 300},
		{AccountID: 143, Platform: "gemini", HealthyCount: 5, LastTestLatencyMs: 50},
	}
	scores := supplierGroupSchedulingElectionScores(members, 1.0, 0.5, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin)

	// 三者次数都是 5，归一后同为 0.5，差异全部来自用时项。
	require.InDelta(t, 0.5+0.5*1.0, scores[141], 1e-9, "同平台最快者用时归一为 1")
	require.InDelta(t, 0.5+0.5*0.0, scores[142], 1e-9, "同平台最慢者用时归一为 0")
	require.InDelta(t, 0.5+0.5*0.5, scores[143], 1e-9, "单样本平台拿中性分 0.5")
}

// 次数封顶：超过上限的连续成功不再加分，避免资历压制实际表现。
func TestGroupElectionScoresCapCountContribution(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{AccountID: 151, Platform: "openai", HealthyCount: 50, LastTestLatencyMs: 100},
		{AccountID: 152, Platform: "openai", HealthyCount: 10, LastTestLatencyMs: 900},
	}
	scores := supplierGroupSchedulingElectionScores(members, 1.0, 0.5, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin)

	require.InDelta(t, 1.0+0.5*1.0, scores[151], 1e-9)
	require.InDelta(t, 1.0+0.5*0.0, scores[152], 1e-9, "次数达到上限后归一为 1，不再拉开差距")
}

// 权重缺省回落默认值，保证旧配置（config_json 里没有这两个字段）行为不变；
// 用时权重显式配 0 是合法配置，不能被当成缺失一并回落。
func TestGroupElectionNormalizeConfigWeights(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1})
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionCountWeight, cfg.CountWeight, 1e-9)
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionLatencyWeight, cfg.LatencyWeight, 1e-9)

	// 0 与"字段缺失"在 JSON 里无法区分，一律按未配置回落，否则老任务升级后会静默关掉用时。
	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{
		TopN: 1, CountWeight: 2, LatencyWeight: 0,
	})
	require.InDelta(t, 2.0, cfg.CountWeight, 1e-9)
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionLatencyWeight, cfg.LatencyWeight, 1e-9, "0 视为未配置，回落默认")

	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{
		TopN: 1, CountWeight: 9999, LatencyWeight: -1,
	})
	require.InDelta(t, MaxSupplierGroupSchedulingElectionWeight, cfg.CountWeight, 1e-9)
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionLatencyWeight, cfg.LatencyWeight, 1e-9)
}

// 连续失败阈值同样要能从旧配置平滑升级：config_json 里没有该字段时是 0，
// 一律按默认 2 处理，配 1 才能退回「一次失败立刻关」。
func TestGroupElectionNormalizeConfigFailureThreshold(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1})
	require.Equal(t, DefaultSupplierGroupSchedulingElectionFailureThreshold, cfg.FailureThreshold)

	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1, FailureThreshold: 1})
	require.Equal(t, 1, cfg.FailureThreshold, "显式配 1 必须生效（退回失败即关）")

	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1, FailureThreshold: 9999})
	require.Equal(t, MaxSupplierGroupSchedulingElectionFailureThreshold, cfg.FailureThreshold)
}

// 权重真的能改变当选结果：同样两个账号，用时权重为 0 时次数多的赢，
// 权重提到与次数等同时更快的赢。
func TestGroupElectionLatencyWeightChangesWinner(t *testing.T) {
	// 171 次数满分但最慢；172 次数只有一半但最快。
	newMembers := func() []SupplierGroupSchedulingElectionMember {
		return []SupplierGroupSchedulingElectionMember{
			{GroupID: 1, AccountID: 171, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 10, LastTestLatencyMs: 1000},
			{GroupID: 1, AccountID: 172, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 100},
		}
	}

	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	// 用时权重压到很低：次数的分量占绝对多数，171 应当选。
	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, CountWeight: 1, LatencyWeight: 0.1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[171], "用时权重很低时由次数说话")
	_, touched172 := store.calls[172]
	require.False(t, touched172)

	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, CountWeight: 1, LatencyWeight: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[172], "提高用时权重后更快的 172 应翻盘")
	_, touched171 := store.calls[171]
	require.False(t, touched171)
}

// 迟滞死区：挑战者更快但没快过死区比例(默认 15%)时，保持在任者不换人 —— 这是压住
// 「两个正常账号每轮对拍」抖动的核心。71 在任 1000ms，72 只快 10%(900ms) < 15%，应保住 71。
// 两者次数都封顶(20)，胜负只看延迟，专门验证死区在「两候选」这种最易抖动的场景里确实生效。
func TestGroupElectionHysteresisKeepsIncumbentWithinMargin(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 71, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000},
		{GroupID: 1, AccountID: 72, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 900},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, []int64{71}, result.Groups[0].WinnerIDs, "挑战者只快 10% 未过 15% 死区，应保住在任者 71")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount)
	_, touched71 := store.calls[71]
	require.False(t, touched71, "在任者保持当选，不写库")
	_, touched72 := store.calls[72]
	require.False(t, touched72, "挑战者落选且本就未开，不写库")
}

// 迟滞死区：挑战者快过死区比例时正常换人。72 比 71 快 30%(700 vs 1000) > 15%，应夺位。
func TestGroupElectionHysteresisSwitchesBeyondMargin(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 71, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000},
		{GroupID: 1, AccountID: 72, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 700},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, false, store.calls[71], "被快 30% 的挑战者盖过死区 → 关闭")
	require.Equal(t, true, store.calls[72], "快过 15% 死区 → 开启")
}

// 窗口平均 + 成功率惩罚：201 窗口均值更低(800ms)但成功率极差(10/60)，被成功率放大成 4000ms；
// 202 稍慢(2000ms)但稳(58/60)。202 应当选 —— 只算成功样本会掩盖「偶尔快一下、实则一直失败」的账号，
// 成功率把被剔除的失败重新计入代价。同时验证明细里显示的是窗口均值(base)，不带惩罚。
func TestGroupElectionWindowAverageWithSuccessRatePenalty(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 201, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20,
			AvgLatencyMs: 800, LatencySuccessCount: 10, LatencyTotalCount: 60},
		{GroupID: 1, AccountID: 202, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20,
			AvgLatencyMs: 2000, LatencySuccessCount: 58, LatencyTotalCount: 60},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[202], "稳定的 202 应当选，不该被偶尔快一下的 201 抢走")
	_, touched201 := store.calls[201]
	require.False(t, touched201)

	require.Len(t, result.Items, 1, "只有当选并开启的 202 产生变更明细")
	require.Equal(t, int64(202), result.Items[0].AccountID)
	require.Equal(t, int64(2000), result.Items[0].LatencyMs, "明细显示窗口均值(base)，不含成功率惩罚")
}

// 少样本不信任：201 窗口均值很低(100ms)但只有 2 个成功样本(< 默认 3)，不信任该均值，
// 回退到最近单值(5000ms) → 落选；202 单值 200ms 当选。防止几次抽样就把赢家换掉。
func TestGroupElectionLatencyFallsBackWhenTooFewSamples(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 211, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20,
			AvgLatencyMs: 100, LatencySuccessCount: 2, LatencyTotalCount: 2, LastTestLatencyMs: 5000},
		{GroupID: 1, AccountID: 212, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20,
			LastTestLatencyMs: 200},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[212], "样本不足时 211 回退单值 5000ms，更快的 212 当选")
	_, touched211 := store.calls[211]
	require.False(t, touched211)
	require.Equal(t, int64(200), result.Items[0].LatencyMs, "212 明细显示其单值 200ms")
}

// 直接钉住有效延迟(显示 base 与评分 scoring)的口径：窗口可信时取均值，评分再按成功率惩罚。
func TestGroupElectionLatencyHelper(t *testing.T) {
	// 可信窗口：成功 10、总 60 → 成功率 0.167 但被 floor 0.2 兜住，评分 = 800 / 0.2 = 4000。
	base, scoring := supplierGroupSchedulingElectionLatency(SupplierGroupSchedulingElectionMember{
		AvgLatencyMs: 800, LatencySuccessCount: 10, LatencyTotalCount: 60, LastTestLatencyMs: 111,
	}, DefaultSupplierGroupSchedulingElectionLatencyMinSamples)
	require.Equal(t, int64(800), base, "显示用取窗口均值")
	require.Equal(t, int64(4000), scoring, "评分用按成功率惩罚，成功率低于地板时按地板 0.2 除")

	// 成功率高：评分几乎等于均值本身。
	base, scoring = supplierGroupSchedulingElectionLatency(SupplierGroupSchedulingElectionMember{
		AvgLatencyMs: 2000, LatencySuccessCount: 58, LatencyTotalCount: 60, LastTestLatencyMs: 111,
	}, DefaultSupplierGroupSchedulingElectionLatencyMinSamples)
	require.Equal(t, int64(2000), base)
	require.InDelta(t, 2069, scoring, 2, "2000 / (58/60) ≈ 2069")

	// 样本不足：base 与 scoring 都回退最近单值，绝不比现状更差。
	base, scoring = supplierGroupSchedulingElectionLatency(SupplierGroupSchedulingElectionMember{
		AvgLatencyMs: 100, LatencySuccessCount: 2, LatencyTotalCount: 2, LastTestLatencyMs: 5000,
	}, DefaultSupplierGroupSchedulingElectionLatencyMinSamples)
	require.Equal(t, int64(5000), base)
	require.Equal(t, int64(5000), scoring)
}

// 迟滞死区、平均窗口、最少样本三项同样要能从旧配置平滑升级（缺失=0 → 默认），并各自 clamp 上限。
func TestGroupElectionNormalizeConfigLatencyAndMargin(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1})
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionSwitchMargin, cfg.SwitchMargin, 1e-9)
	require.Equal(t, DefaultSupplierGroupSchedulingElectionLatencyWindowMinutes, cfg.LatencyWindowMinutes)
	require.Equal(t, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, cfg.LatencyMinSamples)

	capped := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{
		TopN: 1, SwitchMargin: 9, LatencyWindowMinutes: 999999, LatencyMinSamples: 999999,
	})
	require.InDelta(t, MaxSupplierGroupSchedulingElectionSwitchMargin, capped.SwitchMargin, 1e-9)
	require.Equal(t, MaxSupplierGroupSchedulingElectionLatencyWindowMinutes, capped.LatencyWindowMinutes)
	require.Equal(t, MaxSupplierGroupSchedulingElectionLatencyMinSamples, capped.LatencyMinSamples)

	// 显式配一个极小正数不被当成缺失（用于近乎关闭迟滞）。
	tiny := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1, SwitchMargin: 0.0001})
	require.InDelta(t, 0.0001, tiny.SwitchMargin, 1e-12)
}

// 在任者健康锁定：开关打开且分组里开着的账号(301)测试正常时，即便存在明显更优的挑战者(302)，
// 也直接保留现状、跳过换人；开关关闭时同样的数据则正常换人，证明锁定确实由开关控制。
func TestGroupElectionKeepHealthyIncumbentLocksGroup(t *testing.T) {
	newMembers := func() []SupplierGroupSchedulingElectionMember {
		return []SupplierGroupSchedulingElectionMember{
			{GroupID: 1, AccountID: 301, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000},
			{GroupID: 1, AccountID: 302, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 100},
		}
	}

	// 分组 1 被 opt-in 锁定：保留 301，不动 302。
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{1}}, time.Now())
	require.NoError(t, err)
	require.Equal(t, []int64{301}, result.Groups[0].WinnerIDs, "在任者健康 → 锁定分组，保留 301")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount)
	require.Empty(t, store.calls, "锁定分组不产生任何调度写库")

	// 分组 1 未被 opt-in：同样数据下 302 明显更优 → 正常换人，证明差异来自分组级开关。
	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[302], "未锁定时更优的 302 应正常当选")
	require.Equal(t, false, store.calls[301], "未锁定时落选的 301 应被关闭")

	// 只锁定别的分组(999)时，分组 1 不受影响，仍正常换人 —— 验证锁定确实按分组粒度生效。
	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{999}}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[302], "只锁定了 999，分组 1 不在列表内应正常换人")
}

// 只在在任者健康时锁定：开着的账号(311)测试失败时不锁定，仍走正常择优 —— 更优的 312 被开启，
// 绝不因为开关就把一个坏分组锁死。
func TestGroupElectionKeepHealthyIncumbentRunsElectionWhenIncumbentFailed(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 311, Platform: "openai", Schedulable: true, LastTestStatus: "failed"},
		{GroupID: 1, AccountID: 312, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{1}}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[312], "开着的账号失败 → 不锁定，正常择优开启更优的 312")
}

// 必需模型覆盖：TopN=1 下最优账号 401 只支持 bbb，若不兜底 aaa 会随换人断供。
// 配了必需模型 aaa → 额外开启唯一的健康支持者 402（哪怕它综合分更低），保证 aaa 不断供。
func TestGroupElectionRequiredModelSupplementsHealthySupporter(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 401, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, ModelMapping: map[string]any{"bbb": "bbb"}},
		{GroupID: 1, AccountID: 402, Platform: "anthropic", Schedulable: false, LastTestStatus: "success", HealthyCount: 1, ModelMapping: map[string]any{"aaa": "aaa"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                  1,
		RequiredModelsByGroup: map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[402], "aaa 的唯一健康支持者 402 应被必需模型覆盖补选开启")
	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 0, result.RequiredModelUncoveredCount, "aaa 有健康支持者，不应告警")
	require.ElementsMatch(t, []int64{401, 402}, result.Groups[0].WinnerIDs, "最优 401 + 必需模型支持者 402 都应是赢家")
}

// 必需模型已被最优赢家覆盖：赢家 411 本身支持 aaa → 不额外开启任何人，落选的 412 保持关闭。
func TestGroupElectionRequiredModelAlreadyCoveredByWinner(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 411, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, ModelMapping: map[string]any{"aaa": "aaa", "bbb": "bbb"}},
		{GroupID: 1, AccountID: 412, Platform: "anthropic", Schedulable: false, LastTestStatus: "success", HealthyCount: 1, ModelMapping: map[string]any{"ccc": "ccc"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                  1,
		RequiredModelsByGroup: map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	_, touched := store.calls[412]
	require.False(t, touched, "aaa 已被赢家 411 覆盖，不应补选 412")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.RequiredModelUncoveredCount)
	require.Equal(t, []int64{411}, result.Groups[0].WinnerIDs)
}

// 必需模型的支持者全部失败：不硬留失败账号（让健康的 421 生效），只记一条告警并带上失败支持者 ID。
func TestGroupElectionRequiredModelAllSupportersFailedWarns(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 421, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, ModelMapping: map[string]any{"bbb": "bbb"}},
		{GroupID: 1, GroupName: "G1", AccountID: 422, Platform: "anthropic", Schedulable: false, LastTestStatus: "failed", ModelMapping: map[string]any{"aaa": "aaa"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                  1,
		RequiredModelsByGroup: map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	_, touched := store.calls[422]
	require.False(t, touched, "aaa 的支持者全失败 → 不硬开失败账号")
	require.Equal(t, 1, result.RequiredModelUncoveredCount)
	require.Len(t, result.RequiredModelWarnings, 1)
	require.Equal(t, "aaa", result.RequiredModelWarnings[0].Model)
	require.Equal(t, int64(1), result.RequiredModelWarnings[0].GroupID)
	require.Equal(t, "G1", result.RequiredModelWarnings[0].GroupName)
	require.Equal(t, []int64{422}, result.RequiredModelWarnings[0].FailedAccountIDs)
}

// 必需模型是硬底线：即便分组被「在任者健康锁定」，锁定组的在任 501 没覆盖 aaa 时，
// 仍会额外开启健康支持者 502（只增开、不动在任，不破坏锁定语义）。
func TestGroupElectionRequiredModelSupplementsLockedGroup(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 501, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, ModelMapping: map[string]any{"bbb": "bbb"}},
		{GroupID: 1, AccountID: 502, Platform: "anthropic", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, ModelMapping: map[string]any{"aaa": "aaa"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
		RequiredModelsByGroup:        map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[502], "必需模型是硬底线，锁定组也要补齐 aaa 的支持者 502")
	_, touched501 := store.calls[501]
	require.False(t, touched501, "锁定组的在任 501 应保持不动")
	require.ElementsMatch(t, []int64{501, 502}, result.Groups[0].WinnerIDs)
}

// 空 model_mapping 账号支持所有模型：赢家 511 无映射即覆盖 aaa → 不补选、不告警。
func TestGroupElectionRequiredModelCoveredByUnmappedAccount(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 511, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20},
		{GroupID: 1, AccountID: 512, Platform: "anthropic", Schedulable: false, LastTestStatus: "success", HealthyCount: 1, ModelMapping: map[string]any{"aaa": "aaa"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                  1,
		RequiredModelsByGroup: map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	_, touched := store.calls[512]
	require.False(t, touched, "无映射账号支持所有模型，aaa 已覆盖，不应补选 512")
	require.Equal(t, 0, result.RequiredModelUncoveredCount)
	require.Equal(t, []int64{511}, result.Groups[0].WinnerIDs)
}

// 归一化必需模型：丢弃非正 groupID，模型名 trim、按小写去重保序，清洗后为空的分组整条丢弃。
func TestGroupElectionNormalizeRequiredModels(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{
		RequiredModelsByGroup: map[int64][]string{
			1:  {" aaa ", "aaa", "AAA", "bbb", ""},
			0:  {"x"},
			-3: {"y"},
			2:  {"  ", ""},
		},
	})
	require.Equal(t, []string{"aaa", "bbb"}, cfg.RequiredModelsByGroup[1])
	_, hasZero := cfg.RequiredModelsByGroup[0]
	require.False(t, hasZero, "非正 groupID 应被丢弃")
	_, hasNeg := cfg.RequiredModelsByGroup[-3]
	require.False(t, hasNeg)
	_, hasEmpty := cfg.RequiredModelsByGroup[2]
	require.False(t, hasEmpty, "清洗后为空的分组应整条丢弃")
}
