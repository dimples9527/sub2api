package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTwoCaptchaSolveTurnstile(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/createTask", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId": 0,
			"taskId":  12345,
		})
	})
	mux.HandleFunc("/getTaskResult", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId": 0,
			"status":  "ready",
			"solution": map[string]any{
				"token": "turnstile-token",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider, err := New(Config{
		Provider: ProviderTwoCaptcha,
		APIKey:   "test-key",
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	two, ok := provider.(*twoCaptcha)
	if !ok {
		t.Fatalf("expected *twoCaptcha, got %T", provider)
	}
	two.http = server.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token, err := provider.SolveTurnstile(ctx, "site-key", "https://example.com")
	if err != nil {
		t.Fatalf("SolveTurnstile() error = %v", err)
	}
	if token != "turnstile-token" {
		t.Fatalf("SolveTurnstile() token = %q, want turnstile-token", token)
	}
}

func TestValidateConfigRequiresAPIKey(t *testing.T) {
	t.Parallel()
	if err := ValidateConfig(Config{Provider: ProviderTwoCaptcha}); err == nil {
		t.Fatal("ValidateConfig() error = nil, want api key empty error")
	}
}
