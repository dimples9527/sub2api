package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
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
	/*
		取结构体里的 Message，而不是 err.Error()：ApplicationError 的 Error() 会把
		code / reason / metadata 一并拼进去，用整串比对等于把错误包装格式也钉进断言，
		包装一变就红 —— 而这条测试真正要守的是「提示是中文」和「reason 可被前端识别」。
	*/
	var appErr *infraerrors.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("错误类型 = %T，期望 *infraerrors.ApplicationError", err)
	}
	if got := appErr.Message; got != "input_price 必须大于或等于 0" {
		t.Fatalf("错误信息 = %q，期望为中文价格校验提示", got)
	}
	if got := appErr.Reason; got != "MODEL_SQUARE_PRICE_INVALID" {
		t.Fatalf("错误 reason = %q，期望 MODEL_SQUARE_PRICE_INVALID", got)
	}
}

func modelSquareFloatPtr(v float64) *float64 {
	return &v
}

/*
分组绑定是模型在模型广场里的归属来源，规范化必须严格：非法值混进去会让展示页按分组
筛选时出现「这个分组里有个 ID 根本不存在」的静默错配，而排序则直接决定前端
「未保存改动」检测会不会误报（同一组 ID 换个顺序就是不同的序列化结果）。
*/
func TestUpstreamManagementServiceModelSquareConfigNormalizesGroupIDs(t *testing.T) {
	ctx := context.Background()
	settings := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(nil, nil, settings, nil)

	saved, err := svc.UpdateModelSquareConfig(ctx, ModelSquareConfig{Platforms: []ModelSquarePlatformConfig{{
		Platform: "openai",
		Models: []ModelSquarePlatformModelConfig{
			{ID: "gpt-5.5", GroupIDs: []int64{9, 3, 3, 0, -2, 7}},
			{ID: "unbound-model"},
			{ID: "garbage-model", GroupIDs: []int64{0, -1}},
		},
	}}})
	if err != nil {
		t.Fatalf("UpdateModelSquareConfig returned error: %v", err)
	}

	bound := saved.Platforms[0].Models[0].GroupIDs
	if len(bound) != 3 || bound[0] != 3 || bound[1] != 7 || bound[2] != 9 {
		t.Fatalf("bound group ids = %#v, want [3 7 9]", bound)
	}
	// 未绑定与「全是非法值」必须都归到 nil，而不是空切片：json 的 omitempty 只在 nil 上生效，
	// 存成 [] 会让「解绑全部」和「从未绑定」在存储层变成两种形态，前端脏检测随之误报。
	if unbound := saved.Platforms[0].Models[1].GroupIDs; unbound != nil {
		t.Fatalf("unbound group ids = %#v, want nil", unbound)
	}
	if garbage := saved.Platforms[0].Models[2].GroupIDs; garbage != nil {
		t.Fatalf("all-invalid group ids = %#v, want nil", garbage)
	}

	raw := settings.values[SettingKeyModelSquarePlatformConfigs]
	if strings.Contains(raw, `"group_ids":[]`) {
		t.Fatalf("空绑定不应序列化成空数组：%s", raw)
	}

	loaded, err := svc.GetModelSquareConfig(ctx)
	if err != nil {
		t.Fatalf("GetModelSquareConfig after save returned error: %v", err)
	}
	reloaded := loaded.Platforms[0].Models[0].GroupIDs
	if len(reloaded) != 3 || reloaded[0] != 3 || reloaded[1] != 7 || reloaded[2] != 9 {
		t.Fatalf("reloaded group ids = %#v, want [3 7 9]", reloaded)
	}
}
