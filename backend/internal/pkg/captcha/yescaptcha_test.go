package captcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestYesCaptchaSolveTurnstile(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/createTask", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("createTask method = %s, want POST", r.Method)
		}
		var body struct {
			ClientKey string `json:"clientKey"`
			Task      struct {
				Type       string `json:"type"`
				WebsiteURL string `json:"websiteURL"`
				WebsiteKey string `json:"websiteKey"`
			} `json:"task"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode createTask body: %v", err)
			return
		}
		if body.ClientKey != "test-key" {
			t.Errorf("clientKey = %q, want test-key", body.ClientKey)
		}
		if body.Task.Type != "TurnstileTaskProxyless" {
			t.Errorf("task.type = %q, want TurnstileTaskProxyless", body.Task.Type)
		}
		if body.Task.WebsiteURL != "https://example.com/login" {
			t.Errorf("task.websiteURL = %q, want https://example.com/login", body.Task.WebsiteURL)
		}
		if body.Task.WebsiteKey != "site-key" {
			t.Errorf("task.websiteKey = %q, want site-key", body.Task.WebsiteKey)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId": 0,
			"taskId":  12345,
		})
	})
	mux.HandleFunc("/getTaskResult", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ClientKey string `json:"clientKey"`
			TaskID    any    `json:"taskId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode getTaskResult body: %v", err)
			return
		}
		if body.ClientKey != "test-key" {
			t.Errorf("clientKey = %q, want test-key", body.ClientKey)
		}
		if body.TaskID != "12345" {
			t.Errorf("taskId = %#v, want 12345", body.TaskID)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId": 0,
			"status":  "ready",
			"solution": map[string]any{
				"token": "yescaptcha-token",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider, err := New(Config{
		Provider: ProviderYesCaptcha,
		APIKey:   "test-key",
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token, err := provider.SolveTurnstile(ctx, "site-key", "https://example.com/login")
	if err != nil {
		t.Fatalf("SolveTurnstile() error = %v", err)
	}
	if token != "yescaptcha-token" {
		t.Fatalf("SolveTurnstile() token = %q, want yescaptcha-token", token)
	}
}

func TestYesCaptchaCreateTaskSupportsStringTaskID(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/createTask", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId": 0,
			"taskId":  "task-string",
		})
	})
	mux.HandleFunc("/getTaskResult", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TaskID string `json:"taskId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode getTaskResult body: %v", err)
			return
		}
		if body.TaskID != "task-string" {
			t.Errorf("taskId = %q, want task-string", body.TaskID)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId": 0,
			"status":  "ready",
			"solution": map[string]any{
				"token": "string-task-token",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider, err := New(Config{
		Provider: ProviderYesCaptcha,
		APIKey:   "test-key",
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	token, err := provider.SolveTurnstile(ctx, "site-key", "https://example.com")
	if err != nil {
		t.Fatalf("SolveTurnstile() error = %v", err)
	}
	if token != "string-task-token" {
		t.Fatalf("SolveTurnstile() token = %q, want string-task-token", token)
	}
}

func TestYesCaptchaReadyWithoutTokenReturnsError(t *testing.T) {
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
				"token": "",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider, err := New(Config{
		Provider: ProviderYesCaptcha,
		APIKey:   "test-key",
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = provider.SolveTurnstile(ctx, "site-key", "https://example.com")
	if err == nil {
		t.Fatal("SolveTurnstile() error = nil, want empty token error")
	}
	if !strings.Contains(err.Error(), "yescaptcha") {
		t.Fatalf("SolveTurnstile() error = %q, want yescaptcha context", err)
	}
}

func TestYesCaptchaFetchResultStopsOnFailedTask(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/getTaskResult", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorId":          0,
			"errorCode":        "TASK_FAILED",
			"errorDescription": "site verification failed",
			"status":           "failed",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider := newYesCaptcha(Config{
		Provider: ProviderYesCaptcha,
		APIKey:   "test-key",
		Endpoint: server.URL,
	})

	token, ready, err := provider.fetchResult(context.Background(), "task-1")
	require.Error(t, err)
	require.False(t, ready)
	require.Empty(t, token)
	require.Contains(t, err.Error(), "site verification failed")
}
