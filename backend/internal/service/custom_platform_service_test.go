package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type customPlatformServiceRepoStub struct {
	created *CustomPlatform
}

func (s *customPlatformServiceRepoStub) List(context.Context, bool) ([]*CustomPlatform, error) {
	return nil, nil
}

func (s *customPlatformServiceRepoStub) GetByID(_ context.Context, id int64) (*CustomPlatform, error) {
	if s.created != nil && s.created.ID == id {
		return s.created, nil
	}
	return nil, ErrCustomPlatformNotFound
}

func (s *customPlatformServiceRepoStub) GetByCode(context.Context, string) (*CustomPlatform, error) {
	return nil, ErrCustomPlatformNotFound
}

func (s *customPlatformServiceRepoStub) Create(_ context.Context, item *CustomPlatform) error {
	item.ID = 1
	s.created = item
	return nil
}

func (s *customPlatformServiceRepoStub) Update(context.Context, *CustomPlatform) error {
	return nil
}

func (s *customPlatformServiceRepoStub) Delete(context.Context, int64) error {
	return nil
}

func TestNormalizeCustomPlatformColor(t *testing.T) {
	t.Run("未指定颜色时使用默认色", func(t *testing.T) {
		item, err := normalizeCustomPlatform(CustomPlatformUpsertParams{Code: "foo", Name: "Foo"})
		require.NoError(t, err)
		require.Equal(t, defaultCustomPlatformColor, item.Color)
	})

	t.Run("合法颜色归一化为小写", func(t *testing.T) {
		item, err := normalizeCustomPlatform(CustomPlatformUpsertParams{Code: "foo", Name: "Foo", Color: "#3B82F6"})
		require.NoError(t, err)
		require.Equal(t, "#3b82f6", item.Color)
	})

	t.Run("非法颜色被拒绝", func(t *testing.T) {
		for _, color := range []string{"blue", "#12345", "123456", "#gggggg", "#1234567"} {
			_, err := normalizeCustomPlatform(CustomPlatformUpsertParams{Code: "foo", Name: "Foo", Color: color})
			require.ErrorIs(t, err, ErrCustomPlatformInvalid, "color %q 应被拒绝", color)
		}
	})

	t.Run("创建时持久化颜色", func(t *testing.T) {
		repo := &customPlatformServiceRepoStub{}
		svc := NewCustomPlatformService(repo)
		item, err := svc.Create(context.Background(), CustomPlatformUpsertParams{
			Code:      "foo",
			Name:      "Foo",
			Color:     "#8b5cf6",
			SortOrder: 5,
		})
		require.NoError(t, err)
		require.Equal(t, "#8b5cf6", item.Color)
		require.Equal(t, "#8b5cf6", repo.created.Color)
		require.Equal(t, "foo", repo.created.Code)
	})
}
