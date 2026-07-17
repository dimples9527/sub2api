package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type upstreamAccountHealthGuardRecordRepository struct {
	db *sql.DB
}

func NewUpstreamAccountHealthGuardRecordRepository(db *sql.DB) service.UpstreamAccountHealthGuardRecordStore {
	return &upstreamAccountHealthGuardRecordRepository{db: db}
}

func (r *upstreamAccountHealthGuardRecordRepository) SaveRecord(ctx context.Context, record service.UpstreamAccountHealthGuardRunRecord, keepLimit int) error {
	if r == nil || r.db == nil {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	skipReasons := record.Summary.SkipReasons
	if skipReasons == nil {
		skipReasons = []service.UpstreamAccountHealthGuardSkipReason{}
	}
	skipReasonsJSON, err := json.Marshal(skipReasons)
	if err != nil {
		return fmt.Errorf("marshal health guard skip reasons: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO upstream_account_health_guard_runs (
			id, trigger, status, message, started_at, finished_at,
			total_accounts, checked_count, healthy_count, slow_count, failed_count,
			skipped_count, disabled_count, recovered_count, unchanged_count, skip_reasons, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NOW())
		ON CONFLICT (id) DO UPDATE SET
			trigger = EXCLUDED.trigger,
			status = EXCLUDED.status,
			message = EXCLUDED.message,
			started_at = EXCLUDED.started_at,
			finished_at = EXCLUDED.finished_at,
			total_accounts = EXCLUDED.total_accounts,
			checked_count = EXCLUDED.checked_count,
			healthy_count = EXCLUDED.healthy_count,
			slow_count = EXCLUDED.slow_count,
			failed_count = EXCLUDED.failed_count,
			skipped_count = EXCLUDED.skipped_count,
			disabled_count = EXCLUDED.disabled_count,
			recovered_count = EXCLUDED.recovered_count,
			unchanged_count = EXCLUDED.unchanged_count,
			skip_reasons = EXCLUDED.skip_reasons
	`, record.ID, record.Trigger, record.Status, record.Message, record.StartedAt, record.FinishedAt,
		record.Summary.TotalAccounts, record.Summary.CheckedCount, record.Summary.HealthyCount,
		record.Summary.SlowCount, record.Summary.FailedCount, record.Summary.SkippedCount,
		record.Summary.DisabledCount, record.Summary.RecoveredCount, record.Summary.UnchangedCount,
		string(skipReasonsJSON),
	); err != nil {
		return fmt.Errorf("save health guard run: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM upstream_account_health_guard_run_items WHERE run_id = $1`, record.ID); err != nil {
		return fmt.Errorf("replace health guard run items: %w", err)
	}
	for _, item := range record.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO upstream_account_health_guard_run_items (
				run_id, account_id, account_name, platform, provider_slug, provider_name,
				model_id, schedulable_before, schedulable_after, status, test_status,
				latency_ms, latency_limit_ms, consecutive_failed, consecutive_slow,
				consecutive_healthy, action, reason, error_message, started_at, finished_at, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,NOW())
		`, record.ID, item.AccountID, item.AccountName, item.Platform, item.ProviderSlug, item.ProviderName,
			item.ModelID, item.SchedulableBefore, item.SchedulableAfter, item.Status, item.TestStatus,
			item.LatencyMs, item.LatencyLimitMs, item.ConsecutiveFailed, item.ConsecutiveSlow,
			item.ConsecutiveHealthy, item.Action, item.Reason, item.ErrorMessage, item.StartedAt, item.FinishedAt,
		); err != nil {
			return fmt.Errorf("save health guard run item: %w", err)
		}
	}

	if keepLimit > 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM upstream_account_health_guard_runs
			WHERE id IN (
				SELECT id FROM upstream_account_health_guard_runs
				ORDER BY finished_at DESC, created_at DESC, id DESC
				OFFSET $1
			)
		`, keepLimit); err != nil {
			return fmt.Errorf("prune health guard runs: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *upstreamAccountHealthGuardRecordRepository) ListRecords(ctx context.Context, limit int) ([]service.UpstreamAccountHealthGuardRunRecord, error) {
	if r == nil || r.db == nil {
		return []service.UpstreamAccountHealthGuardRunRecord{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, trigger, status, message, started_at, finished_at,
			total_accounts, checked_count, healthy_count, slow_count, failed_count,
			skipped_count, disabled_count, recovered_count, unchanged_count, skip_reasons
		FROM upstream_account_health_guard_runs
		ORDER BY finished_at DESC, created_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []service.UpstreamAccountHealthGuardRunRecord{}
	for rows.Next() {
		var record service.UpstreamAccountHealthGuardRunRecord
		var skipReasonsRaw []byte
		if err := rows.Scan(
			&record.ID,
			&record.Trigger,
			&record.Status,
			&record.Message,
			&record.StartedAt,
			&record.FinishedAt,
			&record.Summary.TotalAccounts,
			&record.Summary.CheckedCount,
			&record.Summary.HealthyCount,
			&record.Summary.SlowCount,
			&record.Summary.FailedCount,
			&record.Summary.SkippedCount,
			&record.Summary.DisabledCount,
			&record.Summary.RecoveredCount,
			&record.Summary.UnchangedCount,
			&skipReasonsRaw,
		); err != nil {
			return nil, err
		}
		if len(skipReasonsRaw) > 0 {
			if err := json.Unmarshal(skipReasonsRaw, &record.Summary.SkipReasons); err != nil {
				return nil, fmt.Errorf("decode health guard skip reasons: %w", err)
			}
		}
		record.Items = []service.UpstreamAccountHealthGuardRunItem{}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return records, nil
	}

	// The list endpoint keeps full details for the latest run. Older runs only
	// include adjusted accounts so the scheduling adjustment log can span recent
	// runs without returning every checked account.
	latestRunID := records[0].ID
	itemRows, err := r.db.QueryContext(ctx, `
		SELECT
			account_id, account_name, platform, provider_slug, provider_name,
			model_id, schedulable_before, schedulable_after, status, test_status,
			latency_ms, latency_limit_ms, consecutive_failed, consecutive_slow,
			consecutive_healthy, action, reason, error_message, started_at, finished_at
		FROM upstream_account_health_guard_run_items
		WHERE run_id = $1
		ORDER BY provider_slug, account_id, id
	`, latestRunID)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	items := []service.UpstreamAccountHealthGuardRunItem{}
	for itemRows.Next() {
		var item service.UpstreamAccountHealthGuardRunItem
		if err := itemRows.Scan(upstreamAccountHealthGuardRunItemScanDest(&item)...); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}
	records[0].Items = items

	if len(records) <= 1 {
		return records, nil
	}
	recordIndexByID := make(map[string]int, len(records))
	olderRunIDs := make([]string, 0, len(records)-1)
	for index := 1; index < len(records); index++ {
		recordIndexByID[records[index].ID] = index
		olderRunIDs = append(olderRunIDs, records[index].ID)
	}
	query, args := upstreamAccountHealthGuardAdjustedItemsQuery(olderRunIDs)
	adjustedRows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer adjustedRows.Close()

	for adjustedRows.Next() {
		var runID string
		var item service.UpstreamAccountHealthGuardRunItem
		dest := append([]any{&runID}, upstreamAccountHealthGuardRunItemScanDest(&item)...)
		if err := adjustedRows.Scan(dest...); err != nil {
			return nil, err
		}
		index, ok := recordIndexByID[runID]
		if !ok {
			continue
		}
		records[index].Items = append(records[index].Items, item)
	}
	if err := adjustedRows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func upstreamAccountHealthGuardRunItemScanDest(item *service.UpstreamAccountHealthGuardRunItem) []any {
	return []any{
		&item.AccountID,
		&item.AccountName,
		&item.Platform,
		&item.ProviderSlug,
		&item.ProviderName,
		&item.ModelID,
		&item.SchedulableBefore,
		&item.SchedulableAfter,
		&item.Status,
		&item.TestStatus,
		&item.LatencyMs,
		&item.LatencyLimitMs,
		&item.ConsecutiveFailed,
		&item.ConsecutiveSlow,
		&item.ConsecutiveHealthy,
		&item.Action,
		&item.Reason,
		&item.ErrorMessage,
		&item.StartedAt,
		&item.FinishedAt,
	}
}

func upstreamAccountHealthGuardAdjustedItemsQuery(runIDs []string) (string, []any) {
	args := make([]any, 0, len(runIDs))
	placeholders := make([]string, 0, len(runIDs))
	for index, runID := range runIDs {
		args = append(args, runID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
	}
	return fmt.Sprintf(`
		SELECT
			run_id, account_id, account_name, platform, provider_slug, provider_name,
			model_id, schedulable_before, schedulable_after, status, test_status,
			latency_ms, latency_limit_ms, consecutive_failed, consecutive_slow,
			consecutive_healthy, action, reason, error_message, started_at, finished_at
		FROM upstream_account_health_guard_run_items
		WHERE run_id IN (%s)
			AND action IN ('disabled', 'recovered')
		ORDER BY run_id, finished_at DESC, id DESC
	`, strings.Join(placeholders, ",")), args
}
