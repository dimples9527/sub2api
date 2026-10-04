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

// 分组级 TopN 覆盖的清洗：丢弃非正 groupID 与非正 TopN（等于该组回落全局），超上限钳到上限，
// 清洗后为空则收敛成 nil（与 required_models 同口径，空覆盖不落库）。
func TestGroupElectionNormalizeTopNByGroup(t *testing.T) {
	require.Nil(t, normalizeSupplierGroupElectionTopNByGroup(nil))
	require.Nil(t, normalizeSupplierGroupElectionTopNByGroup(map[int64]int{}))
	require.Nil(t, normalizeSupplierGroupElectionTopNByGroup(map[int64]int{0: 3, -1: 2, 5: 0, 6: -4}),
		"非正 groupID 与非正 TopN 全部丢弃后应为 nil")

	cleaned := normalizeSupplierGroupElectionTopNByGroup(map[int64]int{
		1:  2,
		2:  9999, // 超上限 → 钳到 Max
		3:  0,    // 非正 → 丢弃（该组回落全局）
		-4: 5,    // 非正 groupID → 丢弃
	})
	require.Equal(t, map[int64]int{
		1: 2,
		2: MaxSupplierGroupSchedulingElectionTopN,
	}, cleaned)
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

// 分组级 TopN 覆盖：全局 TopN=1，但 group 1 单独配成 2 → 该组开前两名、关第三名；
// 未配置覆盖的 group 2 仍回落全局 TopN=1，只开最优、关次优。证明覆盖只作用于命中的分组。
func TestGroupElectionTopNByGroupOverride(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// group 1：override=2，连续成功前两名(31,32)开，第三名(33)超额被关。
		{GroupID: 1, AccountID: 31, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
		{GroupID: 1, AccountID: 32, Schedulable: false, LastTestStatus: "success", HealthyCount: 5},
		{GroupID: 1, AccountID: 33, Schedulable: true, LastTestStatus: "success", HealthyCount: 1},
		// group 2：无 override，回落全局 TopN=1，只开最优(41)、关次优(42)。
		{GroupID: 2, AccountID: 41, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
		{GroupID: 2, AccountID: 42, Schedulable: true, LastTestStatus: "success", HealthyCount: 5},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:        1,
		TopNByGroup: map[int64]int{1: 2},
	}, time.Now())
	require.NoError(t, err)

	// group 1 生效 TopN=2：31、32 开，33 关。
	require.Equal(t, true, store.calls[31])
	require.Equal(t, true, store.calls[32])
	require.Equal(t, false, store.calls[33], "第三名超出该组生效 TopN(2)，应被关闭")
	// group 2 无覆盖 → 按全局 TopN=1：41 开、42 关。
	require.Equal(t, true, store.calls[41])
	require.Equal(t, false, store.calls[42], "未配置覆盖的分组仍按全局 TopN=1，次优应被关闭")
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
	scores := supplierGroupSchedulingElectionScores(members, 1.0, 0.5, 0.0, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin, DefaultSupplierGroupSchedulingElectionCountScoreCap, false)

	// 三者次数都是 5，归一后同为 0.5，差异全部来自用时项。
	require.InDelta(t, 0.5+0.5*1.0, scores[141].Score, 1e-9, "同平台最快者用时归一为 1")
	require.InDelta(t, 0.5+0.5*0.0, scores[142].Score, 1e-9, "同平台最慢者用时归一为 0")
	require.InDelta(t, 0.5+0.5*0.5, scores[143].Score, 1e-9, "单样本平台拿中性分 0.5")
}

// 次数封顶：超过上限的连续成功不再加分，避免资历压制实际表现。
func TestGroupElectionScoresCapCountContribution(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{AccountID: 151, Platform: "openai", HealthyCount: 50, LastTestLatencyMs: 100},
		{AccountID: 152, Platform: "openai", HealthyCount: 10, LastTestLatencyMs: 900},
	}
	scores := supplierGroupSchedulingElectionScores(members, 1.0, 0.5, 0.0, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin, DefaultSupplierGroupSchedulingElectionCountScoreCap, false)

	require.InDelta(t, 1.0+0.5*1.0, scores[151].Score, 1e-9)
	require.InDelta(t, 1.0+0.5*0.0, scores[152].Score, 1e-9, "次数达到上限后归一为 1，不再拉开差距")
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

// 次数封顶值同样要能从旧配置平滑升级：config_json 里没有该字段时是 0，
// 一律按默认 10 处理 —— 升级后老分组的当选结果必须一字不变。
func TestGroupElectionNormalizeConfigCountScoreCap(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1})
	require.Equal(t, DefaultSupplierGroupSchedulingElectionCountScoreCap, cfg.CountScoreCap)

	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1, CountScoreCap: 30})
	require.Equal(t, 30, cfg.CountScoreCap, "显式配置必须生效")

	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1, CountScoreCap: 9999})
	require.Equal(t, MaxSupplierGroupSchedulingElectionCountScoreCap, cfg.CountScoreCap)

	// 负数同样回落默认：JSON 里它和"未配置"没法区分，而拒绝非法值属于校验层（validateSupplierAutomationTask）。
	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1, CountScoreCap: -5})
	require.Equal(t, DefaultSupplierGroupSchedulingElectionCountScoreCap, cfg.CountScoreCap)
}

// 封顶值就是次数分的分母：同一个 12 次的账号，封顶 10 时满分、封顶 100 时只拿 0.12。
func TestGroupElectionScoresCountCapScalesContribution(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{AccountID: 161, Platform: "openai", HealthyCount: 12, LastTestLatencyMs: 100},
		{AccountID: 162, Platform: "openai", HealthyCount: 12, LastTestLatencyMs: 100},
	}

	// 用时权重传 0，把这一项摘掉，只看次数分的变化。
	capped := supplierGroupSchedulingElectionScores(members, 1.0, 0.0, 0.0, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin, 10, false)
	require.InDelta(t, 1.0, capped[161].Score, 1e-9, "封顶 10：12 次已到顶，拿满分")

	loose := supplierGroupSchedulingElectionScores(members, 1.0, 0.0, 0.0, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin, 100, false)
	require.InDelta(t, 0.12, loose[161].Score, 1e-9, "封顶 100：12 次只拿 12/100")
}

// 优先级分：同一组内数值越小优先级越高，min-max 映射 — 最小者 1.0、最大者 0；
// 该组开关关闭（priorityEnabled=false）时优先级一律取 0，不影响既有排序。
func TestGroupElectionScoresPriorityNormalizeWithinGroup(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{AccountID: 171, Platform: "openai", HealthyCount: 5, LastTestLatencyMs: 100, Priority: 1},
		{AccountID: 172, Platform: "openai", HealthyCount: 5, LastTestLatencyMs: 100, Priority: 3},
		{AccountID: 173, Platform: "openai", HealthyCount: 5, LastTestLatencyMs: 100, Priority: 4},
	}
	// 次数、用时都完全相同，差异只来自优先级项；打开开关、用均衡权重 0.5。
	scores := supplierGroupSchedulingElectionScores(members, 1.0, 0.5, 0.5, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin, DefaultSupplierGroupSchedulingElectionCountScoreCap, true)
	baseScore := 1.0*0.5 + 0.5*0.5 // 次数分 5/10 × 1.0 + 用时中性分 0.5 × 0.5，三者相同
	require.InDelta(t, baseScore+0.5*1.0, scores[171].Score, 1e-9, "优先级最小者（最高优先级）映射 1.0")
	require.InDelta(t, baseScore+0.5*0.0, scores[173].Score, 1e-9, "优先级最大者映射 0")
	require.InDelta(t, baseScore+0.5*(float64(4-3)/float64(4-1)), scores[172].Score, 1e-9, "中间者按比例映射")

	// 开关关闭时优先级不参与：三者得分完全一致（只看次数/用时）。
	off := supplierGroupSchedulingElectionScores(members, 1.0, 0.0, 0.5, DefaultSupplierGroupSchedulingElectionLatencyMinSamples, DefaultSupplierGroupSchedulingElectionSwitchMargin, DefaultSupplierGroupSchedulingElectionCountScoreCap, false)
	require.InDelta(t, off[171].Score, off[172].Score, 1e-9, "未开启优先级计分时优先级权重空转")
	require.Equal(t, 0.0, off[171].PriorityScore)
}

// 优先级真正改变当选结果：都健康时，优先级越高（数值越小）越占优势。
func TestGroupElectionPriorityChangesWinner(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// 两个账号健康次数、用时都相同，唯一差别是优先级：131 更低（更高优先级）。
		{GroupID: 1, Platform: "openai", AccountID: 131, Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 100, Priority: 1},
		{GroupID: 1, Platform: "openai", AccountID: 132, Schedulable: false, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 100, Priority: 2},
	}}
	// 权重故意拉大优先级话语权，验证「都健康时优先级更高者当选」。
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)
	_, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                  1,
		PriorityWeight:        DefaultSupplierGroupSchedulingElectionPriorityWeight,
		PriorityEnabledGlobal: true,
	}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[131], "都健康且其它维度相同时，更高优先级的 131 当选")
	require.Equal(t, false, store.calls[132])

	// 开关默认关闭：优先级不参与，回到「按 ID 兜底」的确定性结果（ID 小的 131 先）。
	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(repo, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[131], "未开启优先级时行为与升级前一致")
	require.Equal(t, false, store.calls[132])
}

// 优先级权重的归一化缺省回落，与 Count/Latency 同规则；分组名单去重排序落库。
func TestGroupElectionNormalizeConfigPriority(t *testing.T) {
	cfg := normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{TopN: 1})
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionPriorityWeight, cfg.PriorityWeight, 1e-9)
	require.False(t, cfg.PriorityEnabledGlobal, "默认全局关闭，与升级前一致")

	cfg = normalizeSupplierGroupSchedulingElectionConfig(SupplierGroupSchedulingElectionConfig{
		TopN: 1, PriorityWeight: 0, PriorityEnabledGroupIDs: []int64{5, 0, -1, 5, 3}, PriorityDisabledGroupIDs: []int64{7, 2, 2, 7},
	})
	require.InDelta(t, DefaultSupplierGroupSchedulingElectionPriorityWeight, cfg.PriorityWeight, 1e-9, "0 视为未配置回落默认")
	require.Equal(t, []int64{3, 5}, cfg.PriorityEnabledGroupIDs, "强制开启名单去重排序")
	require.Equal(t, []int64{2, 7}, cfg.PriorityDisabledGroupIDs, "强制关闭名单去重排序")
}

// 把封顶做成可配，必须真的改变当选结果 —— 只改归一化、忘了把值传进评分实现时，这条会报红。
// 同一组数据：封顶 10 时次数打平、更快的 122 赢；封顶 100 时资历重新说话、更老的 121 赢。
func TestGroupElectionCountScoreCapChangesWinner(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 121, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 100, LastTestLatencyMs: 3000},
		{GroupID: 1, AccountID: 122, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 12, LastTestLatencyMs: 100},
	}}

	cappedStore := newFakeGroupElectionStore()
	cappedSvc := NewSupplierGroupSchedulingElectionService(repo, cappedStore)
	_, err := cappedSvc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN: 1, CountScoreCap: DefaultSupplierGroupSchedulingElectionCountScoreCap,
	}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, cappedStore.calls[122], "封顶 10：100 次与 12 次同分，更快的 122 当选")

	looseStore := newFakeGroupElectionStore()
	looseSvc := NewSupplierGroupSchedulingElectionService(repo, looseStore)
	_, err = looseSvc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN: 1, CountScoreCap: 100,
	}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, looseStore.calls[121], "封顶 100：100 次拉开差距，资历更老的 121 当选")
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
	// 锁定保住的在任者(301)虽然 before==after、没写库，仍要落进明细：切换日志的「含未切换」
	// 视图靠它显示「在任者健康锁定，本次跳过」。挑战者 302 未开启、非在任者，不应被这条规则带出。
	require.Len(t, result.Items, 1, "锁定保住的在任者 301 要进明细，供含未切换视图展示")
	require.Equal(t, int64(301), result.Items[0].AccountID)
	require.Equal(t, result.Items[0].SchedulableBefore, result.Items[0].SchedulableAfter, "锁定保留 → 前后状态一致")
	require.True(t, supplierGroupElectionItemHasLockedIncumbent(result.Items[0]), "该明细应带 Locked && Elected 的分组裁决")

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

// 锁定分组里在任者一次抖动失败（未到关闭阈值）：不破锁、不换人。更优的挑战者 312 绝不被开出，
// 在任者 311 由失败闸门原地留着等翻盘。这正是修复「单次失败就开替补 → 抖动翻盘后两账号被永久焊死」的多活累积——
// 单在任者分组尤其致命：311 一失败 scheduledHealthy 就空了，破锁口径若只看「有没有健康在任者」，锁根本合不上。
func TestGroupElectionKeepHealthyIncumbentHoldsThroughTransientFailure(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 311, Platform: "openai", Schedulable: true, LastTestStatus: "failed"},
		{GroupID: 1, AccountID: 312, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{1}}, time.Now())
	require.NoError(t, err)
	_, opened := store.calls[312]
	require.False(t, opened, "在任者未到阈值的失败 → 保持锁定，绝不开出替补 312")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount)
	// 在任者 311 保持开着，走失败闸门「待观察」，前后状态一致。
	require.Len(t, result.Items, 1, "只有被留着待观察的 311 进明细")
	require.Equal(t, int64(311), result.Items[0].AccountID)
	require.True(t, result.Items[0].SchedulableAfter, "在任者 311 本轮仍保持开着")
	require.Equal(t, SupplierGroupSchedulingElectionActionNone, result.Items[0].Action)
	require.Equal(t, 1, result.PendingCount, "311 记为失败待观察")
}

// 锁定分组里在任者连续失败到关闭阈值：破锁，走正常择优开出替补 312，并关闭坏在任者 311。
// 破锁口径与失败闸门同步——两者都在「连续失败达到 FailureThreshold」这一刻同时触发。
func TestGroupElectionKeepHealthyIncumbentBreaksLockAtThreshold(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 311, Platform: "openai", Schedulable: true, LastTestStatus: "failed", FailedCount: 1},
		{GroupID: 1, AccountID: 312, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 100},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	// 默认阈值 2：311 已欠 1 次，本轮再失败即达 2/2 → 破锁、关它、开替补。
	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{1}}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[312], "在任者连续失败到阈值 → 破锁择优，开启更优的 312")
	require.Equal(t, false, store.calls[311], "达到阈值的坏在任者 311 被关闭")
	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 1, result.DisabledCount)
}

// 锁定分组里「当前开着的在任者」已经超过 TopN（多活已累积）：此时不能退回正常择优
// ——正常择优取的是本组综合分前 N 名，会在已有的多活之上再加一个，只增不减。
// 正解是就地收敛：只保留 TopN 个在任者，多出来的让位关闭。
func TestGroupElectionKeepHealthyIncumbentConvergesWhenOverCapacity(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 301, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 1000},
		{GroupID: 1, AccountID: 302, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 1000},
		{GroupID: 1, AccountID: 303, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 10, LastTestLatencyMs: 1000},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{1}}, time.Now())
	require.NoError(t, err)
	require.Equal(t, []int64{303}, result.Groups[0].WinnerIDs, "在任者超过 TopN → 只保留综合分最高的 303")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 2, result.DisabledCount, "多活的另外两个在任者被收敛关闭")
	require.Equal(t, false, store.calls[301])
	require.Equal(t, false, store.calls[302])
	_, touched303 := store.calls[303]
	require.False(t, touched303, "当选的 303 本就开着，不产生调度写库")
}

// 回归守卫（2026-09-29 生产故障）：三个分组共享同一批账号，且开启全局健康锁定。
//
// 旧实现下，锁定把「当前开着的 3 个在任者」全部记为赢家；而 winner 是账号级、跨组取并集，
// 于是三个分组各显示 3 个开启，且永远关不掉——每轮再开一个组内 top1，旧的靠并集保住，只增不减。
// 修复后：锁定组只保留 TopN 个在任者 → 三个分组选出同一账号 → 并集只剩 1 个。
func TestGroupElectionSharedMembersConvergeToTopNUnderGlobalLock(t *testing.T) {
	members := make([]SupplierGroupSchedulingElectionMember, 0, 9)
	for _, groupID := range []int64{1, 2, 3} {
		for _, seed := range []struct {
			accountID int64
			healthy   int
		}{{101, 1}, {102, 5}, {103, 10}} {
			members = append(members, SupplierGroupSchedulingElectionMember{
				GroupID: groupID, AccountID: seed.accountID, Platform: "openai",
				Schedulable: true, LastTestStatus: "success",
				HealthyCount: seed.healthy, LastTestLatencyMs: 1000,
			})
		}
	}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: members}, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                       1,
		KeepHealthyIncumbentGlobal: true,
	}, time.Now())
	require.NoError(t, err)

	// 三个分组各自只认 103 → 账号级并集只有 103 开着，每组内可见的开启数都是 1。
	require.Len(t, result.Groups, 3)
	for _, group := range result.Groups {
		require.Equal(t, []int64{103}, group.WinnerIDs, "分组 %d 应收敛到唯一的 103", group.GroupID)
	}
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 2, result.DisabledCount, "多活的 101/102 被关闭")
	require.Equal(t, false, store.calls[101])
	require.Equal(t, false, store.calls[102])
	_, touched103 := store.calls[103]
	require.False(t, touched103, "当选的 103 本就开着，不产生调度写库")
}

// 回归守卫（2026-09-30 生产故障）：锁定分组已经开着「别组的在任者」，本轮绝不能再开本组 top1。
//
// 旧实现：在任者数一超过 TopN 就放弃锁定、退回正常择优；而正常择优取的是本组综合分前 N 名，
// 于是会在已经开着的账号之上**再加一个**。实测：gpt plus 组本来开着 446 / 445（都是别组在任者、
// 本组关不掉），本轮又开出本组 top1 361，变成 3 个；下一轮容量判断依然超，于是继续加开。
// 修复后：锁定组只保留最多 TopN 个在任者，且优先保留「已被其它分组选为赢家」的那些。
func TestGroupElectionKeepHealthyIncumbentDoesNotOpenWhenAlreadyServed(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// 分组 8 只有 446 → 先处理，446 先成为赢家。
		{GroupID: 8, AccountID: 446, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 5000},
		// 分组 59 开着 446（别组在任者）与 445；361 综合分高得多，但本组已经开够了账号。
		{GroupID: 59, AccountID: 446, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 5000},
		{GroupID: 59, AccountID: 445, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 2, LastTestLatencyMs: 5000},
		{GroupID: 59, AccountID: 361, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 30, LastTestLatencyMs: 100},
		// 分组 152 只有 445 → 445 由它保住。
		{GroupID: 152, AccountID: 445, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 2, LastTestLatencyMs: 5000},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                       1,
		KeepHealthyIncumbentGlobal: true,
	}, time.Now())
	require.NoError(t, err)

	_, opened := store.calls[361]
	require.False(t, opened, "本组已经开着 446/445，绝不能再开综合分更高的 361")
	require.Equal(t, []int64{446}, result.Groups[0].WinnerIDs, "分组 8 保留它的在任者 446")
	require.Equal(t, []int64{446}, result.Groups[1].WinnerIDs, "分组 59 优先保留被别组共用的 446，不再占第二个名额")
	require.Equal(t, []int64{445}, result.Groups[2].WinnerIDs, "分组 152 保留 445")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount, "446/445 都被各自分组保住，361 本就关着")
}

// 存量自愈：锁定分组已经累积出多活（3 个开着），其中两个被别的分组共用 →
// 只保留共用的那两个，只服务本组的那个被收敛关闭，并明确标成「开启数达上限」而不是「非分组最优」。
func TestGroupElectionKeepHealthyIncumbentConvergesAccumulatedStock(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 8, AccountID: 446, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 5000},
		{GroupID: 59, AccountID: 446, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 5000},
		{GroupID: 59, AccountID: 445, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 2, LastTestLatencyMs: 5000},
		{GroupID: 59, AccountID: 361, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 30, LastTestLatencyMs: 100},
		{GroupID: 152, AccountID: 445, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 2, LastTestLatencyMs: 5000},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                       1,
		KeepHealthyIncumbentGlobal: true,
	}, time.Now())
	require.NoError(t, err)

	// 361 是本组综合分最高的账号（健康次数 30），但本组已经开着被别组共用的 446/445，它必须让位：
	// 留着它，本组就是 3 个开启，突破「每组开启账号数」这个硬上限。
	require.Equal(t, []int64{446}, result.Groups[1].WinnerIDs, "分组 59 收敛到被共用的 446")
	require.Equal(t, false, store.calls[361], "只服务本组的 361 被收敛关闭")
	require.Equal(t, 1, result.DisabledCount)

	var item361 *SupplierGroupSchedulingElectionAccountItem
	for index := range result.Items {
		if result.Items[index].AccountID == 361 {
			item361 = &result.Items[index]
		}
	}
	require.NotNil(t, item361)
	require.Equal(t, SupplierGroupSchedulingElectionReasonOverCapacity, item361.Reason,
		"关闭原因必须说明是「开启数达上限」，否则运维会去查一个根本不存在的评分问题")

	// 明细要能把「被容量收敛」与「在任者健康锁定保留」区分开 —— 两者都带 Locked，但一个关一个留。
	var decision59 *SupplierGroupSchedulingElectionDecisionDetail
	for index := range item361.GroupDecisions {
		if item361.GroupDecisions[index].GroupID == 59 {
			decision59 = &item361.GroupDecisions[index]
		}
	}
	require.NotNil(t, decision59)
	require.True(t, decision59.Locked, "本组本轮确实没做择优")
	require.True(t, decision59.OverCapacity, "被容量收敛掉的成员必须单独标出来")
	require.False(t, decision59.Elected)
}

// 全局健康锁定 + 分组级反向覆盖：全局开则默认锁定所有分组，force-off 名单可单独排除，
// force-on 名单在全局关时单独锁定，且 force-off 优先于 force-on。
// 两个分组的在任者(501/601)都明显弱于挑战者(502/602)，锁定与否用「有没有换人」即可区分。
func TestGroupElectionKeepHealthyIncumbentGlobalAndOverrides(t *testing.T) {
	newMembers := func() []SupplierGroupSchedulingElectionMember {
		return []SupplierGroupSchedulingElectionMember{
			{GroupID: 1, AccountID: 501, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000},
			{GroupID: 1, AccountID: 502, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 100},
			{GroupID: 2, AccountID: 601, Platform: "openai", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000},
			{GroupID: 2, AccountID: 602, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 100},
		}
	}

	// 1) 全局锁定打开、无名单 → 两个分组都锁定，谁都不换。
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGlobal: true}, time.Now())
	require.NoError(t, err)
	require.Equal(t, []int64{501}, result.Groups[0].WinnerIDs, "全局锁定 → 分组 1 保留在任者 501")
	require.Equal(t, []int64{601}, result.Groups[1].WinnerIDs, "全局锁定 → 分组 2 保留在任者 601")
	require.Empty(t, store.calls, "全局锁定下不产生任何调度写库")

	// 2) 全局锁定打开、force-off 排除分组 2 → 分组 1 仍锁定、分组 2 恢复正常择优换人。
	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                                 1,
		KeepHealthyIncumbentGlobal:           true,
		KeepHealthyIncumbentExcludedGroupIDs: []int64{2},
	}, time.Now())
	require.NoError(t, err)
	_, touched501 := store.calls[501]
	_, touched502 := store.calls[502]
	require.False(t, touched501, "分组 1 仍锁定，在任者 501 不应被动")
	require.False(t, touched502, "分组 1 仍锁定，挑战者 502 不应被开启")
	require.Equal(t, true, store.calls[602], "分组 2 被 force-off 排除 → 正常择优开启更优的 602")
	require.Equal(t, false, store.calls[601], "分组 2 正常择优 → 落选的 601 被关闭")

	// 3) 全局锁定关闭、force-on 只锁分组 1 → 分组 1 锁定、分组 2 正常换人（等价于旧的 opt-in 行为）。
	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
	}, time.Now())
	require.NoError(t, err)
	_, touched501 = store.calls[501]
	require.False(t, touched501, "全局关、force-on 锁定分组 1 → 501 不应被动")
	require.Equal(t, true, store.calls[602], "全局关时分组 2 不在任何名单 → 正常择优换人")

	// 4) force-off 优先于 force-on：同一分组同时进两个名单时按「不锁定」处理。
	store = newFakeGroupElectionStore()
	svc = NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: newMembers()}, store)
	_, err = svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                                 1,
		KeepHealthyIncumbentGlobal:           true,
		KeepHealthyIncumbentGroupIDs:         []int64{1},
		KeepHealthyIncumbentExcludedGroupIDs: []int64{1},
	}, time.Now())
	require.NoError(t, err)
	require.Equal(t, true, store.calls[502], "force-off 优先 → 分组 1 不锁定，正常择优开启更优的 502")
	require.Equal(t, false, store.calls[501], "force-off 优先 → 分组 1 落选的 501 被关闭")
}

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

// 回归守卫（2026-10-01 生产故障）：必需模型支持者不能被「更快的同类支持者」顶掉。
//
// 生产现象：两个分组一小时对调两次 —— 每轮都是「上一轮的补选者被 over_capacity 关掉 +
// 另一个账号带 required_models 被补选开启」，且四条记录全带 locked:true。
// 根因：收敛按「综合分前 TopN」硬切，而补选跑在收敛之后、只增开 ——
// 上一轮的补选者下一轮必被当成「超出 TopN 的在任者」淘汰，补选再按「当轮最高分支持者」补一个；
// 而综合分里的用时分是组内 min-max 归一化的，延迟抖一下补选目标就换人。
// 修复：收敛保留集 = TopN 名额 + 必需模型支持者（额外保留、不占名额）。
//
// ⚠️ 场景必须让「在任支持者」不是「当轮最高分支持者」，否则单轮内「收敛关它 + 补选又开它」
// 会互相抵消，修复前后结果相同、测试没有区分力。这里 302 在任但分数低、303 关着但分数更高。
func TestGroupElectionRequiredModelSupporterSurvivesConvergence(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// 301：本组综合分最高的在任者，但不支持 aaa。
		{GroupID: 1, GroupName: "G1", AccountID: 301, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000, ModelMapping: map[string]any{"bbb": "bbb"}},
		// 302：上一轮为覆盖 aaa 被补选开启的支持者，综合分低。
		{GroupID: 1, GroupName: "G1", AccountID: 302, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 3000, ModelMapping: map[string]any{"aaa": "aaa"}},
		// 303：同样支持 aaa 且综合分更高，但当前关着 —— 修复前它会把在任的 302 顶掉。
		{GroupID: 1, GroupName: "G1", AccountID: 303, Platform: "anthropic", Schedulable: false, LastTestStatus: "success", HealthyCount: 10, LastTestLatencyMs: 1500, ModelMapping: map[string]any{"aaa": "aaa"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
		RequiredModelsByGroup:        map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	require.Empty(t, store.calls, "aaa 已由在任的 302 覆盖 → 既不该关 302，也不该开 303")
	require.ElementsMatch(t, []int64{301, 302}, result.Groups[0].WinnerIDs,
		"TopN 最优 301 + 必需模型支持者 302 都保留")
	require.Equal(t, 0, result.DisabledCount)
	require.Equal(t, 0, result.EnabledCount)
}

// 跨轮稳定性：对「每轮翻烙饼」的直接复刻 —— 连跑两轮都不该产生任何调度写库。
// 修复前第一轮就会「关 302、开 303」，第二轮再反向换一次（生产日志里每轮一关一开的来源）。
func TestGroupElectionRequiredModelSupporterStableAcrossRounds(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 301, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000, ModelMapping: map[string]any{"bbb": "bbb"}},
		{GroupID: 1, GroupName: "G1", AccountID: 302, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 3000, ModelMapping: map[string]any{"aaa": "aaa"}},
		{GroupID: 1, GroupName: "G1", AccountID: 303, Platform: "anthropic", Schedulable: false, LastTestStatus: "success", HealthyCount: 10, LastTestLatencyMs: 1500, ModelMapping: map[string]any{"aaa": "aaa"}},
	}
	config := SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
		RequiredModelsByGroup:        map[int64][]string{1: {"aaa"}},
	}

	firstStore := newFakeGroupElectionStore()
	first := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: members}, firstStore)
	_, err := first.Run(context.Background(), config, time.Now())
	require.NoError(t, err)
	require.Empty(t, firstStore.calls, "第一轮状态已稳定，不该换人")

	// 真实任务每轮都读回上一轮写下的 schedulable；本轮没有写库，状态不变，再跑一轮仍应无写库。
	secondStore := newFakeGroupElectionStore()
	second := NewSupplierGroupSchedulingElectionService(&fakeGroupElectionRepo{members: members}, secondStore)
	result, err := second.Run(context.Background(), config, time.Now())
	require.NoError(t, err)
	require.Empty(t, secondStore.calls, "第二轮同样不该换人")
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount)
}

// 支持者不会无限累积：同组有多个 aaa 支持者都在任时，只为未覆盖的模型保留综合分最高的一个。
// ⚠️ 这一条不区分修复前后（补选会把被关的那个再开回来），它守的是「全部保留」这种退化改法。
func TestGroupElectionRequiredModelKeepersDoNotOverKeep(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 301, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000, ModelMapping: map[string]any{"bbb": "bbb"}},
		{GroupID: 1, AccountID: 302, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000, ModelMapping: map[string]any{"aaa": "aaa"}},
		{GroupID: 1, AccountID: 303, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 3000, ModelMapping: map[string]any{"aaa": "aaa"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
		RequiredModelsByGroup:        map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	require.ElementsMatch(t, []int64{301, 302}, result.Groups[0].WinnerIDs,
		"aaa 只需一个支持者 → 保留综合分更高的 302，303 收敛关闭")
	require.Equal(t, false, store.calls[303])
	require.Equal(t, 1, result.DisabledCount)
}

// 支持必需模型的账号本身就是 TopN 最优时不额外保留任何人（保持原有收敛行为）。
func TestGroupElectionRequiredModelKeeperSkippedWhenTopNCovers(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 301, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000, ModelMapping: map[string]any{"aaa": "aaa"}},
		{GroupID: 1, AccountID: 302, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 1, LastTestLatencyMs: 3000, ModelMapping: map[string]any{"bbb": "bbb"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
		RequiredModelsByGroup:        map[int64][]string{1: {"aaa"}},
	}, time.Now())
	require.NoError(t, err)

	require.Equal(t, []int64{301}, result.Groups[0].WinnerIDs,
		"aaa 已由 TopN 最优 301 覆盖 → 不额外保留 302")
	require.Equal(t, false, store.calls[302], "302 应被容量收敛关闭")
	require.Equal(t, 1, result.DisabledCount)
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

// 演练模式（总开关）：照常算出谁该开谁该关、照常进明细与切换日志，但一个都不真写。
// 关键在 SchedulableAfter 必须保留期望值 —— 切换日志的取数条件就是 before <> after，
// 一旦像"写库失败"那样回滚成 before，演练就等于什么都没记。
func TestGroupElectionDryRunSkipsSchedulableWrite(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 10, Schedulable: true, LastTestStatus: "success", HealthyCount: 3},
		{GroupID: 1, GroupName: "G1", AccountID: 11, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, DryRun: true}, time.Now())
	require.NoError(t, err)

	require.Empty(t, store.calls, "演练模式不得调用 SetSchedulable")
	require.True(t, result.DryRun)
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount, "建议数不能混进真实生效的计数里")
	require.Equal(t, 1, result.SuggestedEnabledCount)
	require.Equal(t, 1, result.SuggestedDisabledCount)

	require.Len(t, result.Items, 2)
	require.Equal(t, int64(10), result.Items[0].AccountID)
	require.True(t, result.Items[0].Suggested)
	require.True(t, result.Items[0].SchedulableBefore)
	require.False(t, result.Items[0].SchedulableAfter, "落选的建议关：期望值要留在明细里，日志才取得到")

	require.Equal(t, int64(11), result.Items[1].AccountID)
	require.True(t, result.Items[1].Suggested)
	require.False(t, result.Items[1].SchedulableBefore)
	require.True(t, result.Items[1].SchedulableAfter, "当选的建议开：同上，回滚成 before 会让日志变空")
	require.Equal(t, SupplierGroupSchedulingElectionActionEnabled, result.Items[1].Action)
}

// 演练分组名单（opt-in）：只有名单内的分组只出建议，其余分组照常生效。
// 名单与总开关是并集，所以这里不能开总开关，否则分不出"名单真的起作用"。
func TestGroupElectionDryRunGroupIDsScopedToGroup(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 20, Schedulable: true, LastTestStatus: "success", HealthyCount: 1},
		{GroupID: 1, GroupName: "G1", AccountID: 21, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
		{GroupID: 2, GroupName: "G2", AccountID: 22, Schedulable: true, LastTestStatus: "success", HealthyCount: 1},
		{GroupID: 2, GroupName: "G2", AccountID: 23, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN: 1, DryRunGroupIDs: []int64{1},
	}, time.Now())
	require.NoError(t, err)

	require.True(t, result.DryRun, "配了演练分组名单就要在摘要里标出来")
	// G1 只出建议：20 建议关、21 建议开，都不写库。
	_, touched20 := store.calls[20]
	_, touched21 := store.calls[21]
	require.False(t, touched20)
	require.False(t, touched21)
	// G2 照常生效。
	require.Equal(t, false, store.calls[22])
	require.Equal(t, true, store.calls[23])

	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 1, result.DisabledCount)
	require.Equal(t, 1, result.SuggestedEnabledCount)
	require.Equal(t, 1, result.SuggestedDisabledCount)

	byID := make(map[int64]SupplierGroupSchedulingElectionAccountItem, len(result.Items))
	for _, item := range result.Items {
		byID[item.AccountID] = item
	}
	require.True(t, byID[20].Suggested)
	require.True(t, byID[21].Suggested)
	require.False(t, byID[22].Suggested, "不在演练名单里的分组是真的改了，不能标成建议")
	require.False(t, byID[23].Suggested)
}

// 演练仍要累计连续失败轮次：演练轮次若不计数，多轮演练永远看不到「达到阈值才会关」的那一天，
// 演练就退化成只能观察第一轮。
func TestGroupElectionDryRunStillPersistsFailedCount(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 30, Schedulable: true, LastTestStatus: "failed", HealthyCount: 0},
		{GroupID: 1, GroupName: "G1", AccountID: 31, Schedulable: false, LastTestStatus: "success", HealthyCount: 5},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, DryRun: true}, time.Now())
	require.NoError(t, err)

	require.Empty(t, store.calls, "演练模式下失败的账号也不该被真的关掉")
	require.Equal(t, 1, store.extraUpdates[30][supplierGroupElectionFailedCountExtraKey], "失败轮次照常记账")
	// 30 未达阈值保持原状（不产生建议），31 当选但只出建议。
	require.Equal(t, 1, result.SuggestedEnabledCount)
	require.Equal(t, 0, result.SuggestedDisabledCount)
	require.Equal(t, 1, result.PendingCount)
}

// 参选资格判据：last_test_status = success，或健康守护连续成功计数 > 0。
func TestGroupElectionMemberSelectablePredicate(t *testing.T) {
	require.True(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: "success"}))
	require.True(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: "failed", HealthyCount: 1}),
		"健康计数 > 0 的失败账号仍可参选")
	require.True(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: "", HealthyCount: 2}),
		"状态为空但健康计数 > 0 也可参选")
	require.False(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: "failed", HealthyCount: 0}))
	require.False(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: "", HealthyCount: 0}))
	require.True(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: " success ", HealthyCount: 0}),
		"状态两侧空白要归一后再比较")
}

// 候选集放宽：last_test_status = failed 但健康计数 > 0 的账号也能当选开启。
// 它是本组唯一账号；不放宽的话它会落到「失败 + 无备选」分支被原地留着，
// 结果是分组里开着的是个被判定失败的账号，而实际健康的账号反而开不出来。
func TestGroupElectionSelectsHealthyAccountDespiteFailedStatus(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 401, Schedulable: false, LastTestStatus: "failed", HealthyCount: 6},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[401], "健康计数 > 0 即具备参选资格，应当当选开启")
	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount)
	require.Equal(t, 0, result.PendingCount, "有资格的账号走健康分支，不该落进失败待观察")
	require.Len(t, result.Items, 1)
	require.Equal(t, SupplierGroupSchedulingElectionReasonElected, result.Items[0].Reason)
	// 分组统计仍按原始 last_test_status 归类，它不算 success —— 候选口径与统计口径是两回事。
	require.Equal(t, 0, result.Groups[0].SuccessCount)
	require.Equal(t, 1, result.Groups[0].FailedCount)
}

// 有参选资格的账号不累计「连续失败轮次」：它本轮按健康账号裁决，不是一次失败。
// 否则它一旦资格失效（健康计数归零），之前攒下的轮次会立刻把它推到阈值之上被关掉。
func TestGroupElectionSelectableAccountDoesNotChargeFailedCount(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 411, Schedulable: true, LastTestStatus: "failed", HealthyCount: 3, FailedCount: 1},
		{GroupID: 1, GroupName: "G1", AccountID: 412, Schedulable: false, LastTestStatus: "success", HealthyCount: 9},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	// 412 连续成功更多当选；411 有资格但落选 → 按"非分组最优"关闭，而不是走失败闸门。
	require.Equal(t, true, store.calls[412])
	require.Equal(t, false, store.calls[411])
	require.Equal(t, 0, store.extraUpdates[411][supplierGroupElectionFailedCountExtraKey],
		"有参选资格的账号失败轮次要清零，不能继续累计")
	require.Equal(t, 0, result.PendingCount)
	require.Equal(t, SupplierGroupSchedulingElectionReasonNotElected, result.Items[0].Reason)
}

// 锁定分组里的在任者具备参选资格（健康计数 > 0）但 last_test_status 是 failed 时，必须按健康在任者对待。
// 否则它拿不到赢家标记、进不了 scheduledHealthy，却会在裁决里走健康分支被判成"落选"关掉——
// 而锁定的全部意义就是不动它。
func TestGroupElectionKeepHealthyIncumbentTreatsSelectableFailedAsHealthy(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 421, Platform: "openai", Schedulable: true, LastTestStatus: "failed", HealthyCount: 4},
		{GroupID: 1, AccountID: 422, Platform: "openai", Schedulable: false, LastTestStatus: "success", HealthyCount: 30, LastTestLatencyMs: 50},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1, KeepHealthyIncumbentGroupIDs: []int64{1}}, time.Now())
	require.NoError(t, err)

	require.Empty(t, store.calls, "锁定分组不产生任何调度写库，在任者 421 不该被关")
	require.Equal(t, []int64{421}, result.Groups[0].WinnerIDs, "有资格的在任者按健康在任者记为赢家")
	require.Equal(t, 0, result.DisabledCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(421), result.Items[0].AccountID)
	require.True(t, result.Items[0].SchedulableAfter, "在任者 421 必须保持开着")
}

// 分组保底：一个全失败、当前又全部关着的分组，必须兜底开一个账号。
// 闸门一（noAlternative）只管"不关"，从不开人；没有这段逻辑，这种分组会永远停在 0 开启。
func TestGroupElectionKeepAliveOpensOneInAllFailedGroup(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 501, Platform: "openai", Schedulable: false, LastTestStatus: "failed", FailedCount: 0},
		{GroupID: 1, GroupName: "G1", AccountID: 502, Platform: "openai", Schedulable: false, LastTestStatus: "failed", FailedCount: 0},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[501], "全失败分组必须兜底开一个")
	require.NotContains(t, store.calls, int64(502), "保底只开一个，其余账号不该被顺带拨动")
	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 0, result.DisabledCount)
	require.Equal(t, []int64{501}, result.Groups[0].WinnerIDs, "保底当选也要记进分组明细")
	require.Equal(t, 1, result.Groups[0].WinnerCount)

	require.Len(t, result.Items, 2)
	require.Equal(t, int64(501), result.Items[0].AccountID)
	require.Equal(t, SupplierGroupSchedulingElectionReasonKeepAlive, result.Items[0].Reason)
	require.Equal(t, SupplierGroupSchedulingElectionActionEnabled, result.Items[0].Action)
	// 未入选的那个仍按闸门一"保持现状"，不该被保底顺带开启。
	require.Equal(t, int64(502), result.Items[1].AccountID)
	require.Equal(t, SupplierGroupSchedulingElectionReasonKeepLastOne, result.Items[1].Reason)
	require.Equal(t, 1, result.KeptCount)

	// 逐组依据要能区分"保底开启"与"择优入选"，否则日志看起来像择优算错了。
	decision := result.Items[0].GroupDecisions[0]
	require.True(t, decision.KeepAlive)
	require.True(t, decision.Elected)
}

// 保底账号不受失败闸门约束：连续失败已到阈值、且组内没有备选，它仍必须被开起来。
// 这条是最容易漏的一环——若不特判，它会先命中闸门一（noAlternative）并原样返回
// schedulableBefore = false，保底就成了空操作。
func TestGroupElectionKeepAliveOverridesFailureGates(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 511, Schedulable: false, LastTestStatus: "failed", FailedCount: 9},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[511], "已到失败阈值的账号仍要被保底开起来")
	require.Len(t, result.Items, 1)
	require.Equal(t, SupplierGroupSchedulingElectionReasonKeepAlive, result.Items[0].Reason)
	require.NotEqual(t, SupplierGroupSchedulingElectionReasonFailed, result.Items[0].Reason)
}

// 保底只挑一个：组内已有一个开启账号时，不得再顺带多开。
func TestGroupElectionKeepAliveSkippedWhenGroupAlreadyServed(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 521, Schedulable: true, LastTestStatus: "success", HealthyCount: 5},
		{GroupID: 1, GroupName: "G1", AccountID: 522, Schedulable: false, LastTestStatus: "failed", FailedCount: 0},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Empty(t, store.calls, "本组已有开启账号，不该产生任何拨动")
	require.Equal(t, []int64{521}, result.Groups[0].WinnerIDs)
	require.Equal(t, 0, result.EnabledCount)
	// 522 走失败闸门（未达阈值）保持原状，而不是被保底开启。
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(522), result.Items[0].AccountID)
	require.Equal(t, 1, result.PendingCount)
	require.False(t, result.Items[0].GroupDecisions[0].KeepAlive)
}

// 全未测过的分组同样要保底：这类账号默认会被"未测过不产生变更"跳过，
// 不特判就会落到 SkippedCount 里、永远开不起来。
func TestGroupElectionKeepAliveOpensUntestedAccount(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 531, Schedulable: false, LastTestStatus: ""},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[531], "全未测过的分组也必须兜底开一个")
	require.Equal(t, 0, result.SkippedCount, "保底账号不能被当作未测过而跳过")
	require.Equal(t, 1, result.EnabledCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, SupplierGroupSchedulingElectionReasonKeepAlive, result.Items[0].Reason)
}

// 保底候选必须完全确定：同一组失败轮次更少的账号优先，且不受输入顺序影响。
// 否则每轮换一个人开，日志上就是毫无理由地来回横跳。
func TestGroupElectionKeepAliveCandidateIsDeterministic(t *testing.T) {
	members := []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 541, Schedulable: false, LastTestStatus: "failed", FailedCount: 1},
		{GroupID: 1, GroupName: "G1", AccountID: 542, Schedulable: false, LastTestStatus: "failed", FailedCount: 0},
		{GroupID: 1, GroupName: "G1", AccountID: 543, Schedulable: false, LastTestStatus: "failed", FailedCount: 0},
	}
	// 正序与逆序各跑一次，结论必须一致。
	for _, input := range [][]SupplierGroupSchedulingElectionMember{members, {members[2], members[1], members[0]}} {
		repo := &fakeGroupElectionRepo{members: input}
		store := newFakeGroupElectionStore()
		svc := NewSupplierGroupSchedulingElectionService(repo, store)

		result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
		require.NoError(t, err)

		require.Equal(t, []int64{542}, result.Groups[0].WinnerIDs,
			"失败轮次最少（离阈值最远）者优先，同级按账号 ID 升序")
		require.Equal(t, true, store.calls[542])
	}
}

// 上游已不可用的账号一律失去参选资格，无论它的测试状态与健康计数多好看。
// 这两个字段在上游停用后是**冻结的旧数据**（健康守护的候选查询带 p.enabled = TRUE，不再刷新它们），
// 拿它们当"健康"用，等于让一个请求必然打不通的账号继续占着 TopN 名额。
func TestGroupElectionUpstreamUnavailableIsNotSelectable(t *testing.T) {
	require.False(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{LastTestStatus: "success", UpstreamUnavailable: true}),
		"测试状态 success 也不能抵消「上游已停用」")
	require.False(t, supplierGroupSchedulingElectionMemberSelectable(
		SupplierGroupSchedulingElectionMember{HealthyCount: 9, UpstreamUnavailable: true}),
		"健康计数 > 0 也不能抵消「上游已停用」")
}

// 回归守卫（2026-10-04 生产故障）：僵尸账号占着 TopN 名额，真健康的账号反而开不出来。
//
// 生产现象：分组 136 的 580（wahaha）测试状态与健康计数一直是 success/健康，
// 但它的上游账号所属供应商已停用 —— 健康守护自停用那刻起不再检查它，状态就此冻结。
// 而选举成员查询**不带任何供应商过滤**，于是它仍被当成健康在任者记为赢家，
// 挤掉了真正健康的 526，表现为「这个分组一个账号也没开启」。
// 修复：这类账号失去资格、不再进候选池、并被主动关闭（请求打过去只会失败）。
func TestGroupElectionUpstreamUnavailableClosesAndYieldsSeat(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// 580 综合分更高（健康次数多、延迟低），修复前它必胜 —— 这正是它有区分力的原因。
		{GroupID: 136, GroupName: "Kiro AWS", AccountID: 580, Platform: "anthropic", Schedulable: true,
			LastTestStatus: "success", HealthyCount: 30, LastTestLatencyMs: 100, UpstreamUnavailable: true},
		{GroupID: 136, GroupName: "Kiro AWS", AccountID: 526, Platform: "anthropic", Schedulable: false,
			LastTestStatus: "success", HealthyCount: 2, LastTestLatencyMs: 2000},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{136},
	}, time.Now())
	require.NoError(t, err)

	require.Equal(t, []int64{526}, result.Groups[0].WinnerIDs,
		"僵尸账号不占名额后，真健康的 526 才轮得到当选")
	// 用整表断言而不是逐个 store.calls[id]：map 取不存在的键返回 false，
	// 「要求它是 false」的断言会被"压根没写过"假通过。
	require.Equal(t, map[int64]bool{526: true, 580: false}, store.calls,
		"526 必须被开启，上游已停用的 580 必须被关闭 —— 两者都要真的落库")
	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 1, result.DisabledCount)

	items := map[int64]SupplierGroupSchedulingElectionAccountItem{}
	for _, item := range result.Items {
		items[item.AccountID] = item
	}
	require.Equal(t, SupplierGroupSchedulingElectionReasonUpstreamUnavailable, items[580].Reason,
		"关闭原因要说明是上游不可用，否则运维会去查一个并不存在的评分或容量问题")
	require.Equal(t, SupplierGroupSchedulingElectionActionDisabled, items[580].Action)
	require.True(t, items[580].GroupDecisions[0].UpstreamUnavailable,
		"逐组依据也要带上游停用标记：前端只按依据分类，缺了它只能显示「未参与择优」，等于没写清原因")
	require.Equal(t, SupplierGroupSchedulingElectionReasonElected, items[526].Reason)
}

// 上游已不可用的账号也不能被「分组保底」挑中：保底的本意是"别让分组空着"，
// 而把它开起来只会让这个分组在调度上看起来有账号、实际每个请求都失败 ——
// 比空着更难发现，因为监控上它是"有开启账号"的。
//
// 形状刻意与 TestGroupElectionKeepAliveOpensOneInAllFailedGroup 完全一致（唯一失败账号、关着、未到阈值），
// 只差 UpstreamUnavailable 这一个标志：那边必须开，这边必须不开。这样断言才有区分力。
func TestGroupElectionUpstreamUnavailableNotPickedByKeepAlive(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, GroupName: "G1", AccountID: 601, Schedulable: false,
			LastTestStatus: "failed", FailedCount: 0, UpstreamUnavailable: true},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)

	require.Empty(t, store.calls, "保底候选必须排除上游已停用的账号")
	require.Empty(t, result.Groups[0].WinnerIDs)
	require.Equal(t, 0, result.EnabledCount)
	require.Equal(t, 0, result.SkippedCount, "它是 failed，不该被当成「未测过」跳过")
	// 它本就关着、也没有任何 hold 或锁定标记，因此按既有规则不计入运行明细 ——
	// 这里如实断言，避免把「明细里没有它」误读成「本规则没生效」。
	require.Equal(t, 1, result.UnchangedCount)
	require.Empty(t, result.Items)
}

// 补选当选的原因必须写明「为覆盖哪个必需模型」，不能笼统写「分组内最优」。
//
// 生产现象：分组 143/155 每轮把 585 关掉、把 583 开起来。583 多半不是综合分前 N，
// 它只是因为 585 的 model_mapping 不含本组必需模型才被补选 ——
// 记成「分组内最优」，运维就会去翻评分找一个根本不存在的算法问题。
func TestGroupElectionRequiredModelSupplementReason(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// 585 综合分更高但在任，且不支持 claude-fable-5-1。
		{GroupID: 143, GroupName: "cc长期稳定", AccountID: 585, Platform: "anthropic", Schedulable: true,
			LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000, ModelMapping: map[string]any{"bbb": "bbb"}},
		// 583 综合分略低，但支持必需模型 → 只能补选它。
		{GroupID: 143, GroupName: "cc长期稳定", AccountID: 583, Platform: "anthropic", Schedulable: false,
			LastTestStatus: "success", HealthyCount: 18, LastTestLatencyMs: 1200, ModelMapping: map[string]any{"claude-fable-5-1": "claude-fable-5-1"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{143},
		RequiredModelsByGroup:        map[int64][]string{143: {"claude-fable-5-1"}},
	}, time.Now())
	require.NoError(t, err)

	require.Equal(t, true, store.calls[583], "必需模型只能由 583 提供 → 必须补选开启")
	require.ElementsMatch(t, []int64{585, 583}, result.Groups[0].WinnerIDs,
		"在任的 585 保持锁定不动，583 作为必需模型支持者额外开启")

	items := map[int64]SupplierGroupSchedulingElectionAccountItem{}
	for _, item := range result.Items {
		items[item.AccountID] = item
	}
	require.Equal(t, "分组必需模型 claude-fable-5-1 无在任账号支持，补选开启", items[583].Reason,
		"补选必须点名它覆盖的模型，否则日志读起来像打分算错了")
	require.Equal(t, SupplierGroupSchedulingElectionActionEnabled, items[583].Action)
	require.Equal(t, []string{"claude-fable-5-1"}, items[583].GroupDecisions[0].RequiredModels,
		"逐组依据里也要留下该模型，供切换日志展开")
}

// 收敛关闭的原因要能区分两种情形：单纯「开多了」与「本组必需模型已由保留的账号覆盖」。
//
// 生产现象：用户手动开启 585 后，下一轮又被收敛关掉、583 被保留，日志只说「超过上限」——
// 用户无法理解为什么留下的是 583。真实原因是 585 一个必需模型都不覆盖、而 583 覆盖，
// 它被关掉是**必需模型这条硬底线**决定的，不是评分问题。
func TestGroupElectionConvergenceReasonForNonCoveringAccount(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		// 583 综合分更高且覆盖必需模型 → 占住唯一的 TopN 名额。
		{GroupID: 143, GroupName: "cc长期稳定", AccountID: 583, Platform: "anthropic", Schedulable: true,
			LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000, ModelMapping: map[string]any{"claude-fable-5-1": "claude-fable-5-1"}},
		// 585 被人工开启，但不覆盖必需模型 → 收敛时必须让位。
		{GroupID: 143, GroupName: "cc长期稳定", AccountID: 585, Platform: "anthropic", Schedulable: true,
			LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000, ModelMapping: map[string]any{"bbb": "bbb"}},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{143},
		RequiredModelsByGroup:        map[int64][]string{143: {"claude-fable-5-1"}},
	}, time.Now())
	require.NoError(t, err)

	require.Equal(t, []int64{583}, result.Groups[0].WinnerIDs)
	// 整表断言：既要求 585 被真的关掉，也要求当选的 583 不产生任何调度写库（它本就开着）。
	require.Equal(t, map[int64]bool{585: false}, store.calls,
		"585 不覆盖必需模型 → 收敛关闭；583 本就开着 → 无写库")
	require.Equal(t, 1, result.DisabledCount)

	var item585 *SupplierGroupSchedulingElectionAccountItem
	for index := range result.Items {
		if result.Items[index].AccountID == 585 {
			item585 = &result.Items[index]
		}
	}
	require.NotNil(t, item585)
	require.Equal(t, "分组在任账号数超过上限，本组必需模型由保留账号覆盖，收敛关闭", item585.Reason,
		"必须点明「必需模型由保留账号覆盖」，否则用户只能看到「超过上限」，解释不了为什么留 583")

	decision := item585.GroupDecisions[0]
	require.True(t, decision.OverCapacity, "容量收敛标记照旧")
	require.True(t, decision.OverCapacityRequiredModel, "还要单独标出叠加了必需模型这一层原因")
	require.False(t, decision.Elected)
}

// 补选原因里的模型名必须去重且有序：requiredModelsByGroup 是 map，遍历顺序随机，
// 不去重排序会让同一状态每次刷新换个说法（"aaa、bbb" 与 "bbb、aaa"），日志就没法拿来对比。
func TestGroupElectionAccountRequiredModelsSortedDedup(t *testing.T) {
	account := &supplierGroupElectionAccount{requiredModelsByGroup: map[int64][]string{
		143: {"ccc", "aaa"},
		155: {"aaa", "bbb"}, // aaa 与 143 重复，跨组只应出现一次
	}}
	require.Equal(t, []string{"aaa", "bbb", "ccc"}, supplierGroupElectionAccountRequiredModels(account),
		"跨组去重 + 字典序，保证文案稳定")

	require.Nil(t, supplierGroupElectionAccountRequiredModels(nil))
	require.Nil(t, supplierGroupElectionAccountRequiredModels(&supplierGroupElectionAccount{}))
}

// 「本组配了必需模型」是收敛原因升级的前提：没配必需模型的分组不该出现新文案，
// 否则所有容量收敛都会被写成"必需模型由保留账号覆盖"，把真实原因掩盖掉。
func TestGroupElectionConvergenceReasonStaysPlainWithoutRequiredModels(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 301, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 20, LastTestLatencyMs: 1000},
		{GroupID: 1, AccountID: 302, Platform: "anthropic", Schedulable: true, LastTestStatus: "success", HealthyCount: 5, LastTestLatencyMs: 3000},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{
		TopN:                         1,
		KeepHealthyIncumbentGroupIDs: []int64{1},
	}, time.Now())
	require.NoError(t, err)

	var item302 *SupplierGroupSchedulingElectionAccountItem
	for index := range result.Items {
		if result.Items[index].AccountID == 302 {
			item302 = &result.Items[index]
		}
	}
	require.NotNil(t, item302)
	require.Equal(t, SupplierGroupSchedulingElectionReasonOverCapacity, item302.Reason)
	require.False(t, item302.GroupDecisions[0].OverCapacityRequiredModel,
		"没配必需模型时不得出现该标记")
}
