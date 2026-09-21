package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 分组择优调度每轮都会回写「连续失败轮次」，它只是任务内部的记账，不参与调度快照。
// 一旦这个键掉出中性名单，每一轮择优都会触发调度 outbox 与调度桶重建——那是全量开销，
// 而任务本身每小时都可能跑一次，所以这条必须被钉住。
func TestGroupElectionFailedCountExtraIsSchedulerNeutral(t *testing.T) {
	require.True(t, isSchedulerNeutralExtraKey("supplier_group_election_failed_count"))
	require.False(t, shouldEnqueueSchedulerOutboxForExtraUpdates(map[string]any{
		"supplier_group_election_failed_count": 2,
	}))
}

func TestUpstreamBillingProbeExtraIsSchedulerNeutral(t *testing.T) {
	require.True(t, isSchedulerNeutralExtraKey("upstream_billing_probe"))
	require.True(t, isSchedulerNeutralExtraKey("upstream_billing_probe_enabled"))
	require.False(t, shouldEnqueueSchedulerOutboxForExtraUpdates(map[string]any{
		"upstream_billing_probe":         map[string]any{"status": "ok"},
		"upstream_billing_probe_enabled": true,
	}))
}
