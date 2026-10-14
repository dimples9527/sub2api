package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// modelDegradationRepository 上游模型降智检测的持久化实现。
type modelDegradationRepository struct {
	db *sql.DB
}

// NewModelDegradationRepository 创建降智检测仓储。
func NewModelDegradationRepository(db *sql.DB) service.ModelDegradationRepository {
	return &modelDegradationRepository{db: db}
}

const modelDegradationRunColumns = `id, user_id, base_url, model_id, protocol, reasoning_effort, status, verdict,
       logic_passed, logic_total, render_verdict, error_message, duration_ms,
       created_at, completed_at`

const modelDegradationItemColumns = `id, run_id, task_type, task_key, prompt, expected, response,
       passed, verdict, detail, source_hash, duration_ms, created_at`

func (r *modelDegradationRepository) CreateRun(ctx context.Context, run *service.ModelDegradationRun) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	query := `
INSERT INTO model_audit_runs
    (user_id, base_url, model_id, protocol, reasoning_effort, status, verdict, logic_passed, logic_total,
     render_verdict, error_message, duration_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		run.UserID,
		run.BaseURL,
		run.ModelID,
		run.Protocol,
		run.ReasoningEffort,
		run.Status,
		run.Verdict,
		run.LogicPassed,
		run.LogicTotal,
		run.RenderVerdict,
		run.ErrorMessage,
		run.DurationMs,
	).Scan(&run.ID, &run.CreatedAt)
}

func (r *modelDegradationRepository) CreateItems(ctx context.Context, items []service.ModelDegradationItem) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin model audit items transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO model_audit_items
    (run_id, task_type, task_key, prompt, expected, response,
     passed, verdict, detail, source_hash, duration_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)
	if err != nil {
		return fmt.Errorf("prepare model audit items insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for i := range items {
		item := &items[i]
		if _, err := stmt.ExecContext(ctx,
			item.RunID,
			item.TaskType,
			item.TaskKey,
			item.Prompt,
			item.Expected,
			item.Response,
			item.Passed,
			item.Verdict,
			item.Detail,
			item.SourceHash,
			item.DurationMs,
		); err != nil {
			return fmt.Errorf("insert model audit item %s: %w", item.TaskKey, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit model audit items: %w", err)
	}
	return nil
}

func (r *modelDegradationRepository) FinishRun(ctx context.Context, run *service.ModelDegradationRun) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE model_audit_runs
SET status = $1,
    verdict = $2,
    logic_passed = $3,
    logic_total = $4,
    render_verdict = $5,
    error_message = $6,
    duration_ms = $7,
    completed_at = NOW()
WHERE id = $8`,
		run.Status,
		run.Verdict,
		run.LogicPassed,
		run.LogicTotal,
		run.RenderVerdict,
		run.ErrorMessage,
		run.DurationMs,
		run.ID,
	)
	if err != nil {
		return fmt.Errorf("finish model audit run %d: %w", run.ID, err)
	}
	return nil
}

// GetRun 读取单次检测及其明细。user_id 参与过滤，避免越权读取他人记录。
func (r *modelDegradationRepository) GetRun(ctx context.Context, userID, runID int64) (*service.ModelDegradationRun, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model audit repository database is not initialized")
	}

	run := &service.ModelDegradationRun{}
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT `+modelDegradationRunColumns+` FROM model_audit_runs WHERE id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(
		&run.ID,
		&run.UserID,
		&run.BaseURL,
		&run.ModelID,
		&run.Protocol,
		&run.ReasoningEffort,
		&run.Status,
		&run.Verdict,
		&run.LogicPassed,
		&run.LogicTotal,
		&run.RenderVerdict,
		&run.ErrorMessage,
		&run.DurationMs,
		&run.CreatedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model audit run %d: %w", runID, err)
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	items, err := r.listItems(ctx, runID)
	if err != nil {
		return nil, err
	}
	run.Items = items
	return run, nil
}

// ListRuns 读取用户的检测历史，按时间倒序。列表不返回明细，详情按需再查。
func (r *modelDegradationRepository) ListRuns(ctx context.Context, userID int64, limit int) ([]service.ModelDegradationRun, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model audit repository database is not initialized")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+modelDegradationRunColumns+` FROM model_audit_runs
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list model audit runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	runs := make([]service.ModelDegradationRun, 0, limit)
	for rows.Next() {
		var run service.ModelDegradationRun
		var completedAt sql.NullTime
		if err := rows.Scan(
			&run.ID,
			&run.UserID,
			&run.BaseURL,
			&run.ModelID,
			&run.Protocol,
			&run.ReasoningEffort,
			&run.Status,
			&run.Verdict,
			&run.LogicPassed,
			&run.LogicTotal,
			&run.RenderVerdict,
			&run.ErrorMessage,
			&run.DurationMs,
			&run.CreatedAt,
			&completedAt,
		); err != nil {
			return nil, fmt.Errorf("scan model audit run: %w", err)
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model audit runs: %w", err)
	}
	return runs, nil
}

func (r *modelDegradationRepository) listItems(ctx context.Context, runID int64) ([]service.ModelDegradationItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+modelDegradationItemColumns+` FROM model_audit_items WHERE run_id = $1 ORDER BY id`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list model audit items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.ModelDegradationItem, 0, 8)
	for rows.Next() {
		var item service.ModelDegradationItem
		if err := rows.Scan(
			&item.ID,
			&item.RunID,
			&item.TaskType,
			&item.TaskKey,
			&item.Prompt,
			&item.Expected,
			&item.Response,
			&item.Passed,
			&item.Verdict,
			&item.Detail,
			&item.SourceHash,
			&item.DurationMs,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan model audit item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model audit items: %w", err)
	}
	return items, nil
}

func (r *modelDegradationRepository) CountSourceHash(ctx context.Context, hash string) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("model audit repository database is not initialized")
	}
	if hash == "" {
		return 0, nil
	}

	var count int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM model_audit_items WHERE source_hash = $1`, hash,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count model audit source hash: %w", err)
	}
	return count, nil
}

func (r *modelDegradationRepository) MarkStaleRunsFailed(ctx context.Context, startedBefore time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE model_audit_runs
SET status = $1,
    error_message = $2,
    completed_at = NOW()
WHERE status = $3 AND created_at < $4`,
		service.ModelDegradationStatusFailed,
		"检测超时或服务重启导致中断",
		service.ModelDegradationStatusRunning,
		startedBefore,
	)
	if err != nil {
		return fmt.Errorf("mark stale model audit runs failed: %w", err)
	}
	return nil
}

func (r *modelDegradationRepository) HasActiveRun(ctx context.Context, userID int64, createdAfter time.Time) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("model audit repository database is not initialized")
	}

	// 走 idx_model_audit_runs_user_created，先按用户收敛再比时间；
	// 单个用户的记录很少，status 不建索引也无所谓。
	var exists bool
	if err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM model_audit_runs
    WHERE user_id = $1 AND status = $2 AND created_at > $3
)`,
		userID,
		service.ModelDegradationStatusRunning,
		createdAfter,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active model audit run: %w", err)
	}
	return exists, nil
}

// MarkRunCancelled 把指定的检测标记为用户中止。
//
// WHERE 里同时带 user_id 和 status = running：前者防止越权中止别人的记录，
// 后者让「已完结 / 已中止过」的行返回 (false, nil) 而不是被二次覆盖。
// 放进同一条 UPDATE 里判断，就不需要先 SELECT 再 UPDATE 的事务。
func (r *modelDegradationRepository) MarkRunCancelled(ctx context.Context, userID, runID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("model audit repository database is not initialized")
	}

	res, err := r.db.ExecContext(ctx, `
UPDATE model_audit_runs
SET status = $1,
    verdict = $2,
    error_message = $3,
    completed_at = NOW()
WHERE id = $4 AND user_id = $5 AND status = $6`,
		service.ModelDegradationStatusCancelled,
		service.ModelDegradationVerdictUnknown,
		service.ModelDegradationCancelledMessage,
		runID,
		userID,
		service.ModelDegradationStatusRunning,
	)
	if err != nil {
		return false, fmt.Errorf("cancel model audit run %d: %w", runID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("cancel model audit run %d: %w", runID, err)
	}
	return affected > 0, nil
}
