package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierProviderRepoStub struct {
	items                        []*SupplierProvider
	next                         int64
	costTrends                   []SupplierProviderCostTrendPoint
	costBreakdowns               []SupplierProviderCostBreakdown
	balanceSummaryDays           []SupplierProviderBalanceSummaryDay
	balanceCosts                 []SupplierProviderBalanceCostDay
	costTrendCalls               int
	disableAfterAuthFailureCalls int
	disableMessages              []string
}

type supplierProviderTypeRepoStub struct {
	items []*SupplierProviderType
	next  int64
}

type supplierCostDeviationThresholdStub struct {
	threshold float64
}

func (s supplierCostDeviationThresholdStub) SupplierCostDeviationThreshold(context.Context) float64 {
	return s.threshold
}

type supplierCostSourceResolverStub struct {
	resolution SupplierCostSourceResolution
}

func (s supplierCostSourceResolverStub) ResolveCostSource(context.Context, int64) (SupplierCostSourceResolution, error) {
	return s.resolution, nil
}

func (r *supplierProviderRepoStub) List(_ context.Context, params SupplierProviderListParams) ([]*SupplierProvider, int64, error) {
	matched := r.filtered(params)
	start := (params.Page - 1) * params.PageSize
	if start < 0 {
		start = 0
	}
	if start > len(matched) {
		start = len(matched)
	}
	end := start + params.PageSize
	if params.PageSize <= 0 || end > len(matched) {
		end = len(matched)
	}
	out := make([]*SupplierProvider, 0, end-start)
	for _, item := range matched[start:end] {
		clone := *item
		out = append(out, &clone)
	}
	return out, int64(len(matched)), nil
}

func (r *supplierProviderRepoStub) ListCostTrends(_ context.Context, start, end time.Time, providerID int64) ([]SupplierProviderCostTrendPoint, error) {
	r.costTrendCalls++
	if r.costTrends != nil {
		return r.costTrends, nil
	}
	return []SupplierProviderCostTrendPoint{}, nil
}

func (r *supplierProviderRepoStub) ListCostBreakdowns(_ context.Context, start, end time.Time, providerID int64) ([]SupplierProviderCostBreakdown, error) {
	if r.costBreakdowns != nil {
		return r.costBreakdowns, nil
	}
	return []SupplierProviderCostBreakdown{}, nil
}

func (r *supplierProviderRepoStub) ListBalanceSummaryDays(_ context.Context) ([]SupplierProviderBalanceSummaryDay, error) {
	if r.balanceSummaryDays != nil {
		return r.balanceSummaryDays, nil
	}
	return []SupplierProviderBalanceSummaryDay{}, nil
}

func (r *supplierProviderRepoStub) ListBalanceCosts(_ context.Context, _, _ time.Time, _ int64) ([]SupplierProviderBalanceCostDay, error) {
	if r.balanceCosts != nil {
		return r.balanceCosts, nil
	}
	return []SupplierProviderBalanceCostDay{}, nil
}

func (r *supplierProviderRepoStub) Summary(_ context.Context, params SupplierProviderListParams) (SupplierProviderSummary, error) {
	items := r.filtered(params)
	summary := SupplierProviderSummary{TotalCount: int64(len(items))}
	for _, item := range items {
		if item.Enabled {
			summary.EnabledCount++
		}
		if item.RiskLevel == "high" || item.RiskLevel == "critical" {
			summary.HighRiskCount++
		}
		if item.EstimatedDays != nil && *item.EstimatedDays < 3 {
			summary.LowBalanceCount++
		}
		if item.SyncStatus == "failed" {
			summary.SyncFailureCount++
		}
		summary.RateRiskCount += item.RateRiskCount
	}
	return summary, nil
}

func (r *supplierProviderRepoStub) filtered(params SupplierProviderListParams) []*SupplierProvider {
	out := make([]*SupplierProvider, 0, len(r.items))
	for _, item := range r.items {
		if params.Enabled != nil && item.Enabled != *params.Enabled {
			continue
		}
		clone := *item
		out = append(out, &clone)
	}
	return out
}
func (r *supplierProviderRepoStub) GetByID(_ context.Context, id int64) (*SupplierProvider, error) {
	for _, item := range r.items {
		if item.ID == id {
			clone := *item
			return &clone, nil
		}
	}
	return nil, ErrSupplierProviderNotFound
}
func (r *supplierProviderRepoStub) Create(_ context.Context, item *SupplierProvider) error {
	r.next++
	item.ID = r.next
	if len(r.items) == 0 || item.IsDefault {
		for _, existing := range r.items {
			existing.IsDefault = false
		}
		item.IsDefault = true
	}
	clone := *item
	r.items = append(r.items, &clone)
	return nil
}
func (r *supplierProviderRepoStub) Update(_ context.Context, item *SupplierProvider) error {
	for index := range r.items {
		if r.items[index].ID == item.ID {
			if item.IsDefault {
				for _, existing := range r.items {
					existing.IsDefault = false
				}
			}
			clone := *item
			r.items[index] = &clone
			return nil
		}
	}
	return ErrSupplierProviderNotFound
}

func (r *supplierProviderRepoStub) DisableAfterAuthFailure(_ context.Context, providerID int64, message string, syncedAt time.Time) error {
	for _, item := range r.items {
		if item.ID != providerID {
			continue
		}
		item.Enabled = false
		item.SyncStatus = SupplierSyncStatusFailed
		item.SyncMessage = message
		item.LastSyncAt = &syncedAt
		r.disableAfterAuthFailureCalls++
		r.disableMessages = append(r.disableMessages, message)
		return nil
	}
	return ErrSupplierProviderNotFound
}
func (r *supplierProviderRepoStub) Delete(context.Context, int64) error { return nil }
func (r *supplierProviderRepoStub) SetDefault(ctx context.Context, id int64) (*SupplierProvider, error) {
	for _, item := range r.items {
		item.IsDefault = item.ID == id
	}
	return r.GetByID(ctx, id)
}

func (r *supplierProviderTypeRepoStub) List(_ context.Context, enabledOnly bool) ([]*SupplierProviderType, error) {
	out := make([]*SupplierProviderType, 0, len(r.items))
	for _, item := range r.items {
		if enabledOnly && !item.Enabled {
			continue
		}
		clone := *item
		out = append(out, &clone)
	}
	return out, nil
}

func (r *supplierProviderTypeRepoStub) GetByID(_ context.Context, id int64) (*SupplierProviderType, error) {
	for _, item := range r.items {
		if item.ID == id {
			clone := *item
			return &clone, nil
		}
	}
	return nil, ErrSupplierProviderTypeNotFound
}

func (r *supplierProviderTypeRepoStub) GetByCode(_ context.Context, code string) (*SupplierProviderType, error) {
	for _, item := range r.items {
		if item.Code == code {
			clone := *item
			return &clone, nil
		}
	}
	return nil, ErrSupplierProviderTypeNotFound
}

func (r *supplierProviderTypeRepoStub) Create(_ context.Context, item *SupplierProviderType) error {
	for _, existing := range r.items {
		if existing.Code == item.Code {
			return ErrSupplierProviderTypeExists
		}
	}
	r.next++
	item.ID = r.next
	clone := *item
	r.items = append(r.items, &clone)
	return nil
}

func (r *supplierProviderTypeRepoStub) Update(_ context.Context, item *SupplierProviderType) error {
	for _, existing := range r.items {
		if existing.Code == item.Code && existing.ID != item.ID {
			return ErrSupplierProviderTypeExists
		}
	}
	for index := range r.items {
		if r.items[index].ID == item.ID {
			clone := *item
			r.items[index] = &clone
			return nil
		}
	}
	return ErrSupplierProviderTypeNotFound
}

func (r *supplierProviderTypeRepoStub) Delete(_ context.Context, id int64) error {
	for index := range r.items {
		if r.items[index].ID == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return nil
		}
	}
	return ErrSupplierProviderTypeNotFound
}

type supplierEncryptorStub struct{}

func (supplierEncryptorStub) Encrypt(value string) (string, error) { return "encrypted:" + value, nil }
func (supplierEncryptorStub) Decrypt(value string) (string, error) { return value, nil }

func validSupplierProviderParams() SupplierProviderUpsertParams {
	return SupplierProviderUpsertParams{
		Code:                       "primary",
		Name:                       "主供应商",
		ProviderType:               "sub2api",
		BaseURL:                    "https://supplier.example.com",
		Password:                   "secret",
		AccountRateMultiplierScale: 1,
		Enabled:                    true,
	}
}

func TestSupplierProviderServiceCreateStoresPlainAndRedactsCredential(t *testing.T) {
	repo := &supplierProviderRepoStub{}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	created, err := service.Create(context.Background(), validSupplierProviderParams())
	require.NoError(t, err)
	require.True(t, created.CredentialConfigured)
	require.Empty(t, created.PasswordEncrypted)
	require.Equal(t, "secret", repo.items[0].PasswordEncrypted)
}

func TestSupplierProviderServiceUpdateStoresPlainCredential(t *testing.T) {
	repo := &supplierProviderRepoStub{next: 1, items: []*SupplierProvider{{ID: 1, Code: "primary", PasswordEncrypted: "old-secret"}}}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	params := validSupplierProviderParams()
	params.Password = "new-secret"

	_, err := service.Update(context.Background(), 1, params)

	require.NoError(t, err)
	require.Equal(t, "new-secret", repo.items[0].PasswordEncrypted)
}

func TestSupplierProviderServiceCreateSub2APIClearsUsername(t *testing.T) {
	repo := &supplierProviderRepoStub{}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	params := validSupplierProviderParams()
	params.Email = " owner@example.com "
	params.Username = " stale-login@example.com "

	_, err := service.Create(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, "owner@example.com", repo.items[0].Email)
	require.Empty(t, repo.items[0].Username)
}

func TestSupplierProviderServiceStoresNewAPIAuthModeAndRejectsUnknownMode(t *testing.T) {
	repo := &supplierProviderRepoStub{}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	params := validSupplierProviderParams()
	params.ProviderType = SupplierProviderTypeNewAPI
	params.Username = "root"
	params.NewAPIAuthMode = " cookie_session "

	created, err := service.Create(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, SupplierNewAPIAuthModeCookieSession, created.NewAPIAuthMode)
	require.Equal(t, SupplierNewAPIAuthModeCookieSession, repo.items[0].NewAPIAuthMode)

	params.Code = "invalid-newapi-auth-mode"
	params.NewAPIAuthMode = "unsupported"
	_, err = service.Create(context.Background(), params)

	require.ErrorIs(t, err, ErrSupplierProviderInvalid)
}

func TestSupplierProviderServiceUpdateKeepsCredentialWhenPasswordBlank(t *testing.T) {
	repo := &supplierProviderRepoStub{next: 1, items: []*SupplierProvider{{ID: 1, Code: "primary", PasswordEncrypted: "encrypted:old"}}}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	params := validSupplierProviderParams()
	params.Password = ""
	params.Name = "更新名称"
	updated, err := service.Update(context.Background(), 1, params)
	require.NoError(t, err)
	require.Equal(t, "更新名称", updated.Name)
	require.Equal(t, "encrypted:old", repo.items[0].PasswordEncrypted)
}

func TestSupplierProviderServiceListBuildsSummaryAndRedacts(t *testing.T) {
	days := 1.5
	repo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Enabled: true, RiskLevel: "high", EstimatedDays: &days, SyncStatus: "failed", RateRiskCount: 2, PasswordEncrypted: "cipher"}}}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	result, err := service.List(context.Background(), SupplierProviderListParams{})
	require.NoError(t, err)
	require.Equal(t, SupplierProviderSummary{TotalCount: 1, EnabledCount: 1, HighRiskCount: 1, LowBalanceCount: 1, SyncFailureCount: 1, RateRiskCount: 2}, result.Summary)
	require.True(t, result.Items[0].CredentialConfigured)
	require.Empty(t, result.Items[0].PasswordEncrypted)
}

func TestSupplierProviderServiceListSummaryUsesAllMatchedRows(t *testing.T) {
	days := 1.5
	repo := &supplierProviderRepoStub{items: []*SupplierProvider{
		{ID: 1, Enabled: true, RiskLevel: "normal", PasswordEncrypted: "cipher-1"},
		{ID: 2, Enabled: true, RiskLevel: "critical", EstimatedDays: &days, SyncStatus: "failed", RateRiskCount: 3, PasswordEncrypted: "cipher-2"},
	}}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	result, err := service.List(context.Background(), SupplierProviderListParams{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, SupplierProviderSummary{TotalCount: 2, EnabledCount: 2, HighRiskCount: 1, LowBalanceCount: 1, SyncFailureCount: 1, RateRiskCount: 3}, result.Summary)
}

func TestSupplierProviderServiceGetRedactsCredential(t *testing.T) {
	repo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Code: "primary", PasswordEncrypted: "cipher"}}}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	provider, err := service.Get(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, provider.CredentialConfigured)
	require.Empty(t, provider.PasswordEncrypted)
}

func TestSupplierProviderServiceAllowsOnlyOneDefaultProvider(t *testing.T) {
	repo := &supplierProviderRepoStub{}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})

	firstParams := validSupplierProviderParams()
	first, err := service.Create(context.Background(), firstParams)
	require.NoError(t, err)
	require.True(t, first.IsDefault)

	secondParams := validSupplierProviderParams()
	secondParams.Code = "secondary"
	secondParams.Name = "备用供应商"
	secondParams.BaseURL = "https://secondary.example.com"
	secondParams.IsDefault = true
	second, err := service.Create(context.Background(), secondParams)
	require.NoError(t, err)
	require.True(t, second.IsDefault)
	require.False(t, repo.items[0].IsDefault)
	require.True(t, repo.items[1].IsDefault)

	provider, err := service.SetDefault(context.Background(), first.ID)
	require.NoError(t, err)
	require.True(t, provider.IsDefault)
	require.True(t, repo.items[0].IsDefault)
	require.False(t, repo.items[1].IsDefault)
}

func validSupplierProviderTypeParams() SupplierProviderTypeUpsertParams {
	return SupplierProviderTypeUpsertParams{
		Code:               "sub2api",
		Name:               "Sub2API",
		LoginURL:           "https://template.example.com/api/v1/auth/login",
		APIKeysURL:         "https://template.example.com/api/admin/keys",
		GroupsURL:          "https://template.example.com/api/admin/groups",
		AvailableGroupsURL: "https://template.example.com/api/admin/available-groups",
		BalanceURL:         "https://template.example.com/api/admin/balance",
		UsageCostURL:       "https://template.example.com/api/admin/usage-cost",
		Enabled:            true,
		SortOrder:          10,
	}
}

func TestSupplierProviderTypeServiceCreateListUpdateDelete(t *testing.T) {
	repo := &supplierProviderTypeRepoStub{}
	service := NewSupplierProviderTypeService(repo)

	created, err := service.Create(context.Background(), validSupplierProviderTypeParams())
	require.NoError(t, err)
	require.Equal(t, "sub2api", created.Code)
	require.Equal(t, "Sub2API", created.Name)

	items, err := service.List(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, items, 1)

	params := validSupplierProviderTypeParams()
	params.Name = "Sub2API 企业版"
	params.Enabled = false
	updated, err := service.Update(context.Background(), created.ID, params)
	require.NoError(t, err)
	require.Equal(t, "Sub2API 企业版", updated.Name)
	require.False(t, updated.Enabled)

	enabledItems, err := service.List(context.Background(), true)
	require.NoError(t, err)
	require.Empty(t, enabledItems)

	require.NoError(t, service.Delete(context.Background(), created.ID))
	_, err = service.Get(context.Background(), created.ID)
	require.ErrorIs(t, err, ErrSupplierProviderTypeNotFound)
}

func TestSupplierProviderTypeServiceRejectsInvalidURL(t *testing.T) {
	service := NewSupplierProviderTypeService(&supplierProviderTypeRepoStub{})
	params := validSupplierProviderTypeParams()
	params.BalanceURL = "ftp://invalid.example.com/balance"
	_, err := service.Create(context.Background(), params)
	require.ErrorIs(t, err, ErrSupplierProviderTypeInvalid)
}

func TestSupplierProviderTypeServiceAllowsRelativeEndpointTemplates(t *testing.T) {
	repo := &supplierProviderTypeRepoStub{}
	service := NewSupplierProviderTypeService(repo)
	params := validSupplierProviderTypeParams()
	params.LoginURL = "/api/v1/auth/login"
	params.APIKeysURL = "/api/admin/keys"
	params.GroupsURL = "/api/admin/groups"
	params.AvailableGroupsURL = "/api/admin/available-groups"
	params.BalanceURL = "/api/admin/balance"
	params.UsageCostURL = "/api/admin/usage-cost"

	created, err := service.Create(context.Background(), params)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/auth/login", created.LoginURL)
	require.Equal(t, "/api/admin/usage-cost", created.UsageCostURL)
}

func TestSupplierProviderTypeServiceUsesGroupsURLForAvailableGroups(t *testing.T) {
	repo := &supplierProviderTypeRepoStub{}
	service := NewSupplierProviderTypeService(repo)
	params := validSupplierProviderTypeParams()
	params.GroupsURL = "/api/admin/groups"
	params.AvailableGroupsURL = "/api/admin/other-groups"

	created, err := service.Create(context.Background(), params)
	require.NoError(t, err)
	require.Equal(t, "/api/admin/groups", created.GroupsURL)
	require.Equal(t, "/api/admin/groups", created.AvailableGroupsURL)
}

func TestSupplierProviderServiceAppliesTypeTemplateForBlankEndpoints(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{}
	typeRepo := &supplierProviderTypeRepoStub{items: []*SupplierProviderType{{
		ID:                 1,
		Code:               "sub2api",
		Name:               "Sub2API",
		LoginURL:           "https://template.example.com/login",
		APIKeysURL:         "https://template.example.com/keys",
		GroupsURL:          "https://template.example.com/groups",
		AvailableGroupsURL: "https://template.example.com/available-groups",
		BalanceURL:         "https://template.example.com/balance",
		UsageCostURL:       "https://template.example.com/cost",
		Enabled:            true,
	}}}
	service := NewSupplierProviderService(providerRepo, supplierEncryptorStub{}, typeRepo)

	params := validSupplierProviderParams()
	params.LoginURL = "https://provider.example.com/custom-login"
	created, err := service.Create(context.Background(), params)
	require.NoError(t, err)
	require.Equal(t, "https://provider.example.com/custom-login", created.LoginURL)
	require.Equal(t, "https://template.example.com/keys", created.APIKeysURL)
	require.Equal(t, "https://template.example.com/groups", created.GroupsURL)
	require.Equal(t, "https://template.example.com/groups", created.AvailableGroupsURL)
	require.Equal(t, "https://template.example.com/balance", created.BalanceURL)
	require.Equal(t, "https://template.example.com/cost", created.UsageCostURL)
}

func TestSupplierProviderServiceUsesGroupsURLForAvailableGroups(t *testing.T) {
	repo := &supplierProviderRepoStub{}
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	params := validSupplierProviderParams()
	params.GroupsURL = "/api/admin/groups"
	params.AvailableGroupsURL = "/api/admin/other-groups"

	created, err := service.Create(context.Background(), params)
	require.NoError(t, err)
	require.Equal(t, "/api/admin/groups", created.GroupsURL)
	require.Equal(t, "/api/admin/groups", created.AvailableGroupsURL)
}

func TestSupplierProviderServiceListCostTrendsFillsMissingDays(t *testing.T) {
	repo := &supplierProviderRepoStub{costTrends: []SupplierProviderCostTrendPoint{
		{Date: "2026-07-28", UpstreamCost: 12.5, LocalCost: 10, EffectiveCost: 10},
		{Date: "2026-07-30", UpstreamCost: 8, LocalCost: 9.5, EffectiveCost: 8},
	}}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})

	// 依赖运行时 Today，只校验返回长度与日期非空。
	result, err := svc.ListCostTrends(context.Background(), 3, 0)
	require.NoError(t, err)
	require.Equal(t, 3, result.Days)
	require.Len(t, result.Points, 3)
	require.NotEmpty(t, result.StartDate)
	require.NotEmpty(t, result.EndDate)
	for _, point := range result.Points {
		require.NotEmpty(t, point.Date)
	}
}

func TestSupplierProviderServiceListCostTrendsByDateRange(t *testing.T) {
	repo := &supplierProviderRepoStub{costTrends: []SupplierProviderCostTrendPoint{
		{Date: "2026-07-10", UpstreamCost: 1, LocalCost: 2, EffectiveCost: 1},
		{Date: "2026-07-12", UpstreamCost: 3, LocalCost: 4, EffectiveCost: 3},
	}}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-07-10", "2026-07-12", 7)
	require.NoError(t, err)
	require.Equal(t, 3, result.Days)
	require.Equal(t, "2026-07-10", result.StartDate)
	require.Equal(t, "2026-07-12", result.EndDate)
	require.Equal(t, int64(7), result.ProviderID)
	require.Len(t, result.Points, 3)
	require.Equal(t, "2026-07-10", result.Points[0].Date)
	require.Equal(t, 1.0, result.Points[0].UpstreamCost)
	require.Equal(t, 1.0, result.Points[0].EffectiveCost)
	require.Equal(t, "2026-07-11", result.Points[1].Date)
	require.Equal(t, 0.0, result.Points[1].UpstreamCost)
	require.Equal(t, 0.0, result.Points[1].EffectiveCost)
	require.Equal(t, "2026-07-12", result.Points[2].Date)
	require.Equal(t, 3.0, result.Points[2].UpstreamCost)
	require.Equal(t, 3.0, result.Points[2].EffectiveCost)
}

func TestSupplierProviderServiceListCostTrendsIncludesCostBreakdown(t *testing.T) {
	repo := &supplierProviderRepoStub{costBreakdowns: []SupplierProviderCostBreakdown{{
		ProviderID:    7,
		ProviderName:  "主供应商",
		ProviderType:  "sub2api",
		UpstreamCost:  42.5,
		LocalCost:     17.25,
		EffectiveCost: 17.25,
	}}}
	// 阈值设为 1.0 禁用偏差提示，聚焦拆分成本本身的返回。
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 1.0})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-07-10", "2026-07-12", 0)
	require.NoError(t, err)
	require.Len(t, result.Breakdown, 1)
	require.Equal(t, int64(7), result.Breakdown[0].ProviderID)
	require.Equal(t, "主供应商", result.Breakdown[0].ProviderName)
	require.Equal(t, 42.5, result.Breakdown[0].UpstreamCost)
	require.Equal(t, 17.25, result.Breakdown[0].LocalCost)
	require.Equal(t, 17.25, result.Breakdown[0].EffectiveCost)

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(payload, &body))
	require.Contains(t, body, "breakdown")
}

func TestSupplierProviderServiceListCostTrendsFillsEffectiveCostFromBalanceFallback(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	repo := &supplierProviderRepoStub{
		costTrends: []SupplierProviderCostTrendPoint{
			{Date: "2026-07-10", UpstreamCost: 1, LocalCost: 2, EffectiveCost: 1},
			{Date: "2026-07-11", UpstreamCost: 0, LocalCost: 0, EffectiveCost: 0},
		},
		costBreakdowns: []SupplierProviderCostBreakdown{
			{ProviderID: 7, ProviderName: "甲", ProviderType: "sub2api", UpstreamCost: 42.5, LocalCost: 17.25, EffectiveCost: 42.5},
			{ProviderID: 8, ProviderName: "乙", ProviderType: "sub2api", UpstreamCost: 0, LocalCost: 0, EffectiveCost: 0},
		},
		balanceCosts: []SupplierProviderBalanceCostDay{
			{Date: "2026-07-11", ProviderID: 7, BalanceCost: 5},
			{Date: "2026-07-11", ProviderID: 8, BalanceCost: 7},
			{Date: "2026-07-12", ProviderID: 7, BalanceCost: 3},
			{Date: "2026-07-10", ProviderID: 7, BalanceCost: 99},
		},
	}
	// 阈值设为 1.0 禁用偏差提示，聚焦余额保底填充本身。
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 1.0})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-07-10", "2026-07-12", 0)
	require.NoError(t, err)
	require.Equal(t, 3, result.Days)

	// 已有真实成本的日期不被余额保底覆盖。
	require.Equal(t, 1.0, result.Points[0].UpstreamCost)
	require.Equal(t, 1.0, result.Points[0].EffectiveCost)
	// 生效成本缺失的日期按当天各供应商余额差额之和填充；
	// 余额差不是上游口径，上游成本保持缺失，不被伪造。
	require.Equal(t, 12.0, result.Points[1].EffectiveCost)
	require.Zero(t, result.Points[1].UpstreamCost)
	// 只有余额快照、没有 daily_stats 的日期被补上。
	require.Equal(t, 3.0, result.Points[2].EffectiveCost)
	require.Zero(t, result.Points[2].UpstreamCost)

	// 拆分中已有真实成本的供应商不被覆盖。
	require.Equal(t, 42.5, result.Breakdown[0].UpstreamCost)
	require.Equal(t, 42.5, result.Breakdown[0].EffectiveCost)
	// 拆分同理：只补生效成本。
	require.Equal(t, 7.0, result.Breakdown[1].EffectiveCost)
	require.Zero(t, result.Breakdown[1].UpstreamCost)
}

func TestSupplierProviderServiceListCostTrendsWarnsWithoutOverridingWhenDeviationExceedsThreshold(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	// 偏差基准必须是计算成本：同步时正是它顶替上游成本成为生效成本。
	// 这里本地成本贴着上游（10% 偏差），只有计算成本偏离，所以按本地判定会漏报。
	repo := &supplierProviderRepoStub{
		costTrends: []SupplierProviderCostTrendPoint{
			{Date: "2026-08-10", UpstreamCost: 100, CalculatedCost: 10, LocalCost: 90, EffectiveCost: 10},
		},
		costBreakdowns: []SupplierProviderCostBreakdown{
			{ProviderID: 7, ProviderName: "甲", ProviderType: "sub2api", UpstreamCost: 100, CalculatedCost: 10, LocalCost: 90, EffectiveCost: 10},
		},
	}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-08-10", "2026-08-10", 0)
	require.NoError(t, err)

	// 偏差 90% > 50%：只记录提示，四个口径的数值都保持原样。
	require.Equal(t, 100.0, result.Points[0].UpstreamCost)
	require.Equal(t, 10.0, result.Points[0].CalculatedCost)
	require.Equal(t, 90.0, result.Points[0].LocalCost)
	require.Equal(t, 10.0, result.Points[0].EffectiveCost)
	require.Contains(t, result.Points[0].Warning, "生效成本已取计算成本")
	require.Equal(t, 100.0, result.Breakdown[0].UpstreamCost)
	require.Equal(t, 10.0, result.Breakdown[0].CalculatedCost)
	require.Equal(t, 10.0, result.Breakdown[0].EffectiveCost)
	require.Contains(t, result.Breakdown[0].CostWarning, "生效成本已取计算成本")
}

func TestSupplierProviderServiceListCostTrendsKeepsSyncWarningInsteadOfRecomputing(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	// 同步时写入的逐日提示带着余额差与充值明细，比展示层重算的更具体，不能被覆盖。
	syncWarning := "上游成本 100.00 与计算成本 10.00 偏差 90%，生效成本已取计算成本（余额差 8.00 + 充值 2.00）"
	repo := &supplierProviderRepoStub{
		costTrends: []SupplierProviderCostTrendPoint{
			{Date: "2026-08-14", UpstreamCost: 100, CalculatedCost: 10, EffectiveCost: 10, Warning: syncWarning},
		},
		costBreakdowns: []SupplierProviderCostBreakdown{
			{ProviderID: 7, ProviderName: "甲", ProviderType: "sub2api", UpstreamCost: 100, CalculatedCost: 10, EffectiveCost: 10, CostWarning: syncWarning},
		},
	}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-08-14", "2026-08-14", 0)
	require.NoError(t, err)

	require.Equal(t, syncWarning, result.Points[0].Warning)
	require.Equal(t, syncWarning, result.Breakdown[0].CostWarning)
}

func TestSupplierProviderServiceListCostTrendsDoesNotWarnWithinThreshold(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	// 本地成本远离上游不该触发提示：它不参与生效成本的取值。
	repo := &supplierProviderRepoStub{
		costTrends: []SupplierProviderCostTrendPoint{
			{Date: "2026-08-11", UpstreamCost: 100, CalculatedCost: 60, LocalCost: 5, EffectiveCost: 100},
		},
		costBreakdowns: []SupplierProviderCostBreakdown{
			{ProviderID: 7, ProviderName: "甲", ProviderType: "sub2api", UpstreamCost: 100, CalculatedCost: 60, LocalCost: 5, EffectiveCost: 100},
		},
	}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-08-11", "2026-08-11", 0)
	require.NoError(t, err)

	// 上游与计算成本偏差 40% <= 50%，不提示。
	require.Equal(t, 100.0, result.Points[0].UpstreamCost)
	require.Empty(t, result.Points[0].Warning)
	require.Equal(t, 100.0, result.Breakdown[0].UpstreamCost)
	require.Empty(t, result.Breakdown[0].CostWarning)
}

func TestSupplierProviderServiceListCostTrendsDoesNotWarnWithoutCalculatedCost(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	repo := &supplierProviderRepoStub{
		costTrends: []SupplierProviderCostTrendPoint{
			{Date: "2026-08-12", UpstreamCost: 100, CalculatedCost: 0, LocalCost: 10, EffectiveCost: 100},
		},
		costBreakdowns: []SupplierProviderCostBreakdown{
			{ProviderID: 7, ProviderName: "甲", ProviderType: "sub2api", UpstreamCost: 100, CalculatedCost: 0, LocalCost: 10, EffectiveCost: 100},
		},
	}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-08-12", "2026-08-12", 0)
	require.NoError(t, err)

	require.Equal(t, 100.0, result.Points[0].UpstreamCost)
	require.Empty(t, result.Points[0].Warning)
	require.Equal(t, 100.0, result.Breakdown[0].UpstreamCost)
	require.Empty(t, result.Breakdown[0].CostWarning)
}

func TestSupplierProviderServiceListCostTrendsKeepsUpstreamForCalculatedSource(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	// 计算成本优先：today_cost 写的是计算成本，raw_upstream_cost 仍是真实上游值。
	repo := &supplierProviderRepoStub{
		costTrends: []SupplierProviderCostTrendPoint{
			{Date: "2026-08-13", UpstreamCost: 100, CalculatedCost: 10, LocalCost: 12, EffectiveCost: 10},
		},
		costBreakdowns: []SupplierProviderCostBreakdown{
			{ProviderID: 7, ProviderName: "甲", ProviderType: "sub2api", UpstreamCost: 100, CalculatedCost: 10, LocalCost: 12, EffectiveCost: 10},
		},
	}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})
	svc.SetCostSourceResolver(supplierCostSourceResolverStub{resolution: SupplierCostSourceResolution{
		Source:    SupplierCostSourceCalculated,
		Threshold: 0.5,
	}})

	result, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-08-13", "2026-08-13", 7)
	require.NoError(t, err)

	// 上游成本保持真实值，不再被改写成计算成本。
	require.Equal(t, 100.0, result.Points[0].UpstreamCost)
	require.Equal(t, 10.0, result.Points[0].CalculatedCost)
	require.Equal(t, 10.0, result.Points[0].EffectiveCost)
	// 用户主动选定的稳定状态，不逐日刷提示。
	require.Empty(t, result.Points[0].Warning)
	require.Equal(t, 100.0, result.Breakdown[0].UpstreamCost)
	require.Equal(t, 10.0, result.Breakdown[0].EffectiveCost)
	require.Empty(t, result.Breakdown[0].CostWarning)
}

func TestSupplierProviderServiceListCostTrendsByDateRangeRejectsInvalid(t *testing.T) {
	svc := NewSupplierProviderService(&supplierProviderRepoStub{}, supplierEncryptorStub{})

	_, err := svc.ListCostTrendsByDateRange(context.Background(), "bad", "2026-07-12", 0)
	require.Error(t, err)

	_, err = svc.ListCostTrendsByDateRange(context.Background(), "2026-07-12", "2026-07-10", 0)
	require.Error(t, err)
}

func TestSupplierProviderServiceGetBalanceSummary(t *testing.T) {
	repo := &supplierProviderRepoStub{balanceSummaryDays: []SupplierProviderBalanceSummaryDay{
		{Date: "2026-08-01", Balance: 100, Cost: 10},
		{Date: "2026-08-02", Balance: 120, Cost: 12},
		{Date: "2026-08-03", Balance: 90, Cost: 15},
	}}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})

	summary, err := svc.GetBalanceSummary(context.Background())
	require.NoError(t, err)
	require.Equal(t, "2026-08-03", summary.LatestDate)
	require.Equal(t, "2026-08-03", summary.Today.Date)
	require.Equal(t, 90.0, summary.Today.Balance)
	require.Equal(t, 15.0, summary.Today.Cost)
	require.Equal(t, "2026-08-02", summary.Previous.Date)
	require.Equal(t, 120.0, summary.Previous.Balance)
	require.Equal(t, "2026-08-01", summary.History.FirstDate)
	require.Equal(t, 3, summary.History.Days)
	require.Equal(t, 310.0, summary.History.TotalBalance)
	require.Equal(t, 37.0, summary.History.TotalCost)
}

func TestSupplierProviderServiceGetBalanceSummaryEmpty(t *testing.T) {
	svc := NewSupplierProviderService(&supplierProviderRepoStub{}, supplierEncryptorStub{})

	summary, err := svc.GetBalanceSummary(context.Background())
	require.NoError(t, err)
	require.Empty(t, summary.LatestDate)
	require.Empty(t, summary.Today.Date)
	require.Empty(t, summary.Previous.Date)
	require.Equal(t, 0, summary.History.Days)
}

func TestSupplierProviderTypeServiceAcceptsRechargeURLTemplate(t *testing.T) {
	repo := &supplierProviderTypeRepoStub{}
	service := NewSupplierProviderTypeService(repo)
	params := validSupplierProviderTypeParams()
	params.RechargeURL = "/api/log/self?p={page}&page_size={page_size}&type=1&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}"

	created, err := service.Create(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, params.RechargeURL, created.RechargeURL)
}

func TestSupplierProviderServiceAppliesRechargeURLFromTypeTemplate(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{}
	typeRepo := &supplierProviderTypeRepoStub{items: []*SupplierProviderType{{
		ID:          1,
		Code:        "sub2api",
		Name:        "Sub2API",
		RechargeURL: "/api/v1/redeem/history?timezone=Asia%2FShanghai",
		Enabled:     true,
	}}}
	service := NewSupplierProviderService(providerRepo, supplierEncryptorStub{}, typeRepo)

	created, err := service.Create(context.Background(), validSupplierProviderParams())

	require.NoError(t, err)
	require.Equal(t, "/api/v1/redeem/history?timezone=Asia%2FShanghai", created.RechargeURL)
}
