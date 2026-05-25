//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type tempUnscheduleRepoStub struct {
	accountRepoStub
	calls []tempUnscheduleCall
}

type tempUnscheduleCall struct {
	id     int64
	until  time.Time
	reason string
}

func (r *tempUnscheduleRepoStub) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	r.calls = append(r.calls, tempUnscheduleCall{id: id, until: until, reason: reason})
	return nil
}

func TestGatewayServiceTempUnscheduleRetryableError_BadGatewayOnlyTempUnschedulesEmptyResponse(t *testing.T) {
	t.Run("plain_bad_gateway_does_not_temp_unschedule", func(t *testing.T) {
		repo := &tempUnscheduleRepoStub{}
		svc := &GatewayService{accountRepo: repo}

		svc.TempUnscheduleRetryableError(context.Background(), 132, &UpstreamFailoverError{
			StatusCode:             http.StatusBadGateway,
			ResponseBody:           []byte(`{"error":{"message":"upstream bad gateway"}}`),
			RetryableOnSameAccount: true,
		})

		require.Empty(t, repo.calls)
	})

	t.Run("empty_response_bad_gateway_temp_unschedules", func(t *testing.T) {
		repo := &tempUnscheduleRepoStub{}
		svc := &GatewayService{accountRepo: repo}

		svc.TempUnscheduleRetryableError(context.Background(), 132, &UpstreamFailoverError{
			StatusCode:             http.StatusBadGateway,
			ResponseBody:           []byte(`{"error":"empty stream response from upstream"}`),
			RetryableOnSameAccount: true,
		})

		require.Len(t, repo.calls, 1)
		require.Equal(t, int64(132), repo.calls[0].id)
		require.Contains(t, repo.calls[0].reason, "empty stream response")
		require.WithinDuration(t, time.Now().Add(time.Minute), repo.calls[0].until, 5*time.Second)
	})
}
