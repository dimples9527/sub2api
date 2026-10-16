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

// 带 `r.` 前缀：读历史要 LEFT JOIN users 带出发起者邮箱，不带前缀 id 会二义。
const modelDegradationRunColumns = `r.id, r.user_id, r.base_url, r.model_id, r.protocol, r.reasoning_effort,
       r.render_timeout_seconds,
       r.status, r.verdict, r.logic_passed, r.logic_total, r.render_verdict, r.error_message,
       r.duration_ms, r.created_at, r.completed_at, r.group_id, r.group_name`

// 共享历史：不按 user_id 过滤，谁都能读到全部记录（敏感字段由 handler 脱敏后再出网）。
// LEFT JOIN 而不是 INNER JOIN —— 用户被软删后历史记录仍在，
// 用 INNER JOIN 会让这些记录凭空消失。
const modelDegradationRunSelect = `SELECT ` + modelDegradationRunColumns + `, COALESCE(u.email, '')
FROM model_audit_runs r
LEFT JOIN users u ON u.id = r.user_id`

const modelDegradationItemColumns = `id, run_id, task_type, task_key, prompt, expected, response,
       passed, verdict, detail, source_hash, duration_ms, created_at`

func (r *modelDegradationRepository) CreateRun(ctx context.Context, run *service.ModelDegradationRun) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	query := `
INSERT INTO model_audit_runs
    (user_id, base_url, model_id, protocol, reasoning_effort, render_timeout_seconds,
     status, verdict, logic_passed, logic_total,
     render_verdict, error_message, duration_ms, group_id, group_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING id, created_at`

	return r.db.QueryRowContext(ctx, query,
		run.UserID,
		run.BaseURL,
		run.ModelID,
		run.Protocol,
		run.ReasoningEffort,
		run.RenderTimeoutSeconds,
		run.Status,
		run.Verdict,
		run.LogicPassed,
		run.LogicTotal,
		run.RenderVerdict,
		run.ErrorMessage,
		run.DurationMs,
		run.GroupID,
		run.GroupName,
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

// GetRun 读取单次检测及其明细。共享历史：不做归属过滤，任何人都能读。
func (r *modelDegradationRepository) GetRun(ctx context.Context, runID int64) (*service.ModelDegradationRun, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model audit repository database is not initialized")
	}

	run := &service.ModelDegradationRun{}
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		modelDegradationRunSelect+` WHERE r.id = $1`,
		runID,
	).Scan(
		&run.ID,
		&run.UserID,
		&run.BaseURL,
		&run.ModelID,
		&run.Protocol,
		&run.ReasoningEffort,
		&run.RenderTimeoutSeconds,
		&run.Status,
		&run.Verdict,
		&run.LogicPassed,
		&run.LogicTotal,
		&run.RenderVerdict,
		&run.ErrorMessage,
		&run.DurationMs,
		&run.CreatedAt,
		&completedAt,
		&run.GroupID,
		&run.GroupName,
		// 邮箱在 SELECT 里是最后一列（COALESCE(u.email,'')），Scan 顺序要跟 SELECT 一致。
		&run.OwnerEmail,
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

// ListRuns 读取共享的检测历史，按时间倒序。列表不返回明细，详情按需再查。
func (r *modelDegradationRepository) ListRuns(ctx context.Context, limit int) ([]service.ModelDegradationRun, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model audit repository database is not initialized")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx,
		modelDegradationRunSelect+`
ORDER BY r.created_at DESC, r.id DESC
LIMIT $1`,
		limit,
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
			&run.RenderTimeoutSeconds,
			&run.Status,
			&run.Verdict,
			&run.LogicPassed,
			&run.LogicTotal,
			&run.RenderVerdict,
			&run.ErrorMessage,
			&run.DurationMs,
			&run.CreatedAt,
			&completedAt,
			&run.GroupID,
			&run.GroupName,
			// 与 GetRun 同理：邮箱是 SELECT 的最后一列。
			&run.OwnerEmail,
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

// LookupAPIKeyGroup 用检测请求里填的明文 Key 反查平台 API Key 绑定的分组，
// 只认该用户名下的 Key（key 列明文存储，可等值命中）。
// 未命中返回 (nil, "", nil)：填的是第三方上游 Key 属于常态，不是错误。
func (r *modelDegradationRepository) LookupAPIKeyGroup(ctx context.Context, userID int64, apiKey string) (*int64, string, error) {
	if r == nil || r.db == nil {
		return nil, "", fmt.Errorf("model audit repository database is not initialized")
	}

	var groupID sql.NullInt64
	var groupName string
	err := r.db.QueryRowContext(ctx, `
SELECT k.group_id, COALESCE(g.name, '')
FROM api_keys k
LEFT JOIN groups g ON g.id = k.group_id
WHERE k.user_id = $1 AND k.key = $2 AND k.deleted_at IS NULL`,
		userID, apiKey,
	).Scan(&groupID, &groupName)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("lookup api key group: %w", err)
	}
	if !groupID.Valid {
		// Key 命中但未绑分组：分组信息留空。
		return nil, "", nil
	}
	return &groupID.Int64, groupName, nil
}

// DeleteRunsOlderThan 物理删除 created_at 早于 cutoff 的检测记录，单次最多 batchSize 条。
//
// 题目明细不用分开删：model_audit_items.run_id 是 ON DELETE CASCADE 外键。
// 分批而不是一条大 DELETE：一次删掉的行数与事务体积都不可控，会和正常写入抢锁。
// 仍在跑的那一条留着 —— 它的执行协程还活着，删了行只会让它后面 Update 到 0 行。
func (r *modelDegradationRepository) DeleteRunsOlderThan(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("model audit repository database is not initialized")
	}
	if batchSize <= 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
DELETE FROM model_audit_runs
WHERE id IN (
    SELECT id FROM model_audit_runs
    WHERE created_at < $1 AND status <> 'running'
    ORDER BY id
    LIMIT $2
)`, cutoff, batchSize)
	if err != nil {
		return 0, fmt.Errorf("delete model audit runs older than %s: %w", cutoff.Format(time.RFC3339), err)
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted model audit run count: %w", err)
	}
	return deleted, nil
}

// DeleteRunsBeyondKeep 只保留最新的 keep 条，超出的按时间倒序物理删除，单次最多 batchSize 条。
//
// 这是天数闸的兜底：短期高频检测时 30 天也能堆出上万条，而体积几乎全在
// model_audit_items.response（单条最多 256KB）—— 只有条数闸能把磁盘占用钉死。
func (r *modelDegradationRepository) DeleteRunsBeyondKeep(ctx context.Context, keep, batchSize int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("model audit repository database is not initialized")
	}
	if keep <= 0 || batchSize <= 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
DELETE FROM model_audit_runs
WHERE id IN (
    SELECT id FROM model_audit_runs
    WHERE status <> 'running'
    ORDER BY created_at DESC, id DESC
    LIMIT $2 OFFSET $1
)`, keep, batchSize)
	if err != nil {
		return 0, fmt.Errorf("delete model audit runs beyond keep %d: %w", keep, err)
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted model audit run count: %w", err)
	}
	return deleted, nil
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

// MarkStaleRunsFailed 逐行按各自的超时窗口判断失效。
//
// 不能传一个统一的时间点：生成题单题超时是可调的（1~60 分钟），
// 用统一时间点意味着只能按最长的那档算 —— 选了 5 分钟超时的记录要白等一小时才被标失败。
// 这里让每行拿自己的 render_timeout_seconds 加基数比。
func (r *modelDegradationRepository) MarkStaleRunsFailed(ctx context.Context, baseSeconds int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE model_audit_runs
SET status = $1,
    error_message = $2,
    completed_at = NOW()
WHERE status = $3
  AND created_at < NOW() - make_interval(secs => render_timeout_seconds + $4)`,
		service.ModelDegradationStatusFailed,
		"检测超时或服务重启导致中断",
		service.ModelDegradationStatusRunning,
		baseSeconds,
	)
	if err != nil {
		return fmt.Errorf("mark stale model audit runs failed: %w", err)
	}
	return nil
}

// HasActiveRun 判断该用户是否有仍在有效期内的执行中任务，口径与 MarkStaleRunsFailed 一致：
// 逐行按各自的 render_timeout_seconds + baseSeconds 算窗口。
//
// 走 idx_model_audit_runs_user_created，先按用户收敛；窗口右端依赖行内列、不是常量，
// 因此 created_at 上剩下的比较只能逐行做 —— 单个用户的记录很少，代价可忽略。
func (r *modelDegradationRepository) HasActiveRun(ctx context.Context, userID int64, baseSeconds int) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("model audit repository database is not initialized")
	}

	var exists bool
	if err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM model_audit_runs
    WHERE user_id = $1 AND status = $2
      AND created_at > NOW() - make_interval(secs => render_timeout_seconds + $3)
)`,
		userID,
		service.ModelDegradationStatusRunning,
		baseSeconds,
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
