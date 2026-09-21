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
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountTestStatusHTTPUpstream struct {
	resp *http.Response
	// delay 模拟上游往返耗时，用来断言落库的耗时确实是测出来的，而不是常量 0。
	delay time.Duration
}

func (u *accountTestStatusHTTPUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *accountTestStatusHTTPUpstream) DoWithTLS(_ *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	if u.resp == nil {
		return nil, fmt.Errorf("missing response")
	}
	if u.delay > 0 {
		time.Sleep(u.delay)
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
		httpUpstream: &accountTestStatusHTTPUpstream{resp: resp, delay: 20 * time.Millisecond},
	}

	err := svc.TestAccountConnection(newAccountTestStatusContext(), account.ID, "gpt-5.4", "", "")
	require.NoError(t, err)

	require.Equal(t, "success", repo.updatedExtra["last_test_status"])
	require.NotEmpty(t, repo.updatedExtra["last_tested_at"])
	require.Equal(t, "", repo.updatedExtra["last_test_error"])
	// 成功必须落耗时，否则列表永远看不到「上次测试成功的用时」
	latency, ok := repo.updatedExtra["last_test_latency_ms"].(int64)
	require.True(t, ok, "last_test_latency_ms 必须是 int64 毫秒")
	require.Greater(t, latency, int64(0))
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
	// 失败不写耗时：保留上一次成功的数值，避免把成功用时抹成空
	require.NotContains(t, repo.updatedExtra, "last_test_latency_ms")
}

func TestAccountTestService_TestAccountConnectionPersistsCNProviderSuccessStatus(t *testing.T) {
	account := &Account{
		ID:          91,
		Platform:    PlatformDeepseek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":      "sk-deepseek-test",
			"api_protocol": APIProtocolChatCompletions,
			"base_url":     "http://upstream.example/v1",
		},
	}
	repo := &accountTestStatusRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{account.ID: account},
		},
	}
	resp := newAccountTestStatusResponse(http.StatusOK, `data: {"choices":[{"delta":{"content":"chat ok"},"finish_reason":"stop"}]}

data: [DONE]

`)
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: &accountTestStatusHTTPUpstream{resp: resp, delay: 20 * time.Millisecond},
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true},
			},
		},
	}

	err := svc.TestAccountConnection(newAccountTestStatusContext(), account.ID, "deepseek-chat", "", "")
	require.NoError(t, err)

	// 国内厂商分支同样要落耗时，否则该平台的账号永远不显示用时
	cnLatency, ok := repo.updatedExtra["last_test_latency_ms"].(int64)
	require.True(t, ok, "last_test_latency_ms 必须是 int64 毫秒")
	require.Greater(t, cnLatency, int64(0))
	require.Equal(t, "success", repo.updatedExtra["last_test_status"])
	require.NotEmpty(t, repo.updatedExtra["last_tested_at"])
	require.Equal(t, "", repo.updatedExtra["last_test_error"])
}

func TestAccountTestService_TestAccountConnectionPersistsCNProviderFailedStatus(t *testing.T) {
	account := &Account{
		ID:          92,
		Platform:    PlatformDeepseek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":      "sk-deepseek-test",
			"api_protocol": APIProtocolChatCompletions,
			"base_url":     "http://upstream.example/v1",
		},
	}
	repo := &accountTestStatusRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{account.ID: account},
		},
	}
	resp := newAccountTestStatusResponse(http.StatusUnauthorized, `{"error":"invalid DeepSeek token"}`)
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: &accountTestStatusHTTPUpstream{resp: resp},
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true},
			},
		},
	}

	err := svc.TestAccountConnection(newAccountTestStatusContext(), account.ID, "deepseek-chat", "", "")
	require.Error(t, err)

	require.Equal(t, "failed", repo.updatedExtra["last_test_status"])
	require.NotEmpty(t, repo.updatedExtra["last_tested_at"])
	require.Contains(t, repo.updatedExtra["last_test_error"], "invalid DeepSeek token")
	require.NotContains(t, repo.updatedExtra, "last_test_latency_ms")
}

type accountTestStatusDeadlineUpstream struct{}

func (u *accountTestStatusDeadlineUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("Post %q: %w", req.URL.String(), context.DeadlineExceeded)
}

func (u *accountTestStatusDeadlineUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, "", 0, 0)
}

type accountTestStatusCtxAwareRepo struct {
	mockAccountRepoForGemini
	updatedExtra map[string]any
}

func (r *accountTestStatusCtxAwareRepo) UpdateExtra(ctx context.Context, _ int64, updates map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.updatedExtra = updates
	return nil
}

func TestAccountTestService_PersistsFailedStatusAfterParentDeadlineExceeded(t *testing.T) {
	account := &Account{
		ID:          93,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "test-token"},
	}
	repo := &accountTestStatusCtxAwareRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{account.ID: account},
		},
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: &accountTestStatusDeadlineUpstream{},
	}

	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	c := newAccountTestStatusContext()
	c.Request = c.Request.WithContext(expired)

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", "")
	require.Error(t, err)

	require.Equal(t, "failed", repo.updatedExtra["last_test_status"])
	require.NotEmpty(t, repo.updatedExtra["last_tested_at"])
	require.Contains(t, repo.updatedExtra["last_test_error"], "context deadline exceeded")
}
