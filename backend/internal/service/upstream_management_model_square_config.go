package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const SettingKeyModelSquarePlatformConfigs = "model_square_platform_configs"

type ModelSquarePlatformModelConfig struct {
	ID                      string   `json:"id"`
	DisplayName             string   `json:"display_name,omitempty"`
	Source                  string   `json:"source,omitempty"`
	InputPrice              *float64 `json:"input_price,omitempty"`
	OutputPrice             *float64 `json:"output_price,omitempty"`
	CacheWritePrice         *float64 `json:"cache_write_price,omitempty"`
	CacheWrite1hPrice       *float64 `json:"cache_write_1h_price,omitempty"`
	CacheReadPrice          *float64 `json:"cache_read_price,omitempty"`
	InputPricePriority      *float64 `json:"input_price_priority,omitempty"`
	OutputPricePriority     *float64 `json:"output_price_priority,omitempty"`
	CacheWritePricePriority *float64 `json:"cache_write_price_priority,omitempty"`
	CacheReadPricePriority  *float64 `json:"cache_read_price_priority,omitempty"`
	ImageInputPrice         *float64 `json:"image_input_price,omitempty"`
	ImageOutputPrice        *float64 `json:"image_output_price,omitempty"`
	PerRequestPrice         *float64 `json:"per_request_price,omitempty"`
}

type ModelSquarePlatformConfig struct {
	Platform              string                           `json:"platform"`
	Name                  string                           `json:"name,omitempty"`
	SyncedFromAccountID   *int64                           `json:"synced_from_account_id,omitempty"`
	SyncedFromAccountName string                           `json:"synced_from_account_name,omitempty"`
	SyncedAt              *time.Time                       `json:"synced_at,omitempty"`
	Models                []ModelSquarePlatformModelConfig `json:"models"`
}

type ModelSquareConfig struct {
	Platforms []ModelSquarePlatformConfig `json:"platforms"`
	UpdatedAt *time.Time                  `json:"updated_at,omitempty"`
}

func (s *UpstreamManagementService) GetModelSquareConfig(ctx context.Context) (ModelSquareConfig, error) {
	if s == nil || s.settingRepo == nil {
		return defaultModelSquareConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelSquarePlatformConfigs)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaultModelSquareConfig(), nil
		}
		return ModelSquareConfig{}, fmt.Errorf("加载模型广场配置失败：%w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultModelSquareConfig(), nil
	}
	var config ModelSquareConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return ModelSquareConfig{}, infraerrors.InternalServer("MODEL_SQUARE_CONFIG_INVALID", "模型广场配置无效")
	}
	return normalizeModelSquareConfig(config), nil
}

func (s *UpstreamManagementService) UpdateModelSquareConfig(ctx context.Context, input ModelSquareConfig) (ModelSquareConfig, error) {
	if err := validateModelSquareConfig(input); err != nil {
		return ModelSquareConfig{}, err
	}
	config := normalizeModelSquareConfig(input)
	now := time.Now().UTC()
	config.UpdatedAt = &now
	if s == nil || s.settingRepo == nil {
		return config, nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return ModelSquareConfig{}, fmt.Errorf("序列化模型广场配置失败：%w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyModelSquarePlatformConfigs, string(raw)); err != nil {
		return ModelSquareConfig{}, fmt.Errorf("保存模型广场配置失败：%w", err)
	}
	return config, nil
}

func defaultModelSquareConfig() ModelSquareConfig {
	return ModelSquareConfig{Platforms: []ModelSquarePlatformConfig{}}
}

func normalizeModelSquareConfig(input ModelSquareConfig) ModelSquareConfig {
	config := ModelSquareConfig{Platforms: make([]ModelSquarePlatformConfig, 0, len(input.Platforms)), UpdatedAt: input.UpdatedAt}
	seenPlatforms := make(map[string]struct{}, len(input.Platforms))
	for _, platform := range input.Platforms {
		normalized := normalizeModelSquarePlatformConfig(platform)
		if normalized.Platform == "" {
			continue
		}
		if _, ok := seenPlatforms[normalized.Platform]; ok {
			continue
		}
		seenPlatforms[normalized.Platform] = struct{}{}
		config.Platforms = append(config.Platforms, normalized)
	}
	return config
}

func normalizeModelSquarePlatformConfig(input ModelSquarePlatformConfig) ModelSquarePlatformConfig {
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	config := ModelSquarePlatformConfig{
		Platform:              platform,
		Name:                  strings.TrimSpace(input.Name),
		SyncedFromAccountID:   input.SyncedFromAccountID,
		SyncedFromAccountName: strings.TrimSpace(input.SyncedFromAccountName),
		SyncedAt:              input.SyncedAt,
		Models:                make([]ModelSquarePlatformModelConfig, 0, len(input.Models)),
	}
	seenModels := make(map[string]struct{}, len(input.Models))
	for _, model := range input.Models {
		normalized := normalizeModelSquareModelConfig(model)
		if normalized.ID == "" {
			continue
		}
		key := strings.ToLower(normalized.ID)
		if _, ok := seenModels[key]; ok {
			continue
		}
		seenModels[key] = struct{}{}
		config.Models = append(config.Models, normalized)
	}
	return config
}

func normalizeModelSquareModelConfig(input ModelSquarePlatformModelConfig) ModelSquarePlatformModelConfig {
	model := ModelSquarePlatformModelConfig{
		ID:                      strings.TrimSpace(input.ID),
		DisplayName:             strings.TrimSpace(input.DisplayName),
		Source:                  normalizeModelSquareModelSource(input.Source),
		InputPrice:              input.InputPrice,
		OutputPrice:             input.OutputPrice,
		CacheWritePrice:         input.CacheWritePrice,
		CacheWrite1hPrice:       input.CacheWrite1hPrice,
		CacheReadPrice:          input.CacheReadPrice,
		InputPricePriority:      input.InputPricePriority,
		OutputPricePriority:     input.OutputPricePriority,
		CacheWritePricePriority: input.CacheWritePricePriority,
		CacheReadPricePriority:  input.CacheReadPricePriority,
		ImageInputPrice:         input.ImageInputPrice,
		ImageOutputPrice:        input.ImageOutputPrice,
		PerRequestPrice:         input.PerRequestPrice,
	}
	if model.DisplayName == "" {
		model.DisplayName = model.ID
	}
	return model
}

func normalizeModelSquareModelSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "sync":
		return "sync"
	default:
		return "manual"
	}
}

func validateModelSquareConfig(input ModelSquareConfig) error {
	for _, platform := range input.Platforms {
		for _, model := range platform.Models {
			if err := validateModelSquareModelPricing(model); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateModelSquareModelPricing(input ModelSquarePlatformModelConfig) error {
	prices := []struct {
		name  string
		value *float64
	}{
		{name: "input_price", value: input.InputPrice},
		{name: "output_price", value: input.OutputPrice},
		{name: "cache_write_price", value: input.CacheWritePrice},
		{name: "cache_write_1h_price", value: input.CacheWrite1hPrice},
		{name: "cache_read_price", value: input.CacheReadPrice},
		{name: "input_price_priority", value: input.InputPricePriority},
		{name: "output_price_priority", value: input.OutputPricePriority},
		{name: "cache_write_price_priority", value: input.CacheWritePricePriority},
		{name: "cache_read_price_priority", value: input.CacheReadPricePriority},
		{name: "image_input_price", value: input.ImageInputPrice},
		{name: "image_output_price", value: input.ImageOutputPrice},
		{name: "per_request_price", value: input.PerRequestPrice},
	}
	for _, item := range prices {
		if item.value != nil && *item.value < 0 {
			return infraerrors.BadRequest("MODEL_SQUARE_PRICE_INVALID", fmt.Sprintf("%s 必须大于或等于 0", item.name))
		}
	}
	return nil
}
