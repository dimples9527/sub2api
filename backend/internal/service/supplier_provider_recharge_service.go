package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SupplierProviderRecharge struct {
	ID            int64           `json:"id"`
	ProviderID    int64           `json:"provider_id"`
	ProviderName  string          `json:"provider_name"`
	ProviderType  string          `json:"provider_type"`
	ExternalID    string          `json:"external_id"`
	ExternalCode  string          `json:"external_code"`
	RechargeType  string          `json:"recharge_type"`
	Amount        float64         `json:"amount"`
	Status        string          `json:"status"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Description   string          `json:"description"`
	SourcePayload json.RawMessage `json:"source_payload"`
	SyncedAt      time.Time       `json:"synced_at"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SupplierProviderRechargeListParams struct {
	ProviderID int64
	Start      time.Time
	End        time.Time
	Page       int
	PageSize   int
}

type SupplierProviderRechargeListResult struct {
	Items       []SupplierProviderRecharge `json:"items"`
	Total       int64                      `json:"total"`
	TotalAmount float64                    `json:"total_amount"`
	Page        int                        `json:"page"`
	PageSize    int                        `json:"page_size"`
}

type SupplierProviderRechargeRepository interface {
	Upsert(ctx context.Context, providerID int64, records []SupplierProviderRechargeRecord) error
	List(ctx context.Context, params SupplierProviderRechargeListParams) (SupplierProviderRechargeListResult, error)
	Sum(ctx context.Context, providerID int64, start, end time.Time) (float64, error)
	HasRecords(ctx context.Context, providerID int64) (bool, error)
}

type SupplierProviderRechargeSyncParams struct {
	ProviderID int64 `json:"provider_id"`
	FullSync   bool  `json:"full_sync"`
}

type SupplierProviderRechargeSyncResult struct {
	ProviderID   int64     `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	RecordCount  int       `json:"record_count"`
	SyncedAt     time.Time `json:"synced_at"`
}

type SupplierProviderRechargeSyncAllResult struct {
	Items        []SupplierProviderRechargeSyncResult `json:"items"`
	SuccessCount int                                  `json:"success_count"`
	FailedCount  int                                  `json:"failed_count"`
}

type SupplierProviderRechargeService struct {
	providerRepo SupplierProviderRepository
	rechargeRepo SupplierProviderRechargeRepository
	remote       SupplierProviderRemoteRechargeHistoryClient
	encryptor    SecretEncryptor
}

func NewSupplierProviderRechargeService(providerRepo SupplierProviderRepository, rechargeRepo SupplierProviderRechargeRepository, remote SupplierProviderRemoteRechargeHistoryClient, encryptor SecretEncryptor) *SupplierProviderRechargeService {
	return &SupplierProviderRechargeService{providerRepo: providerRepo, rechargeRepo: rechargeRepo, remote: remote, encryptor: encryptor}
}

func (s *SupplierProviderRechargeService) providerPassword(provider *SupplierProvider) string {
	stored := strings.TrimSpace(provider.PasswordEncrypted)
	if stored == "" || s.encryptor == nil {
		return stored
	}
	password, err := s.encryptor.Decrypt(stored)
	if err != nil {
		return stored
	}
	return password
}

func (s *SupplierProviderRechargeService) List(ctx context.Context, params SupplierProviderRechargeListParams) (SupplierProviderRechargeListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 100
	}
	result, err := s.rechargeRepo.List(ctx, params)
	if err != nil {
		return SupplierProviderRechargeListResult{}, fmt.Errorf("list supplier provider recharges: %w", err)
	}
	return result, nil
}

func (s *SupplierProviderRechargeService) Sum(ctx context.Context, providerID int64, start, end time.Time) (float64, error) {
	return s.rechargeRepo.Sum(ctx, providerID, start, end)
}

func (s *SupplierProviderRechargeService) Sync(ctx context.Context, params SupplierProviderRechargeSyncParams) (SupplierProviderRechargeSyncResult, error) {
	provider, err := s.providerRepo.GetByID(ctx, params.ProviderID)
	if err != nil {
		return SupplierProviderRechargeSyncResult{ProviderID: params.ProviderID, Status: SupplierSyncStatusFailed, Message: err.Error()}, err
	}
	result := SupplierProviderRechargeSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, SyncedAt: time.Now(), Status: SupplierSyncStatusSuccess}
	if s.remote == nil {
		err = fmt.Errorf("supplier recharge remote client is unavailable")
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
		return result, err
	}
	password := s.providerPassword(provider)
	end := time.Now()
	start := end.AddDate(0, 0, -7)
	if params.FullSync {
		start = time.Unix(0, 0)
	} else if has, hasErr := s.rechargeRepo.HasRecords(ctx, provider.ID); hasErr != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = hasErr.Error()
		return result, hasErr
	} else if !has {
		start = time.Unix(0, 0)
	}
	records, err := s.remote.FetchRechargeRecords(ctx, provider, password, start, end)
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
		return result, err
	}
	if err = s.rechargeRepo.Upsert(ctx, provider.ID, records); err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
		return result, err
	}
	result.RecordCount = len(records)
	result.Message = fmt.Sprintf("已同步 %d 条充值记录", len(records))
	return result, nil
}

func (s *SupplierProviderRechargeService) SyncAll(ctx context.Context, fullSync bool) (SupplierProviderRechargeSyncAllResult, error) {
	enabled := true
	providers, _, err := s.providerRepo.List(ctx, SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 200})
	if err != nil {
		return SupplierProviderRechargeSyncAllResult{}, err
	}
	result := SupplierProviderRechargeSyncAllResult{Items: make([]SupplierProviderRechargeSyncResult, 0, len(providers))}
	for _, provider := range providers {
		item, syncErr := s.Sync(ctx, SupplierProviderRechargeSyncParams{ProviderID: provider.ID, FullSync: fullSync})
		result.Items = append(result.Items, item)
		if syncErr != nil {
			result.FailedCount++
		} else {
			result.SuccessCount++
		}
	}
	return result, nil
}
