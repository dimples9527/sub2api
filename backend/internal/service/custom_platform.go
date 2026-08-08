package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrCustomPlatformNotFound = infraerrors.NotFound("CUSTOM_PLATFORM_NOT_FOUND", "custom platform not found")
	ErrCustomPlatformExists   = infraerrors.Conflict("CUSTOM_PLATFORM_EXISTS", "custom platform code already exists")
	ErrCustomPlatformInvalid  = infraerrors.BadRequest("CUSTOM_PLATFORM_INVALID", "invalid custom platform configuration")
)

var customPlatformCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,49}$`)

// CustomPlatform 是供应商模块和模型监控模块使用的独立平台字典项。
type CustomPlatform struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomPlatformUpsertParams struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

type CustomPlatformRepository interface {
	List(ctx context.Context, enabledOnly bool) ([]*CustomPlatform, error)
	GetByID(ctx context.Context, id int64) (*CustomPlatform, error)
	GetByCode(ctx context.Context, code string) (*CustomPlatform, error)
	Create(ctx context.Context, item *CustomPlatform) error
	Update(ctx context.Context, item *CustomPlatform) error
	Delete(ctx context.Context, id int64) error
}

type CustomPlatformService interface {
	List(ctx context.Context, enabledOnly bool) ([]*CustomPlatform, error)
	Get(ctx context.Context, id int64) (*CustomPlatform, error)
	Create(ctx context.Context, params CustomPlatformUpsertParams) (*CustomPlatform, error)
	Update(ctx context.Context, id int64, params CustomPlatformUpsertParams) (*CustomPlatform, error)
	Delete(ctx context.Context, id int64) error
	ResolveEnabled(ctx context.Context, code string) (*CustomPlatform, error)
}

type customPlatformService struct {
	repo CustomPlatformRepository
}

func NewCustomPlatformService(repo CustomPlatformRepository) CustomPlatformService {
	return &customPlatformService{repo: repo}
}

func (s *customPlatformService) List(ctx context.Context, enabledOnly bool) ([]*CustomPlatform, error) {
	return s.repo.List(ctx, enabledOnly)
}

func (s *customPlatformService) Get(ctx context.Context, id int64) (*CustomPlatform, error) {
	if id <= 0 {
		return nil, ErrCustomPlatformNotFound
	}
	return s.repo.GetByID(ctx, id)
}

func (s *customPlatformService) Create(ctx context.Context, params CustomPlatformUpsertParams) (*CustomPlatform, error) {
	item, err := normalizeCustomPlatform(params)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByCode(ctx, item.Code); err == nil {
		return nil, ErrCustomPlatformExists
	} else if err != ErrCustomPlatformNotFound {
		return nil, err
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, item.ID)
}

func (s *customPlatformService) Update(ctx context.Context, id int64, params CustomPlatformUpsertParams) (*CustomPlatform, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	item, err := normalizeCustomPlatform(params)
	if err != nil {
		return nil, err
	}
	if item.Code != existing.Code {
		if _, err := s.repo.GetByCode(ctx, item.Code); err == nil {
			return nil, ErrCustomPlatformExists
		} else if err != ErrCustomPlatformNotFound {
			return nil, err
		}
	}
	item.ID = id
	item.CreatedAt = existing.CreatedAt
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *customPlatformService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrCustomPlatformNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *customPlatformService) ResolveEnabled(ctx context.Context, code string) (*CustomPlatform, error) {
	code = normalizeCustomPlatformCode(code)
	if code == "" || IsCorePlatform(code) {
		return nil, ErrCustomPlatformNotFound
	}
	item, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !item.Enabled {
		return nil, ErrCustomPlatformNotFound
	}
	return item, nil
}

func normalizeCustomPlatform(params CustomPlatformUpsertParams) (*CustomPlatform, error) {
	code := normalizeCustomPlatformCode(params.Code)
	name := strings.TrimSpace(params.Name)
	if !customPlatformCodePattern.MatchString(code) || name == "" || IsCorePlatform(code) {
		return nil, ErrCustomPlatformInvalid
	}
	if params.SortOrder < 0 {
		params.SortOrder = 0
	}
	return &CustomPlatform{Code: code, Name: name, Enabled: params.Enabled, SortOrder: params.SortOrder}, nil
}

func normalizeCustomPlatformCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func IsCorePlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok, PlatformComposite:
		return true
	default:
		return false
	}
}

func PlatformLabel(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case PlatformAnthropic:
		return "Anthropic"
	case PlatformOpenAI:
		return "OpenAI"
	case PlatformGemini:
		return "Gemini"
	case PlatformAntigravity:
		return "Antigravity"
	case PlatformGrok:
		return "Grok"
	case PlatformComposite:
		return "Composite"
	default:
		return strings.TrimSpace(platform)
	}
}
