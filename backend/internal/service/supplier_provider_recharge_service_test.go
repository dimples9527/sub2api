package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rechargeServiceRemoteStub struct {
	calls         []int64
	passwords     []string
	errByProvider map[int64]error
}

func (s *rechargeServiceRemoteStub) FetchRechargeRecords(_ context.Context, provider *SupplierProvider, password string, _, _ time.Time) ([]SupplierProviderRechargeRecord, error) {
	s.calls = append(s.calls, provider.ID)
	s.passwords = append(s.passwords, password)
	if err := s.errByProvider[provider.ID]; err != nil {
		return nil, err
	}
	return []SupplierProviderRechargeRecord{{
		ExternalID: "recharge-1",
		Amount:     1,
		OccurredAt: time.Now(),
	}}, nil
}

type rechargeServiceRepoStub struct {
	upsertCalls []int64
}

func (s *rechargeServiceRepoStub) Upsert(_ context.Context, providerID int64, _ []SupplierProviderRechargeRecord) error {
	s.upsertCalls = append(s.upsertCalls, providerID)
	return nil
}

func (s *rechargeServiceRepoStub) List(context.Context, SupplierProviderRechargeListParams) (SupplierProviderRechargeListResult, error) {
	return SupplierProviderRechargeListResult{}, nil
}

func (s *rechargeServiceRepoStub) Sum(context.Context, int64, time.Time, time.Time) (float64, error) {
	return 0, nil
}

func (s *rechargeServiceRepoStub) HasRecords(context.Context, int64) (bool, error) {
	return false, nil
}

type rechargeServiceEncryptorStub struct {
	decryptErr error
}

func (s rechargeServiceEncryptorStub) Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}

func (s rechargeServiceEncryptorStub) Decrypt(string) (string, error) {
	return "", s.decryptErr
}

func TestSupplierProviderRechargeServiceSyncFallsBackToStoredPlaintextPassword(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:                11,
		Name:              "TKAPI2",
		Enabled:           true,
		PasswordEncrypted: "plain-password",
	}}}
	remote := &rechargeServiceRemoteStub{}
	rechargeRepo := &rechargeServiceRepoStub{}
	service := NewSupplierProviderRechargeService(providerRepo, rechargeRepo, remote, rechargeServiceEncryptorStub{
		decryptErr: errors.New("decode base64: illegal base64 data at input byte 8"),
	})

	result, err := service.Sync(context.Background(), SupplierProviderRechargeSyncParams{ProviderID: 11, FullSync: true})

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []string{"plain-password"}, remote.passwords)
}

func TestSupplierProviderRechargeServiceSyncAllOnlySyncsEnabledProviders(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{
		{ID: 1, Name: "enabled-provider", Enabled: true},
		{ID: 2, Name: "disabled-provider", Enabled: false},
	}}
	remote := &rechargeServiceRemoteStub{errByProvider: map[int64]error{2: errors.New("disabled provider should not be called")}}
	rechargeRepo := &rechargeServiceRepoStub{}
	service := NewSupplierProviderRechargeService(providerRepo, rechargeRepo, remote, nil)

	result, err := service.SyncAll(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Zero(t, result.FailedCount)
	require.Equal(t, []int64{1}, remote.calls)
	require.Equal(t, []int64{1}, rechargeRepo.upsertCalls)
}

func TestSupplierProviderRechargeServiceSyncAllContinuesAfterEnabledProviderFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{
		{ID: 1, Name: "failed-provider", Enabled: true},
		{ID: 2, Name: "successful-provider", Enabled: true},
	}}
	remote := &rechargeServiceRemoteStub{errByProvider: map[int64]error{1: errors.New("upstream unavailable")}}
	rechargeRepo := &rechargeServiceRepoStub{}
	service := NewSupplierProviderRechargeService(providerRepo, rechargeRepo, remote, nil)

	result, err := service.SyncAll(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 1, result.FailedCount)
	require.Equal(t, []int64{1, 2}, remote.calls)
	require.Equal(t, []int64{2}, rechargeRepo.upsertCalls)
	require.Equal(t, SupplierSyncStatusFailed, result.Items[0].Status)
	require.Contains(t, result.Items[0].Message, "upstream unavailable")
}
