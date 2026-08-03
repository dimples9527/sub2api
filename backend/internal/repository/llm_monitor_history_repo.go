package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type llmMonitorHistoryRepository struct {
	db *sql.DB
}

// NewLLMMonitorHistoryRepository 创建模型监控历史快照仓储。
func NewLLMMonitorHistoryRepository(db *sql.DB) service.LLMMonitorHistoryRepository {
	return &llmMonitorHistoryRepository{db: db}
}

func (r *llmMonitorHistoryRepository) SaveSnapshot(ctx context.Context, snapshot service.LLMMonitorHistorySnapshot) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("模型监控历史仓储未初始化")
	}
	if !json.Valid(snapshot.Payload) {
		return fmt.Errorf("模型监控历史快照不是有效 JSON")
	}
	hash := sha256.Sum256(snapshot.Payload)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO llm_monitor_histories (
			source_key, period, board, payload, payload_hash, captured_at
		) VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		ON CONFLICT (source_key, period, board) DO UPDATE SET
			payload = EXCLUDED.payload,
			payload_hash = EXCLUDED.payload_hash,
			captured_at = EXCLUDED.captured_at
		WHERE llm_monitor_histories.payload_hash IS DISTINCT FROM EXCLUDED.payload_hash
			OR llm_monitor_histories.captured_at < EXCLUDED.captured_at - INTERVAL '1 day'
	`, snapshot.SourceKey, snapshot.Period, snapshot.Board, snapshot.Payload, hex.EncodeToString(hash[:]), snapshot.CapturedAt)
	if err != nil {
		return fmt.Errorf("保存模型监控历史快照失败: %w", err)
	}
	return nil
}

func (r *llmMonitorHistoryRepository) LoadLatestSnapshot(ctx context.Context, sourceKey, period, board string) (*service.LLMMonitorHistorySnapshot, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("模型监控历史仓储未初始化")
	}
	var snapshot service.LLMMonitorHistorySnapshot
	var payload []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT source_key, period, board, payload, captured_at
		FROM llm_monitor_histories
		WHERE source_key = $1 AND period = $2 AND board = $3
		ORDER BY captured_at DESC
		LIMIT 1
	`, sourceKey, period, board).Scan(
		&snapshot.SourceKey,
		&snapshot.Period,
		&snapshot.Board,
		&payload,
		&snapshot.CapturedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取模型监控历史快照失败: %w", err)
	}
	snapshot.Payload = append([]byte(nil), payload...)
	return &snapshot, nil
}

func (r *llmMonitorHistoryRepository) DeleteBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("模型监控历史仓储未初始化")
	}
	if batchSize <= 0 {
		batchSize = 1000
	}
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM llm_monitor_histories
		WHERE id IN (
			SELECT id
			FROM llm_monitor_histories
			WHERE captured_at < $1
			ORDER BY captured_at ASC
			LIMIT $2
		)
	`, cutoff, batchSize)
	if err != nil {
		return 0, fmt.Errorf("删除过期模型监控历史失败: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("读取模型监控历史删除数量失败: %w", err)
	}
	return deleted, nil
}
