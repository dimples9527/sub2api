# Upstream Provider Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first phase of the independent upstream management module: provider configuration CRUD, Sub2API/NewAPI login tests, and normalized key retrieval.

**Architecture:** Store provider configs as a single JSON setting through `SettingRepository` to avoid migrations. Put provider business logic in new `service/upstream_provider*.go` files with one adapter per upstream type. Add a new admin handler, route group, frontend API module, and one admin page; existing files only get minimal wiring.

**Tech Stack:** Go/Gin service and handler tests, Vue 3 + TypeScript frontend, existing `apiClient`, existing admin layout/sidebar patterns.

---

### Task 1: Backend Provider Service

**Files:**
- Create: `backend/internal/service/upstream_provider.go`
- Create: `backend/internal/service/upstream_provider_adapter.go`
- Create: `backend/internal/service/upstream_provider_sub2api.go`
- Create: `backend/internal/service/upstream_provider_newapi.go`
- Test: `backend/internal/service/upstream_provider_test.go`

- [ ] **Step 1: Write failing service tests**

Cover these behaviors in `backend/internal/service/upstream_provider_test.go`:

```go
func TestUpstreamProviderServiceCreateAndListRedactsPassword(t *testing.T)
func TestUpstreamProviderServiceUpdateKeepsPasswordWhenBlank(t *testing.T)
func TestSub2APIProviderAdapterFetchKeysUsesSingleEndpoint(t *testing.T)
func TestNewAPIProviderAdapterFetchKeysMergesKeysAndGroups(t *testing.T)
func TestNewAPIProviderAdapterWarnsWhenKeyGroupHasNoRatio(t *testing.T)
```

- [ ] **Step 2: Run service tests and verify RED**

Run: `go test ./backend/internal/service -run "TestUpstreamProvider" -count=1`

Expected: FAIL because `NewUpstreamProviderService`, provider types, and adapters do not exist.

- [ ] **Step 3: Implement minimal service and adapters**

Implement:

```go
const SettingKeyUpstreamProviderConfigs = "upstream_provider_configs"

type UpstreamProviderConfig struct {
    Type string `json:"type"`
    Slug string `json:"slug"`
    Name string `json:"name"`
    Enabled bool `json:"enabled"`
    BaseURL string `json:"base_url"`
    LoginURL string `json:"login_url"`
    APIKeysURL string `json:"api_keys_url"`
    GroupsURL string `json:"groups_url"`
    Email string `json:"email"`
    Username string `json:"username"`
    Password string `json:"password,omitempty"`
    PasswordConfigured bool `json:"password_configured,omitempty"`
    AccountNamePrefix string `json:"account_name_prefix"`
    TempDisableMinutes int `json:"temp_disable_minutes,omitempty"`
}
```

Add CRUD methods, password retention, redaction, adapter registry, Sub2API single-endpoint key parsing, and NewAPI login plus keys/groups merge.

- [ ] **Step 4: Run service tests and verify GREEN**

Run: `go test ./backend/internal/service -run "TestUpstreamProvider" -count=1`

Expected: PASS.

### Task 2: Backend Admin Handler And Routes

**Files:**
- Create: `backend/internal/handler/admin/upstream_provider_handler.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/service/wire.go`
- Modify: `backend/internal/server/routes/admin.go`
- Test: `backend/internal/handler/admin/upstream_provider_handler_test.go`

- [ ] **Step 1: Write failing handler tests**

Cover:

```go
func TestUpstreamProviderHandlerCreateListAndUpdate(t *testing.T)
func TestUpstreamProviderHandlerTestSavedProvider(t *testing.T)
```

- [ ] **Step 2: Run handler tests and verify RED**

Run: `go test ./backend/internal/handler/admin -run "TestUpstreamProvider" -count=1`

Expected: FAIL because the handler and routes do not exist.

- [ ] **Step 3: Implement handler and minimal wiring**

Add routes:

```go
GET    /admin/upstream-management/providers
POST   /admin/upstream-management/providers
PUT    /admin/upstream-management/providers/:slug
DELETE /admin/upstream-management/providers/:slug
POST   /admin/upstream-management/providers/:slug/test
GET    /admin/upstream-management/providers/:slug/keys
```

Only touch existing files to inject the new handler/service and call `registerUpstreamManagementRoutes(admin, h)`.

- [ ] **Step 4: Run handler tests and route compile**

Run: `go test ./backend/internal/handler/admin ./backend/internal/server/routes -run "TestUpstreamProvider|TestNoop" -count=1`

Expected: PASS or no tests in routes with successful compilation.

### Task 3: Frontend API And Page

**Files:**
- Create: `frontend/src/api/admin/upstreamProviders.ts`
- Create: `frontend/src/views/admin/upstream-management/UpstreamProvidersView.vue`
- Modify: `frontend/src/api/admin/index.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/api/__tests__/admin.upstreamProviders.spec.ts`

- [ ] **Step 1: Write failing frontend API tests**

Cover list/create/update/delete/test/keys request paths in `admin.upstreamProviders.spec.ts`.

- [ ] **Step 2: Run frontend API tests and verify RED**

Run: `pnpm.cmd test:run src/api/__tests__/admin.upstreamProviders.spec.ts`

Expected: FAIL because `upstreamProviders.ts` does not exist.

- [ ] **Step 3: Implement frontend API and page**

Add typed API wrappers and a focused admin page with:

- Provider list.
- Add/edit modal.
- Type-specific fields for `sub2api` and `newapi`.
- Password not echoed back.
- Test connection result panel.
- Keys sample dialog.

- [ ] **Step 4: Run frontend tests and build**

Run:

```powershell
pnpm.cmd test:run src/api/__tests__/admin.upstreamProviders.spec.ts
pnpm.cmd run build
```

Expected: PASS; build may show existing Vite/Browserslist warnings only.

### Task 4: Final Verification And Commit

**Files:**
- All files changed above.

- [ ] **Step 1: Run backend focused tests**

Run:

```powershell
go test ./backend/internal/service -run "TestUpstreamProvider" -count=1
go test ./backend/internal/handler/admin -run "TestUpstreamProvider" -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend focused test and build**

Run:

```powershell
pnpm.cmd test:run src/api/__tests__/admin.upstreamProviders.spec.ts
pnpm.cmd run build
```

Expected: PASS; known existing warnings are acceptable.

- [ ] **Step 3: Review diff**

Run:

```powershell
git diff --check
git status --short
```

Expected: `git diff --check` has no errors; status contains only upstream provider management changes and this plan file.

- [ ] **Step 4: Commit**

Run:

```powershell
git add docs/superpowers/plans/2026-06-10-upstream-provider-management.md backend frontend
git commit -m "feat: add upstream provider management"
```
