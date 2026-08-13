package service

import (
	"context"
	"encoding/json"
	"testing"
)

func TestUpstreamManagementServiceModelSquareConfigDefaultsAndPersists(t *testing.T) {
	ctx := context.Background()
	settings := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(nil, nil, settings, nil)

	initial, err := svc.GetModelSquareConfig(ctx)
	if err != nil {
		t.Fatalf("GetModelSquareConfig returned error: %v", err)
	}
	if len(initial.Platforms) != 0 {
		t.Fatalf("initial platforms = %d, want 0", len(initial.Platforms))
	}

	input := ModelSquareConfig{Platforms: []ModelSquarePlatformConfig{{
		Platform: " OpenAI ",
		Name:     " OpenAI ",
		Models: []ModelSquarePlatformModelConfig{
			{
				ID:                 " gpt-5.2 ",
				DisplayName:        " GPT-5.2 ",
				Source:             "manual",
				InputPrice:         modelSquareFloatPtr(0.12),
				InputPricePriority: modelSquareFloatPtr(0.24),
				CacheWrite1hPrice:  modelSquareFloatPtr(0.18),
			},
			{ID: "GPT-5.2", Source: "sync"},
			{ID: " ", Source: "manual"},
		},
	}}}

	saved, err := svc.UpdateModelSquareConfig(ctx, input)
	if err != nil {
		t.Fatalf("UpdateModelSquareConfig returned error: %v", err)
	}
	if saved.UpdatedAt == nil {
		t.Fatalf("saved UpdatedAt is nil")
	}
	if len(saved.Platforms) != 1 {
		t.Fatalf("saved platforms = %d, want 1", len(saved.Platforms))
	}
	if saved.Platforms[0].Platform != "openai" {
		t.Fatalf("saved platform = %q, want openai", saved.Platforms[0].Platform)
	}
	if len(saved.Platforms[0].Models) != 1 {
		t.Fatalf("saved models = %d, want 1", len(saved.Platforms[0].Models))
	}
	if saved.Platforms[0].Models[0].ID != "gpt-5.2" {
		t.Fatalf("saved model id = %q, want gpt-5.2", saved.Platforms[0].Models[0].ID)
	}
	if saved.Platforms[0].Models[0].InputPrice == nil || *saved.Platforms[0].Models[0].InputPrice != 0.12 {
		t.Fatalf("saved model input price = %#v, want 0.12", saved.Platforms[0].Models[0].InputPrice)
	}
	if saved.Platforms[0].Models[0].InputPricePriority == nil || *saved.Platforms[0].Models[0].InputPricePriority != 0.24 {
		t.Fatalf("saved model priority input price = %#v, want 0.24", saved.Platforms[0].Models[0].InputPricePriority)
	}
	if saved.Platforms[0].Models[0].CacheWrite1hPrice == nil || *saved.Platforms[0].Models[0].CacheWrite1hPrice != 0.18 {
		t.Fatalf("saved model 1h cache write price = %#v, want 0.18", saved.Platforms[0].Models[0].CacheWrite1hPrice)
	}

	storedRaw := settings.values[SettingKeyModelSquarePlatformConfigs]
	if storedRaw == "" {
		t.Fatalf("stored config is empty")
	}
	var stored ModelSquareConfig
	if err := json.Unmarshal([]byte(storedRaw), &stored); err != nil {
		t.Fatalf("stored config is not valid json: %v", err)
	}

	loaded, err := svc.GetModelSquareConfig(ctx)
	if err != nil {
		t.Fatalf("GetModelSquareConfig after save returned error: %v", err)
	}
	if len(loaded.Platforms) != 1 || loaded.Platforms[0].Models[0].DisplayName != "GPT-5.2" {
		t.Fatalf("loaded config = %#v", loaded)
	}
	if loaded.Platforms[0].Models[0].InputPrice == nil || *loaded.Platforms[0].Models[0].InputPrice != 0.12 {
		t.Fatalf("loaded model input price = %#v, want 0.12", loaded.Platforms[0].Models[0].InputPrice)
	}
	if loaded.Platforms[0].Models[0].InputPricePriority == nil || *loaded.Platforms[0].Models[0].InputPricePriority != 0.24 {
		t.Fatalf("loaded model priority input price = %#v, want 0.24", loaded.Platforms[0].Models[0].InputPricePriority)
	}
	if loaded.Platforms[0].Models[0].CacheWrite1hPrice == nil || *loaded.Platforms[0].Models[0].CacheWrite1hPrice != 0.18 {
		t.Fatalf("loaded model 1h cache write price = %#v, want 0.18", loaded.Platforms[0].Models[0].CacheWrite1hPrice)
	}
}

func TestUpstreamManagementServiceModelSquareConfigRejectsNegativePriceWithChineseMessage(t *testing.T) {
	svc := NewUpstreamManagementService(nil, nil, nil, nil)

	_, err := svc.UpdateModelSquareConfig(context.Background(), ModelSquareConfig{Platforms: []ModelSquarePlatformConfig{{
		Platform: "openai",
		Models: []ModelSquarePlatformModelConfig{{
			ID:         "gpt-5.5",
			InputPrice: modelSquareFloatPtr(-0.1),
		}},
	}}})
	if err == nil {
		t.Fatal("负数价格未返回错误")
	}
	if got := err.Error(); got != "input_price 必须大于或等于 0" {
		t.Fatalf("错误信息 = %q，期望为中文价格校验提示", got)
	}
}

func modelSquareFloatPtr(v float64) *float64 {
	return &v
}
