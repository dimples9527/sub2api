package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthTokenPublicSnapshotMasksSecret(t *testing.T) {
	token := SupplierProviderAuthToken{
		AccessToken:  "0123456789abcdef0123456789abcdef",
		TokenType:    "Bearer",
		ExpiresAt:    time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC),
		UserID:       42,
		CookieHeader: "session=full-cookie-value",
	}

	snapshot := publicSupplierProviderAuthTokenSnapshot(token, time.Minute, time.Date(2026, 8, 5, 11, 59, 30, 0, time.UTC))

	require.Equal(t, 32, snapshot.TokenLength)
	require.Equal(t, "0123…cdef", snapshot.TokenSummary)
	require.NotContains(t, snapshot.TokenSummary, token.AccessToken)
	require.NotContains(t, snapshot.TokenSummary, token.CookieHeader)
	require.True(t, snapshot.CookiePresent)
	require.Equal(t, "Bearer", snapshot.TokenType)
	require.Equal(t, int64(30), snapshot.RemainingSeconds)
	require.NotEmpty(t, snapshot.TokenFingerprint)
}

func TestSupplierProviderAuthTokenPublicSnapshotReportsCachedCookieSession(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	token := SupplierProviderAuthToken{
		CookieHeader: "session=cached-cookie",
		ExpiresAt:    now.Add(time.Minute),
	}

	snapshot := publicSupplierProviderAuthTokenSnapshot(token, time.Minute, now)

	require.Equal(t, SupplierProviderAuthCacheCached, snapshot.Status)
	require.True(t, snapshot.Cached)
	require.True(t, snapshot.CookiePresent)
	require.Empty(t, snapshot.TokenSummary)
	require.Empty(t, snapshot.TokenFingerprint)
	require.Zero(t, snapshot.TokenLength)
}

func TestSupplierProviderAuthTokenPublicSnapshotKeepsTimeExpiredTokenUsable(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	token := SupplierProviderAuthToken{
		AccessToken: "expired-token",
		ExpiresAt:   now.Add(-time.Second),
	}

	snapshot := publicSupplierProviderAuthTokenSnapshot(token, 0, now)

	// 新策略：时间过期不主动判失效，等待接口鉴权失败后再清理。
	require.Equal(t, SupplierProviderAuthCacheCached, snapshot.Status)
	require.Equal(t, int64(0), snapshot.RemainingSeconds)
}

func TestSanitizeSupplierProviderAuthErrorRemovesSecrets(t *testing.T) {
	err := errors.New("login failed token=full-access-token cookie=session=secret password=plain-password redis://user:pass@example.internal:6379")

	message := sanitizeSupplierProviderAuthError(err)

	require.NotContains(t, message, "full-access-token")
	require.NotContains(t, message, "session=secret")
	require.NotContains(t, message, "plain-password")
	require.NotContains(t, message, "redis://")
	require.Contains(t, strings.ToLower(message), "redacted")
	require.LessOrEqual(t, len(message), supplierProviderAuthErrorLimit)
}

func TestSanitizeSupplierProviderAuthErrorRemovesHeadersAndJSONSecrets(t *testing.T) {
	err := errors.New(`request failed Authorization: Bearer auth-secret "access_token":"json-secret" Cookie: sid=cookie-secret; refresh=refresh-secret`)

	message := sanitizeSupplierProviderAuthError(err)

	require.NotContains(t, message, "auth-secret")
	require.NotContains(t, message, "json-secret")
	require.NotContains(t, message, "cookie-secret")
	require.NotContains(t, message, "refresh-secret")
	require.Contains(t, message, "[REDACTED]")
}

func TestSupplierProviderAuthEventInputNormalizesDefaults(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	event := SupplierProviderAuthEventInput{
		ProviderID: 42,
		EventType:  SupplierProviderAuthEventLoginSuccess,
		StartedAt:  now,
		FinishedAt: now.Add(250 * time.Millisecond),
	}

	record := normalizeSupplierProviderAuthEvent(event)

	require.Equal(t, int64(42), record.ProviderID)
	require.Equal(t, SupplierProviderAuthSourceUnknown, record.Source)
	require.Equal(t, SupplierProviderAuthStatusSuccess, record.Status)
	require.Equal(t, int64(250), record.DurationMS)
	require.Equal(t, now, record.StartedAt)
	require.Equal(t, now.Add(250*time.Millisecond), record.FinishedAt)
}
