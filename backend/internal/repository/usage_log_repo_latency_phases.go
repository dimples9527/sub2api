package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// usage_log_latency_phases 是独立侧边表，刻意不挂在 usage_logs 上：
// usage_logs 的插入依赖 6 份按位置对齐的手工列清单（见 usage_log_repo_insert.go），
// 为一个诊断字段增列的错位风险会污染全部计费日志。
const (
	usageLogLatencyPhasesInsertSQL = `
		INSERT INTO usage_log_latency_phases (
			request_id, api_key_id, build_ms, slot_wait_ms, connect_ms, tls_ms, first_byte_ms, conn_reused
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	usageLogLatencyPhasesSelectSQL = `
		SELECT build_ms, slot_wait_ms, connect_ms, tls_ms, first_byte_ms, conn_reused
		FROM usage_log_latency_phases
		WHERE request_id = $1 AND api_key_id = $2
		ORDER BY id DESC
		LIMIT 1`
)

// CreateLatencyPhases 写入一次成功上游 attempt 的耗时分解。
func (r *usageLogRepository) CreateLatencyPhases(ctx context.Context, requestID string, apiKeyID int64, phases *service.LatencyPhases) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || phases == nil {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, usageLogLatencyPhasesInsertSQL,
		requestID,
		apiKeyID,
		nullableInt(phases.BuildMs),
		nullableInt(phases.SlotWaitMs),
		nullableInt(phases.ConnectMs),
		nullableInt(phases.TLSMs),
		nullableInt(phases.FirstByteMs),
		nullableBool(phases.ConnReused),
	)
	return err
}

// GetLatencyPhases 返回最近一条耗时分解；无记录时返回 (nil, nil)。
func (r *usageLogRepository) GetLatencyPhases(ctx context.Context, requestID string, apiKeyID int64) (*service.LatencyPhases, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, nil
	}
	var (
		buildMs     sql.NullInt64
		slotWaitMs  sql.NullInt64
		connectMs   sql.NullInt64
		tlsMs       sql.NullInt64
		firstByteMs sql.NullInt64
		connReused  sql.NullBool
	)
	err := scanSingleRow(ctx, r.sql, usageLogLatencyPhasesSelectSQL, []any{requestID, apiKeyID},
		&buildMs, &slotWaitMs, &connectMs, &tlsMs, &firstByteMs, &connReused)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &service.LatencyPhases{
		BuildMs:     intFromNull(buildMs),
		SlotWaitMs:  intFromNull(slotWaitMs),
		ConnectMs:   intFromNull(connectMs),
		TLSMs:       intFromNull(tlsMs),
		FirstByteMs: intFromNull(firstByteMs),
		ConnReused:  boolFromNull(connReused),
	}, nil
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableBool(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func intFromNull(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int64)
	return &value
}

func boolFromNull(v sql.NullBool) *bool {
	if !v.Valid {
		return nil
	}
	value := v.Bool
	return &value
}
