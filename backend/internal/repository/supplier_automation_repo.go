package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierAutomationRepository struct {
	db *sql.DB
}

func NewSupplierAutomationRepository(db *sql.DB) service.SupplierAutomationRepository {
	return &supplierAutomationRepository{db: db}
}

func (r *supplierAutomationRepository) ListTasks(ctx context.Context) ([]service.SupplierAutomationTask, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, task_code, name, enabled, cron_expression, timeout_seconds,
       config_json, last_status, last_message, last_run_at, next_run_at
FROM supplier_automation_tasks
ORDER BY task_code ASC`)
	if err != nil {
		return nil, fmt.Errorf("query supplier automation tasks: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierAutomationTask, 0)
	for rows.Next() {
		item, err := scanSupplierAutomationTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *supplierAutomationRepository) GetTask(ctx context.Context, code string) (*service.SupplierAutomationTask, error) {
	item, err := scanSupplierAutomationTask(r.db.QueryRowContext(ctx, `
SELECT id, task_code, name, enabled, cron_expression, timeout_seconds,
       config_json, last_status, last_message, last_run_at, next_run_at
FROM supplier_automation_tasks
WHERE task_code=$1`, strings.TrimSpace(code)))
	if err == sql.ErrNoRows {
		return nil, service.ErrSupplierProviderInvalid
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *supplierAutomationRepository) UpdateTask(ctx context.Context, task *service.SupplierAutomationTask) error {
	configRaw, err := json.Marshal(task.Config)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
UPDATE supplier_automation_tasks
SET enabled=$2, cron_expression=$3, timeout_seconds=$4, config_json=$5,
    last_status=$6, last_message=$7, last_run_at=$8, next_run_at=$9, updated_at=NOW()
WHERE task_code=$1`, task.TaskCode, task.Enabled, task.CronExpression, task.TimeoutSeconds,
		string(configRaw), task.LastStatus, task.LastMessage, task.LastRunAt, task.NextRunAt)
	return err
}

func (r *supplierAutomationRepository) CreateRun(ctx context.Context, run *service.SupplierAutomationRun) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO supplier_automation_runs (
  task_code, trigger_source, status, message, processed_count,
  success_count, failed_count, started_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id, created_at`, run.TaskCode, run.TriggerSource, run.Status, run.Message,
		run.ProcessedCount, run.SuccessCount, run.FailedCount, run.StartedAt).Scan(&run.ID, &run.CreatedAt)
}

func (r *supplierAutomationRepository) FinishRun(ctx context.Context, run *service.SupplierAutomationRun) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE supplier_automation_runs
SET status=$2, message=$3, processed_count=$4, success_count=$5,
    failed_count=$6, finished_at=$7
WHERE id=$1`, run.ID, run.Status, run.Message, run.ProcessedCount,
		run.SuccessCount, run.FailedCount, run.FinishedAt)
	return err
}

func (r *supplierAutomationRepository) ListRuns(ctx context.Context, params service.SupplierAutomationRunListParams) (service.SupplierAutomationRunListResult, error) {
	params = normalizeSupplierAutomationRunListParams(params)
	where, args := supplierAutomationRunWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_automation_runs WHERE "+where, args...).Scan(&total); err != nil {
		return service.SupplierAutomationRunListResult{}, err
	}
	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT id, task_code, trigger_source, status, message, processed_count,
       success_count, failed_count, started_at, finished_at, created_at
FROM supplier_automation_runs
WHERE `+where+fmt.Sprintf(" ORDER BY started_at DESC, id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return service.SupplierAutomationRunListResult{}, err
	}
	defer rows.Close()
	items := make([]service.SupplierAutomationRun, 0)
	for rows.Next() {
		item, err := scanSupplierAutomationRun(rows)
		if err != nil {
			return service.SupplierAutomationRunListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierAutomationRunListResult{}, err
	}
	return service.SupplierAutomationRunListResult{Items: items, Total: total, Page: params.Page, PageSize: params.PageSize}, nil
}

func (r *supplierAutomationRepository) RecoverRunning(ctx context.Context, message string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE supplier_automation_runs
SET status=$1, message=$2, finished_at=NOW()
WHERE status=$3`, service.SupplierAutomationStatusFailed, message, service.SupplierAutomationStatusRunning)
	return err
}

func supplierAutomationRunWhere(params service.SupplierAutomationRunListParams) (string, []any) {
	conditions := []string{"1=1"}
	args := make([]any, 0, 2)
	if taskCode := strings.TrimSpace(params.TaskCode); taskCode != "" {
		args = append(args, taskCode)
		conditions = append(conditions, fmt.Sprintf("task_code = $%d", len(args)))
	}
	if status := strings.TrimSpace(params.Status); status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

func normalizeSupplierAutomationRunListParams(params service.SupplierAutomationRunListParams) service.SupplierAutomationRunListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 20
	}
	return params
}

type supplierAutomationTaskScanner interface{ Scan(dest ...any) error }

func scanSupplierAutomationTask(scanner supplierAutomationTaskScanner) (service.SupplierAutomationTask, error) {
	var item service.SupplierAutomationTask
	var configRaw string
	var lastRunAt sql.NullTime
	var nextRunAt sql.NullTime
	if err := scanner.Scan(&item.ID, &item.TaskCode, &item.Name, &item.Enabled,
		&item.CronExpression, &item.TimeoutSeconds, &configRaw, &item.LastStatus,
		&item.LastMessage, &lastRunAt, &nextRunAt); err != nil {
		return service.SupplierAutomationTask{}, err
	}
	if strings.TrimSpace(configRaw) != "" {
		_ = json.Unmarshal([]byte(configRaw), &item.Config)
	}
	if lastRunAt.Valid {
		item.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		item.NextRunAt = &nextRunAt.Time
	}
	return item, nil
}

type supplierAutomationRunScanner interface{ Scan(dest ...any) error }

func scanSupplierAutomationRun(scanner supplierAutomationRunScanner) (service.SupplierAutomationRun, error) {
	var item service.SupplierAutomationRun
	var finishedAt sql.NullTime
	if err := scanner.Scan(&item.ID, &item.TaskCode, &item.TriggerSource, &item.Status,
		&item.Message, &item.ProcessedCount, &item.SuccessCount, &item.FailedCount,
		&item.StartedAt, &finishedAt, &item.CreatedAt); err != nil {
		return service.SupplierAutomationRun{}, err
	}
	if finishedAt.Valid {
		item.FinishedAt = &finishedAt.Time
	}
	return item, nil
}
