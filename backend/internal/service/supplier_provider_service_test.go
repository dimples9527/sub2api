package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierProviderRepoStub struct {
	items []*SupplierProvider
	next  int64
}

type supplierProviderTypeRepoStub struct {
	items []*SupplierProviderType
	next  int64
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
