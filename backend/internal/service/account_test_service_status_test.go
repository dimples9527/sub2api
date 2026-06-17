//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountTestStatusHTTPUpstream struct {
	resp *http.Response
}

func (u *accountTestStatusHTTPUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *accountTestStatusHTTPUpstream) DoWithTLS(_ *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	if u.resp == nil {
		return nil, fmt.Errorf("missing response")
	}
	return u.resp, nil
}

type accountTestStatusRepo struct {
	mockAccountRepoForGemini
	updatedExtra map[string]any
}

func (r *accountTestStatusRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updatedExtra = updates
	return nil
}

func newAccountTestStatusContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)
	return c
}

func newAccountTestStatusResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestAccountTestService_TestAccountConnectionPersistsSuccessStatus(t *testing.T) {
	account := &Account{
		ID:          89,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	repo := &accountTestStatusRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{account.ID: account},
		},
	}
	resp := newAccountTestStatusResponse(http.StatusOK, `data: {"type":"response.completed"}

`)
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: &accountTestStatusHTTPUpstream{resp: resp},
	}

	err := svc.TestAccountConnection(newAccountTestStatusContext(), account.ID, "gpt-5.4", "", "")
	require.NoError(t, err)

	require.Equal(t, "success", repo.updatedExtra["last_test_status"])
	require.NotEmpty(t, repo.updatedExtra["last_tested_at"])
	require.Equal(t, "", repo.updatedExtra["last_test_error"])
}

func TestAccountTestService_TestAccountConnectionPersistsFailedStatus(t *testing.T) {
	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	repo := &accountTestStatusRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{account.ID: account},
		},
	}
	resp := newAccountTestStatusResponse(http.StatusUnauthorized, `{"error":"bad token"}`)
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: &accountTestStatusHTTPUpstream{resp: resp},
	}

	err := svc.TestAccountConnection(newAccountTestStatusContext(), account.ID, "gpt-5.4", "", "")
	require.Error(t, err)

	require.Equal(t, "failed", repo.updatedExtra["last_test_status"])
	require.NotEmpty(t, repo.updatedExtra["last_tested_at"])
	require.Contains(t, repo.updatedExtra["last_test_error"], "bad token")
}
