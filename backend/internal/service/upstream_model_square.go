package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const DefaultUpstreamProviderModelSquarePath = "/api/v1/model-square"

type upstreamProviderModelSquareFetcher interface {
	FetchModelSquare(ctx context.Context, provider UpstreamProviderConfig) ([]byte, error)
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
	for attempt := 0; attempt < 2; attempt++ {
		auth, err := a.ensureAuth(ctx, provider)
		if err != nil {
			return nil, err
		}
		payload, status, err := a.request(ctx, provider, auth, DefaultUpstreamProviderModelSquarePath, "sub2api model square")
		if err != nil {
			return nil, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("sub2api model square", status, payload)
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearAuth(provider.Slug)
				continue
			}
			return nil, requestErr
		}
		return payload, nil
	}
	return nil, fmt.Errorf("sub2api model square failed after auth retry")
}

func (a *NewAPIProviderAdapter) FetchModelSquare(ctx context.Context, provider UpstreamProviderConfig) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		session, err := a.ensureSession(ctx, provider)
		if err != nil {
			return nil, err
		}
		payload, status, err := a.request(ctx, provider, session, DefaultUpstreamProviderModelSquarePath, "newapi model square")
		if err != nil {
			return nil, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("newapi model square", status, payload)
			if attempt == 0 && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearSession(provider.Slug)
				continue
			}
			return nil, requestErr
		}
		return payload, nil
	}
	return nil, fmt.Errorf("newapi model square failed after auth retry")
}

func upstreamProviderAuthFailureHint(status int, payload []byte, err error) bool {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	text := strings.ToLower(strings.TrimSpace(string(payload)))
	if text == "" && err != nil {
		text = strings.ToLower(err.Error())
	}
	for _, keyword := range []string{"unauthorized", "forbidden", "token expired", "invalid token", "session expired", "missing cookie", "auth failed"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
