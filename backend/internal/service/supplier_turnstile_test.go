package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchSupplierNewAPITurnstileSiteKeyUsesPublicSiteKeyWhenCheckFlagIsFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/status", r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true,"data":{"turnstile_check":false,"turnstile_site_key":"newapi-site-key"}}`))
	}))
	defer server.Close()

	siteKey, err := fetchSupplierNewAPITurnstileSiteKey(context.Background(), server.Client(), &SupplierProvider{BaseURL: server.URL})

	require.NoError(t, err)
	require.Equal(t, "newapi-site-key", siteKey)
}
