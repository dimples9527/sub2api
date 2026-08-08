package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type monitorGroupPlatformOverrideRepoStub struct {
	listResult map[int64]string
	listErr    error
	setGroupID int64
	setPlatform string
	setErr     error
	clearGroupID int64
	clearErr   error
}

func (s *monitorGroupPlatformOverrideRepoStub) ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	result := make(map[int64]string, len(s.listResult))
	for k, v := range s.listResult {
		result[k] = v
	}
	return result, nil
}

func (s *monitorGroupPlatformOverrideRepoStub) Set(ctx context.Context, groupID int64, platform string) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.setGroupID = groupID
	s.setPlatform = platform
	return nil
}

func (s *monitorGroupPlatformOverrideRepoStub) Clear(ctx context.Context, groupID int64) error {
	if s.clearErr != nil {
		return s.clearErr
	}
	s.clearGroupID = groupID
	return nil
}

func TestMonitorGroupPlatformOverrideServiceSetNormalizesAndValidatesPlatform(t *testing.T) {
	repo := &monitorGroupPlatformOverrideRepoStub{}
	svc := NewMonitorGroupPlatformOverrideService(repo)

	require.NoError(t, svc.Set(context.Background(), 12, " OpenAI "))
	require.Equal(t, int64(12), repo.setGroupID)
	require.Equal(t, PlatformOpenAI, repo.setPlatform)

	err := svc.Set(context.Background(), 12, "unsupported")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported monitor group platform")
	require.Equal(t, PlatformOpenAI, repo.setPlatform, "unsupported platform should not overwrite the last valid write")
}

func TestMonitorGroupPlatformOverrideServiceListAndClearDelegate(t *testing.T) {
	repo := &monitorGroupPlatformOverrideRepoStub{listResult: map[int64]string{3: PlatformGemini}}
	svc := NewMonitorGroupPlatformOverrideService(repo)

	loaded, err := svc.ListByGroupIDs(context.Background(), []int64{3})
	require.NoError(t, err)
	require.Equal(t, map[int64]string{3: PlatformGemini}, loaded)

	require.NoError(t, svc.Clear(context.Background(), 3))
	require.Equal(t, int64(3), repo.clearGroupID)
}

func TestMonitorGroupPlatformOverrideServiceRejectsInvalidGroupID(t *testing.T) {
	svc := NewMonitorGroupPlatformOverrideService(&monitorGroupPlatformOverrideRepoStub{})
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "set", err: svc.Set(context.Background(), 0, PlatformOpenAI)},
		{name: "clear", err: svc.Clear(context.Background(), -1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.err)
			require.Contains(t, tc.err.Error(), "group id must be positive")
		})
	}
}

func TestMonitorGroupPlatformOverrideServiceErrorsWhenRepoMissing(t *testing.T) {
	var svc MonitorGroupPlatformOverrideService = &monitorGroupPlatformOverrideService{}
	require.Error(t, svc.Set(context.Background(), 1, PlatformOpenAI))
	require.Error(t, svc.Clear(context.Background(), 1))
	loaded, err := svc.ListByGroupIDs(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Empty(t, loaded)
}
