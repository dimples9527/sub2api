package repository

import (
	"context"
	"database/sql"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type upstreamBalanceRepository struct {
	sql sqlExecutor
}

func NewUpstreamBalanceRepository(_ *dbent.Client, db *sql.DB) service.UpstreamBalanceStore {
	return &upstreamBalanceRepository{sql: db}
}

func (r *upstreamBalanceRepository) AddSnapshot(ctx context.Context, snapshot service.UpstreamBalanceSnapshot) (service.UpstreamBalanceSnapshot, error) {
	if snapshot.CapturedAt.IsZero() {
		snapshot.CapturedAt = time.Now().UTC()
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now().UTC()
	}
	if snapshot.AmountScale <= 0 {
		snapshot.AmountScale = 1
	}
	if snapshot.Status == "" {
		snapshot.Status = "success"
	}
	query := `
		INSERT INTO upstream_balance_snapshots (
			provider_slug, provider_name, provider_type, balance, today_cost, amount_scale, status, error, captured_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at
	`
	err := scanSingleRow(ctx, r.sql, query, []any{
		snapshot.ProviderSlug,
		snapshot.ProviderName,
		snapshot.ProviderType,
		snapshot.Balance,
		snapshot.TodayCost,
		snapshot.AmountScale,
		snapshot.Status,
		snapshot.Error,
		snapshot.CapturedAt,
		snapshot.CreatedAt,
	}, &snapshot.ID, &snapshot.CreatedAt)
	return snapshot, err
}

func (r *upstreamBalanceRepository) ListSnapshots(ctx context.Context, startTime, endTime time.Time) ([]service.UpstreamBalanceSnapshot, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, provider_slug, provider_name, provider_type, balance, today_cost, amount_scale, status, error, captured_at, created_at
		FROM upstream_balance_snapshots
		WHERE captured_at >= $1 AND captured_at < $2
		ORDER BY captured_at ASC, id ASC
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.UpstreamBalanceSnapshot{}
	for rows.Next() {
		var item service.UpstreamBalanceSnapshot
		if err := rows.Scan(
			&item.ID,
			&item.ProviderSlug,
			&item.ProviderName,
			&item.ProviderType,
			&item.Balance,
			&item.TodayCost,
			&item.AmountScale,
			&item.Status,
			&item.Error,
			&item.CapturedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *upstreamBalanceRepository) ListSnapshotsBefore(ctx context.Context, before time.Time) ([]service.UpstreamBalanceSnapshot, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT DISTINCT ON (provider_slug)
			id, provider_slug, provider_name, provider_type, balance, today_cost, amount_scale, status, error, captured_at, created_at
		FROM upstream_balance_snapshots
		WHERE captured_at < $1 AND status = 'success'
		ORDER BY provider_slug, captured_at DESC, id DESC
	`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.UpstreamBalanceSnapshot{}
	for rows.Next() {
		var item service.UpstreamBalanceSnapshot
		if err := rows.Scan(
			&item.ID,
			&item.ProviderSlug,
			&item.ProviderName,
			&item.ProviderType,
			&item.Balance,
			&item.TodayCost,
			&item.AmountScale,
			&item.Status,
			&item.Error,
			&item.CapturedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *upstreamBalanceRepository) ListLatestSnapshots(ctx context.Context) ([]service.UpstreamBalanceSnapshot, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT DISTINCT ON (provider_slug)
			id, provider_slug, provider_name, provider_type, balance, today_cost, amount_scale, status, error, captured_at, created_at
		FROM upstream_balance_snapshots
		ORDER BY provider_slug, captured_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.UpstreamBalanceSnapshot{}
	for rows.Next() {
		var item service.UpstreamBalanceSnapshot
		if err := rows.Scan(
			&item.ID,
			&item.ProviderSlug,
			&item.ProviderName,
			&item.ProviderType,
			&item.Balance,
			&item.TodayCost,
			&item.AmountScale,
			&item.Status,
			&item.Error,
			&item.CapturedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *upstreamBalanceRepository) AddRecharge(ctx context.Context, recharge service.UpstreamBalanceRechargeInput) (service.UpstreamBalanceRecharge, error) {
	if recharge.OccurredAt.IsZero() {
		recharge.OccurredAt = time.Now().UTC()
	}
	if recharge.AmountScale <= 0 {
		recharge.AmountScale = 1
	}
	out := service.UpstreamBalanceRecharge{
		ProviderSlug: recharge.ProviderSlug,
		Amount:       recharge.Amount,
		AmountScale:  recharge.AmountScale,
		Note:         recharge.Note,
		OccurredAt:   recharge.OccurredAt,
		CreatedAt:    time.Now().UTC(),
	}
	query := `
		INSERT INTO upstream_balance_recharges (
			provider_slug, provider_name, amount, amount_scale, note, occurred_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at
	`
	err := scanSingleRow(ctx, r.sql, query, []any{
		out.ProviderSlug,
		out.ProviderName,
		out.Amount,
		out.AmountScale,
		out.Note,
		out.OccurredAt,
		out.CreatedAt,
	}, &out.ID, &out.CreatedAt)
	return out, err
}

func (r *upstreamBalanceRepository) ListRecharges(ctx context.Context, startTime, endTime time.Time) ([]service.UpstreamBalanceRecharge, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, provider_slug, provider_name, amount, amount_scale, note, occurred_at, created_at
		FROM upstream_balance_recharges
		WHERE occurred_at >= $1 AND occurred_at < $2
		ORDER BY occurred_at ASC, id ASC
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.UpstreamBalanceRecharge{}
	for rows.Next() {
		var item service.UpstreamBalanceRecharge
		if err := rows.Scan(
			&item.ID,
			&item.ProviderSlug,
			&item.ProviderName,
			&item.Amount,
			&item.AmountScale,
			&item.Note,
			&item.OccurredAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
