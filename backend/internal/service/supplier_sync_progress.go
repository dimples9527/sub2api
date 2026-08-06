package service

import (
	"context"
	"regexp"
	"strings"
	"time"
)

var (
	supplierSyncSensitiveAssignmentPattern = regexp.MustCompile(`(?i)(["']?(?:api[_-]?key|client[_-]?key|access[_-]?token|refresh[_-]?token|authorization|cookie|password|secret|token|turnstile)["']?\s*[:=]\s*["']?)([^"'\s,;}\]]+)`)
	supplierSyncBearerPattern              = regexp.MustCompile(`(?i)(\bBearer\s+)[^\s,;]+`)
	supplierSyncSensitiveQueryPattern      = regexp.MustCompile(`(?i)([?&](?:api[_-]?key|client[_-]?key|access[_-]?token|refresh[_-]?token|password|secret|token|turnstile)=)[^&\s]+`)
)

// SupplierSyncProgressStage 表示供应商同步过程中的可观测阶段。
type SupplierSyncProgressStage string

const (
	SupplierSyncProgressStagePrepare  SupplierSyncProgressStage = "prepare"
	SupplierSyncProgressStageCaptcha  SupplierSyncProgressStage = "captcha"
	SupplierSyncProgressStageSession  SupplierSyncProgressStage = "session"
	SupplierSyncProgressStageLogin    SupplierSyncProgressStage = "login"
	SupplierSyncProgressStageAccounts SupplierSyncProgressStage = "accounts"
	SupplierSyncProgressStageGroups   SupplierSyncProgressStage = "groups"
	SupplierSyncProgressStageBalance  SupplierSyncProgressStage = "balance"
	SupplierSyncProgressStageCost     SupplierSyncProgressStage = "cost"
	SupplierSyncProgressStagePersist  SupplierSyncProgressStage = "persist"
	SupplierSyncProgressStageDone     SupplierSyncProgressStage = "done"
	SupplierSyncProgressStageError    SupplierSyncProgressStage = "error"
)

// SupplierSyncProgressEvent 是同步 SSE 推送给管理端的单条进度事件。
// OK 为 nil 表示进行中，为 true 表示成功，为 false 表示失败。
type SupplierSyncProgressEvent struct {
	Stage   SupplierSyncProgressStage `json:"stage"`
	Message string                    `json:"message"`
	OK      *bool                     `json:"ok,omitempty"`
	Time    time.Time                 `json:"time"`
}

// SupplierSyncProgressObserver 接收同步过程中的进度事件。
type SupplierSyncProgressObserver func(SupplierSyncProgressEvent)

type supplierSyncProgressObserverKey struct{}

// WithSupplierSyncProgressObserver 将进度观察器附加到同步 context。
func WithSupplierSyncProgressObserver(ctx context.Context, observer SupplierSyncProgressObserver) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if observer == nil {
		return ctx
	}
	return context.WithValue(ctx, supplierSyncProgressObserverKey{}, observer)
}

func supplierSyncProgressObserverFromContext(ctx context.Context) SupplierSyncProgressObserver {
	if ctx == nil {
		return nil
	}
	observer, _ := ctx.Value(supplierSyncProgressObserverKey{}).(SupplierSyncProgressObserver)
	return observer
}

// SupplierSyncProgress 推送一条进度事件。没有观察器时安全忽略。
func SupplierSyncProgress(ctx context.Context, stage SupplierSyncProgressStage, message string, ok *bool) {
	observer := supplierSyncProgressObserverFromContext(ctx)
	if observer == nil {
		return
	}
	observer(SupplierSyncProgressEvent{
		Stage:   stage,
		Message: strings.TrimSpace(message),
		OK:      ok,
		Time:    time.Now(),
	})
}

// SupplierSyncProgressOK 推送成功事件。
func SupplierSyncProgressOK(ctx context.Context, stage SupplierSyncProgressStage, message string) {
	ok := true
	SupplierSyncProgress(ctx, stage, message, &ok)
}

// SupplierSyncProgressFail 推送失败事件。
func SupplierSyncProgressFail(ctx context.Context, stage SupplierSyncProgressStage, err error) {
	ok := false
	message := "同步失败"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = sanitizeSupplierSyncProgressMessage(err.Error())
	}
	SupplierSyncProgress(ctx, stage, message, &ok)
}

// sanitizeSupplierSyncProgressMessage 清理并限制同步进度中的错误信息，避免把凭据或超长响应正文推送到管理端。
func sanitizeSupplierSyncProgressMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "同步失败"
	}
	message = supplierSyncBearerPattern.ReplaceAllString(message, `${1}[已隐藏]`)
	message = supplierSyncSensitiveAssignmentPattern.ReplaceAllString(message, `${1}[已隐藏]`)
	message = supplierSyncSensitiveQueryPattern.ReplaceAllString(message, `${1}[已隐藏]`)
	runes := []rune(message)
	if len(runes) > 512 {
		message = string(runes[:512]) + "…"
	}
	return message
}

func supplierSyncProgressStageForScope(scope string) SupplierSyncProgressStage {
	switch strings.TrimSpace(scope) {
	case SupplierSyncScopeAccounts:
		return SupplierSyncProgressStageAccounts
	case SupplierSyncScopeGroups:
		return SupplierSyncProgressStageGroups
	case SupplierSyncScopeBalance:
		return SupplierSyncProgressStageBalance
	case SupplierSyncScopeCost:
		return SupplierSyncProgressStageCost
	default:
		return SupplierSyncProgressStageError
	}
}

func supplierSyncProgressScopeLabel(scope string) string {
	switch strings.TrimSpace(scope) {
	case SupplierSyncScopeAccounts:
		return "API Key"
	case SupplierSyncScopeGroups:
		return "分组"
	case SupplierSyncScopeBalance:
		return "余额"
	case SupplierSyncScopeCost:
		return "成本"
	default:
		return "同步数据"
	}
}
