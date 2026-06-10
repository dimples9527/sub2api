package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const DefaultUpstreamProviderModelSquarePath = "/api/v1/model-square"

type upstreamProviderModelSquareFetcher interface {
	FetchModelSquare(ctx context.Context, provider UpstreamProviderConfig) ([]byte, error)
}

type upstreamModelSquareToken struct {
	Value string
	Type  string
}

func (s *UpstreamProviderService) FetchDefaultModelSquare(ctx context.Context) (json.RawMessage, UpstreamProviderConfig, error) {
	provider, err := s.GetDefaultProvider(ctx)
	if err != nil {
		return nil, UpstreamProviderConfig{}, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return nil, UpstreamProviderConfig{}, err
	}
	fetcher, ok := adapter.(upstreamProviderModelSquareFetcher)
	if !ok {
		return nil, UpstreamProviderConfig{}, fmt.Errorf("upstream provider type %s does not support model square", provider.Type)
	}
	payload, err := fetcher.FetchModelSquare(ctx, provider)
	if err != nil {
		return nil, UpstreamProviderConfig{}, err
	}
	var raw json.RawMessage = payload
	return raw, redactUpstreamProvider(provider), nil
}

func (a *Sub2APIProviderAdapter) FetchModelSquare(ctx context.Context, provider UpstreamProviderConfig) ([]byte, error) {
	token := upstreamModelSquareToken{}
	if provider.Email != "" || provider.Password != "" {
		nextToken, err := a.loginForModelSquare(ctx, provider)
		if err != nil {
			return nil, err
		}
		token = nextToken
	}
	payload, status, err := a.requestModelSquare(ctx, provider, token)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, upstreamProviderHTTPError("sub2api model square", status, payload)
	}
	return payload, nil
}

func (a *Sub2APIProviderAdapter) loginForModelSquare(ctx context.Context, provider UpstreamProviderConfig) (upstreamModelSquareToken, error) {
	loginPath := provider.LoginURL
	if loginPath == "" {
		loginPath = "/api/v1/auth/login"
	}
	body, err := json.Marshal(map[string]string{
		"email":    provider.Email,
		"password": provider.Password,
	})
	if err != nil {
		return upstreamModelSquareToken{}, fmt.Errorf("marshal sub2api model square login payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamProviderURL(provider, loginPath), bytes.NewReader(body))
	if err != nil {
		return upstreamModelSquareToken{}, fmt.Errorf("create sub2api model square login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return upstreamModelSquareToken{}, fmt.Errorf("sub2api model square login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return upstreamModelSquareToken{}, fmt.Errorf("read sub2api model square login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamModelSquareToken{}, upstreamProviderHTTPError("sub2api model square login", resp.StatusCode, raw)
	}
	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
		AccessToken string `json:"access_token"`
		Token       string `json:"token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return upstreamModelSquareToken{}, fmt.Errorf("decode sub2api model square login response: %w", err)
	}
	if parsed.Code != 0 {
		if parsed.Message == "" {
			parsed.Message = "unknown error"
		}
		return upstreamModelSquareToken{}, fmt.Errorf("sub2api model square login failed: %s", parsed.Message)
	}
	token := firstNonEmptyModelSquareString(parsed.Data.AccessToken, parsed.Data.Token, parsed.AccessToken, parsed.Token)
	if token == "" {
		return upstreamModelSquareToken{}, fmt.Errorf("sub2api model square login failed: missing token")
	}
	tokenType := firstNonEmptyModelSquareString(parsed.Data.TokenType, parsed.TokenType, "Bearer")
	return upstreamModelSquareToken{Value: token, Type: tokenType}, nil
}

func (a *Sub2APIProviderAdapter) requestModelSquare(ctx context.Context, provider UpstreamProviderConfig, token upstreamModelSquareToken) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamProviderURL(provider, DefaultUpstreamProviderModelSquarePath), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create sub2api model square request: %w", err)
	}
	if token.Value != "" {
		if token.Type == "" {
			token.Type = "Bearer"
		}
		req.Header.Set("Authorization", token.Type+" "+token.Value)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("sub2api model square request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read sub2api model square response: %w", err)
	}
	return raw, resp.StatusCode, nil
}

func (a *NewAPIProviderAdapter) FetchModelSquare(ctx context.Context, provider UpstreamProviderConfig) ([]byte, error) {
	session, _, _, err := a.login(ctx, provider)
	if err != nil {
		return nil, err
	}
	payload, status, err := a.requestModelSquare(ctx, provider, session)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, upstreamProviderHTTPError("newapi model square", status, payload)
	}
	return payload, nil
}

func (a *NewAPIProviderAdapter) requestModelSquare(ctx context.Context, provider UpstreamProviderConfig, session newAPIProviderSession) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamProviderURL(provider, DefaultUpstreamProviderModelSquarePath), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create newapi model square request: %w", err)
	}
	req.Header.Set("New-Api-User", strconv.FormatInt(session.UserID, 10))
	if session.CookieHeader != "" {
		req.Header.Set("Cookie", session.CookieHeader)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("newapi model square request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read newapi model square response: %w", err)
	}
	return raw, resp.StatusCode, nil
}

func firstNonEmptyModelSquareString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
