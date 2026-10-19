package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// historyVisibilityRepoStub 只实现可见性相关的仓储方法，
// 其余方法由内嵌接口兜底 —— 这些用例不会走到它们。
type historyVisibilityRepoStub struct {
	ModelDegradationRepository
	hiddenGroups []int64
	hiddenHosts  []string
	hostStats    []ModelDegradationHistoryHostStat

	lastGroupID     int64
	lastGroupHidden bool
	lastHost        string
	lastHostHidden  bool
}

func (s *historyVisibilityRepoStub) ListHiddenHistoryTargets(context.Context) ([]int64, []string, error) {
	return s.hiddenGroups, s.hiddenHosts, nil
}

func (s *historyVisibilityRepoStub) SetHiddenHistoryGroup(_ context.Context, groupID int64, hidden bool) error {
	s.lastGroupID = groupID
	s.lastGroupHidden = hidden
	return nil
}

func (s *historyVisibilityRepoStub) SetHiddenHistoryHost(_ context.Context, host string, hidden bool) error {
	s.lastHost = host
	s.lastHostHidden = hidden
	return nil
}

func (s *historyVisibilityRepoStub) ListHistoryHostStats(context.Context, int) ([]ModelDegradationHistoryHostStat, error) {
	return s.hostStats, nil
}

func TestNormalizeHistoryHost(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "裸主机名", raw: "api.foo.com", want: "api.foo.com", ok: true},
		{name: "完整地址去 scheme 与端口", raw: "https://API.Foo.com:8443/v1", want: "api.foo.com", ok: true},
		{name: "任意 scheme 泛化剥离", raw: "ftp://api.foo.com", want: "api.foo.com", ok: true},
		{name: "带路径", raw: " api.foo.com/path ", want: "api.foo.com", ok: true},
		{name: "带查询串", raw: "api.foo.com?x=1", want: "api.foo.com", ok: true},
		{name: "空串", raw: "   ", ok: false},
		{name: "带空格", raw: "api foo.com", ok: false},
		{name: "带用户信息", raw: "https://user@api.foo.com", ok: false},
		{name: "只剩 scheme", raw: "https://", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := normalizeHistoryHost(tc.raw)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSetHistoryGroupHiddenValidation(t *testing.T) {
	stub := &historyVisibilityRepoStub{}
	svc := NewModelDegradationService(stub)

	require.ErrorIs(t, svc.SetHistoryGroupHidden(context.Background(), 0, true), ErrModelDegradationHistoryGroupInvalid)
	require.ErrorIs(t, svc.SetHistoryGroupHidden(context.Background(), -3, false), ErrModelDegradationHistoryGroupInvalid)

	require.NoError(t, svc.SetHistoryGroupHidden(context.Background(), 12, true))
	require.Equal(t, int64(12), stub.lastGroupID)
	require.True(t, stub.lastGroupHidden)

	require.NoError(t, svc.SetHistoryGroupHidden(context.Background(), 12, false))
	require.False(t, stub.lastGroupHidden)
}

func TestSetHistoryHostHiddenNormalizes(t *testing.T) {
	stub := &historyVisibilityRepoStub{}
	svc := NewModelDegradationService(stub)

	require.ErrorIs(t, svc.SetHistoryHostHidden(context.Background(), "https://", true), ErrModelDegradationHistoryHostInvalid)
	require.ErrorIs(t, svc.SetHistoryHostHidden(context.Background(), "api foo.com", true), ErrModelDegradationHistoryHostInvalid)

	require.NoError(t, svc.SetHistoryHostHidden(context.Background(), "https://API.Foo.com:8443/v1", true))
	require.Equal(t, "api.foo.com", stub.lastHost)
	require.True(t, stub.lastHostHidden)
}

func TestHistoryVisibilityReturnsEmptySlices(t *testing.T) {
	svc := NewModelDegradationService(&historyVisibilityRepoStub{})

	config, err := svc.HistoryVisibility(context.Background())
	require.NoError(t, err)
	require.NotNil(t, config)
	require.Equal(t, []int64{}, config.HiddenGroupIDs)
	require.Equal(t, []string{}, config.HiddenHosts)
	require.Equal(t, []ModelDegradationHistoryHostStat{}, config.Hosts)
}
