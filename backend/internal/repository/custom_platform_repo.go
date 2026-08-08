package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type customPlatformRepository struct {
	db *sql.DB
}

func NewCustomPlatformRepository(db *sql.DB) service.CustomPlatformRepository {
	return &customPlatformRepository{db: db}
}

const customPlatformSelect = `
SELECT id, code, name, enabled, sort_order, created_at, updated_at
FROM custom_platforms`

func (r *customPlatformRepository) List(ctx context.Context, enabledOnly bool) ([]*service.CustomPlatform, error) {
	where := "deleted_at IS NULL"
	if enabledOnly {
		where += " AND enabled = TRUE"
	}
	rows, err := r.db.QueryContext(ctx, customPlatformSelect+" WHERE "+where+" ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("query custom platforms: %w", err)
	}
	defer rows.Close()
	items := make([]*service.CustomPlatform, 0)
	for rows.Next() {
		item, scanErr := scanCustomPlatform(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom platforms: %w", err)
	}
	return items, nil
}

func (r *customPlatformRepository) GetByID(ctx context.Context, id int64) (*service.CustomPlatform, error) {
	item, err := scanCustomPlatform(r.db.QueryRowContext(ctx, customPlatformSelect+" WHERE id=$1 AND deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCustomPlatformNotFound
	}
	return item, err
}

func (r *customPlatformRepository) GetByCode(ctx context.Context, code string) (*service.CustomPlatform, error) {
	item, err := scanCustomPlatform(r.db.QueryRowContext(ctx, customPlatformSelect+" WHERE code=$1 AND deleted_at IS NULL", code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCustomPlatformNotFound
	}
	return item, err
}

func (r *customPlatformRepository) Create(ctx context.Context, item *service.CustomPlatform) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO custom_platforms (code, name, enabled, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, updated_at`, item.Code, item.Name, item.Enabled, item.SortOrder).
		Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return mapCustomPlatformError(err)
	}
	return nil
}

func (r *customPlatformRepository) Update(ctx context.Context, item *service.CustomPlatform) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE custom_platforms SET code=$2, name=$3, enabled=$4, sort_order=$5, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, item.ID, item.Code, item.Name, item.Enabled, item.SortOrder)
	if err != nil {
		return mapCustomPlatformError(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrCustomPlatformNotFound
	}
	return nil
}

func (r *customPlatformRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE custom_platforms SET enabled=FALSE, deleted_at=NOW(), updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrCustomPlatformNotFound
	}
	return nil
}

type customPlatformScanner interface{ Scan(dest ...any) error }

func scanCustomPlatform(scanner customPlatformScanner) (*service.CustomPlatform, error) {
	item := &service.CustomPlatform{}
	err := scanner.Scan(&item.ID, &item.Code, &item.Name, &item.Enabled, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func mapCustomPlatformError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrCustomPlatformExists
	}
	return err
}
