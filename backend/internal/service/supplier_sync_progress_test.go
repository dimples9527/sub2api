package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSupplierSyncProgressEmitsLifecycleEvents(t *testing.T) {
	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	SupplierSyncProgress(ctx, SupplierSyncProgressStagePrepare, "准备同步", nil)
	SupplierSyncProgressOK(ctx, SupplierSyncProgressStagePrepare, "准备完成")
	SupplierSyncProgressFail(ctx, SupplierSyncProgressStageCaptcha, errors.New("打码失败"))

	require.Len(t, events, 3)
	require.Equal(t, SupplierSyncProgressStagePrepare, events[0].Stage)
	require.Nil(t, events[0].OK)
	require.False(t, events[0].Time.IsZero())
	require.Equal(t, "准备完成", events[1].Message)
	require.NotNil(t, events[1].OK)
	require.True(t, *events[1].OK)
	require.Equal(t, SupplierSyncProgressStageCaptcha, events[2].Stage)
	require.NotNil(t, events[2].OK)
	require.False(t, *events[2].OK)
}

func TestSupplierSyncProgressWithoutObserverDoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		SupplierSyncProgress(context.Background(), SupplierSyncProgressStageDone, "完成", nil)
		SupplierSyncProgressOK(context.Background(), SupplierSyncProgressStageDone, "完成")
		SupplierSyncProgressFail(context.Background(), SupplierSyncProgressStageError, errors.New("失败"))
	})
}

func TestSupplierSyncProgressFailRedactsSensitiveValues(t *testing.T) {
	var event SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(value SupplierSyncProgressEvent) {
		event = value
	})

	SupplierSyncProgressFail(ctx, SupplierSyncProgressStageAccounts, errors.New(`upstream failed: {"apiKey":"secret-api-key","token":"secret-token"} Authorization: Bearer secret-bearer`))

	require.NotContains(t, event.Message, "secret-api-key")
	require.NotContains(t, event.Message, "secret-token")
	require.NotContains(t, event.Message, "secret-bearer")
	require.Contains(t, event.Message, "[已隐藏]")
}

func TestSupplierProviderSyncServiceEmitsProgressForAccountSync(t *testing.T) {
	provider := &SupplierProvider{
		ID:           42,
		Name:         "测试供应商",
		ProviderType: SupplierProviderTypeSub2API,
		Enabled:      true,
	}
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{provider}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{accounts: []SupplierProviderRemoteAccount{{Key: "key-1"}}}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	result, err := svc.SyncAccounts(ctx, provider.ID, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t,
		[]SupplierSyncProgressStage{
			SupplierSyncProgressStagePrepare,
			SupplierSyncProgressStagePrepare,
			SupplierSyncProgressStageSession,
			SupplierSyncProgressStageAccounts,
			SupplierSyncProgressStageAccounts,
			SupplierSyncProgressStagePersist,
			SupplierSyncProgressStagePersist,
			SupplierSyncProgressStageAccounts,
		},
		progressStages(events),
	)
}

func TestSupplierProviderSyncServiceCostRequestFailureDoesNotMarkPersistFailed(t *testing.T) {
	provider := &SupplierProvider{
		ID:                42,
		Name:              "测试供应商",
		ProviderType:      SupplierProviderTypeSub2API,
		Enabled:           true,
		PasswordEncrypted: "secret",
	}
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{provider}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{costErr: errors.New("成本接口不可用")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	result, err := svc.SyncCost(ctx, provider.ID, time.Now(), SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	var failedStages []SupplierSyncProgressStage
	for _, event := range events {
		if event.OK != nil && !*event.OK {
			failedStages = append(failedStages, event.Stage)
		}
	}
	require.Equal(t, []SupplierSyncProgressStage{SupplierSyncProgressStageCost}, failedStages)
}

func TestSupplierProviderSyncServiceCostPersistFailureDoesNotMarkCostFailed(t *testing.T) {
	provider := &SupplierProvider{
		ID:                42,
		Name:              "测试供应商",
		ProviderType:      SupplierProviderTypeSub2API,
		Enabled:           true,
		PasswordEncrypted: "secret",
	}
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{provider}}
	dataRepo := &supplierProviderDataRepoStub{costErr: errors.New("本地成本写入失败")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	result, err := svc.SyncCost(ctx, provider.ID, time.Now(), SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Equal(t, 1, dataRepo.costCalls)
	var failedStages []SupplierSyncProgressStage
	for _, event := range events {
		if event.OK != nil && !*event.OK {
			failedStages = append(failedStages, event.Stage)
		}
	}
	require.Equal(t, []SupplierSyncProgressStage{SupplierSyncProgressStagePersist}, failedStages)
}

func TestSupplierProviderSyncServiceDoesNotDuplicateStageFailureProgress(t *testing.T) {
	provider := &SupplierProvider{
		ID:           42,
		Name:         "test-provider",
		ProviderType: SupplierProviderTypeSub2API,
		Enabled:      true,
	}
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{provider}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{accountsErr: errors.New("accounts request failed")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	_, err := svc.SyncAccounts(ctx, provider.ID, SupplierSyncTriggerManual)
	require.Error(t, err)

	var failedStages []SupplierSyncProgressStage
	for _, event := range events {
		if event.OK != nil && !*event.OK {
			failedStages = append(failedStages, event.Stage)
		}
	}
	require.Equal(t, []SupplierSyncProgressStage{SupplierSyncProgressStageAccounts}, failedStages)
}

func TestSupplierProviderSyncServiceGroupStatusWriteFailureUsesPersistStage(t *testing.T) {
	provider := &SupplierProvider{
		ID:           42,
		Name:         "test-provider",
		ProviderType: SupplierProviderTypeSub2API,
		Enabled:      true,
	}
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{provider}}
	dataRepo := &supplierProviderDataRepoStub{groupStatusErr: errors.New("group status write failed")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	_, err := svc.SyncGroups(ctx, provider.ID, SupplierSyncTriggerManual)
	require.Error(t, err)

	var failedStages []SupplierSyncProgressStage
	for _, event := range events {
		if event.OK != nil && !*event.OK {
			failedStages = append(failedStages, event.Stage)
		}
	}
	require.Equal(t, []SupplierSyncProgressStage{SupplierSyncProgressStagePersist}, failedStages)
}

func TestSupplierProviderSyncServiceAllFinishFailureUsesPersistStage(t *testing.T) {
	provider := &SupplierProvider{
		ID:           42,
		Name:         "test-provider",
		ProviderType: SupplierProviderTypeSub2API,
		Enabled:      true,
	}
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{provider}}
	dataRepo := &supplierProviderDataRepoStub{finishErr: errors.New("sync run write failed")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	var events []SupplierSyncProgressEvent
	ctx := WithSupplierSyncProgressObserver(context.Background(), func(event SupplierSyncProgressEvent) {
		events = append(events, event)
	})

	_, err := svc.SyncAll(ctx, provider.ID, SupplierSyncTriggerManual)
	require.Error(t, err)

	var failedStages []SupplierSyncProgressStage
	for _, event := range events {
		if event.OK != nil && !*event.OK {
			failedStages = append(failedStages, event.Stage)
		}
	}
	require.Equal(t, []SupplierSyncProgressStage{SupplierSyncProgressStagePersist}, failedStages)
}

func progressStages(events []SupplierSyncProgressEvent) []SupplierSyncProgressStage {
	stages := make([]SupplierSyncProgressStage, 0, len(events))
	for _, event := range events {
		stages = append(stages, event.Stage)
	}
	return stages
}
