package service

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeGroupElectionRepo struct {
	members []SupplierGroupSchedulingElectionMember
}

func (f *fakeGroupElectionRepo) ListGroupSchedulingElectionMembers(ctx context.Context) ([]SupplierGroupSchedulingElectionMember, error) {
	return f.members, nil
}

type fakeGroupElectionStore struct {
	mu    sync.Mutex
	calls map[int64]bool
}

func newFakeGroupElectionStore() *fakeGroupElectionStore {
	return &fakeGroupElectionStore{calls: map[int64]bool{}}
}

func (f *fakeGroupElectionStore) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[id] = schedulable
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

// 单组单活：失败且开着的关掉；成功里连续成功最多的开启；成功落选的关掉；未测过的不动。
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

	// 10 失败→关；11 最优→开；12 成功但落选→关；13 未测→跳过（不写库）。
	require.Equal(t, false, store.calls[10])
	require.Equal(t, true, store.calls[11])
	require.Equal(t, false, store.calls[12])
	_, touched13 := store.calls[13]
	require.False(t, touched13, "未测过的账号不应被改动")

	require.Equal(t, 1, result.EnabledCount)
	require.Equal(t, 2, result.DisabledCount) // 10 + 12
	require.Equal(t, 1, result.SkippedCount)  // 13
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

// 全组无成功账号：失败开着的关掉，整组无赢家（可全灭）。
func TestGroupElectionAllFailedGoesDark(t *testing.T) {
	repo := &fakeGroupElectionRepo{members: []SupplierGroupSchedulingElectionMember{
		{GroupID: 1, AccountID: 51, Schedulable: true, LastTestStatus: "failed"},
		{GroupID: 1, AccountID: 52, Schedulable: true, LastTestStatus: "failed"},
	}}
	store := newFakeGroupElectionStore()
	svc := NewSupplierGroupSchedulingElectionService(repo, store)

	result, err := svc.Run(context.Background(), SupplierGroupSchedulingElectionConfig{TopN: 1}, time.Now())
	require.NoError(t, err)
	require.Equal(t, false, store.calls[51])
	require.Equal(t, false, store.calls[52])
	require.Equal(t, 2, result.DisabledCount)
	require.Equal(t, 0, result.EnabledCount)

	ids := make([]int64, 0, len(store.calls))
	for id := range store.calls {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	require.Equal(t, []int64{51, 52}, ids)
}
