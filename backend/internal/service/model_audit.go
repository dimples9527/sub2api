package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 检测任务的运行状态。
const (
	ModelAuditStatusRunning   = "running"
	ModelAuditStatusCompleted = "completed"
	ModelAuditStatusFailed    = "failed"
	// ModelAuditStatusCancelled 用户手动中止。与 failed 分开是为了让前端
	// 能区分「你自己按了停止」和「上游出错了」，二者的提示完全不同。
	ModelAuditStatusCancelled = "cancelled"
)

// ErrModelAuditRunInProgress 该用户已有一轮检测在执行。
//
// 执行名额是进程级共享的（modelAuditMaxConcurrent），而单轮最长能占用它
// modelAuditTotalTimeout；不拦并发提交的话，多出来的请求不会失败，而是变成
// 无上限的 goroutine 排队 —— 每个排队协程还攥着用户提交的 API Key。
var ErrModelAuditRunInProgress = errors.New("已有一轮检测正在进行，请等它结束后再发起")

// ErrModelAuditRunNotCancellable 要中止的检测已经不在运行中。
//
// 由 MarkRunCancelled 的条件更新保证：WHERE 里带 status = running，
// 已完结或已被中止过一次的行不会被二次更新。
var ErrModelAuditRunNotCancellable = errors.New("该检测已结束，无需中止")

// ModelAuditCancelledMessage 中止原因。仓储的条件更新与 execute 收尾都写
// 这一句，两处指向同一个常量，避免前端看到两种说法。
const ModelAuditCancelledMessage = "已手动中止"

// 思考等级（reasoning_effort）的可选档位，从浅到深。
//
// 取值与源站 manxue.ai 的下拉一致，也落在本仓既有的 REASONING_EFFORT_LEVELS
// （frontend/src/constants/channel.ts）子集内。空串表示不指定。
var modelAuditReasoningEfforts = []string{"low", "medium", "high", "xhigh", "max"}

// normalizeModelAuditReasoningEffort 归一化思考等级。
//
// 空串合法（表示请求里不带该字段）；其余取值必须是已知档位，
// 否则直接拒绝 —— 把任意字符串透传给上游只会换回一个难懂的上游报错。
func normalizeModelAuditReasoningEffort(raw string) (string, error) {
	effort := strings.ToLower(strings.TrimSpace(raw))
	if effort == "" {
		return "", nil
	}
	for _, allowed := range modelAuditReasoningEfforts {
		if effort == allowed {
			return effort, nil
		}
	}
	return "", fmt.Errorf("思考等级无效：%s（可选 %s）", raw, strings.Join(modelAuditReasoningEfforts, " / "))
}

// 上游协议。三种协议的端点、认证头、请求体与响应体形状都不一样，
// 因此必须随检测记录一起存下来，否则无法解释「同一个模型换了协议之后判定变了」。
const (
	// ModelAuditProtocolChatCompletions OpenAI 兼容的 /chat/completions。
	// 也是本功能在支持多协议之前的唯一协议，因此是默认值。
	ModelAuditProtocolChatCompletions = "chat_completions"
	// ModelAuditProtocolResponses OpenAI Responses API（/responses），源站 manxue.ai 的默认协议。
	ModelAuditProtocolResponses = "responses"
	// ModelAuditProtocolMessages Anthropic Messages API（/messages），Claude 官方端点用这个。
	ModelAuditProtocolMessages = "messages"
)

// modelAuditProtocols 协议可选值，顺序即前端下拉顺序。
var modelAuditProtocols = []string{
	ModelAuditProtocolChatCompletions,
	ModelAuditProtocolResponses,
	ModelAuditProtocolMessages,
}

// normalizeModelAuditProtocol 归一化协议。
//
// 空串落到 chat_completions：这是本功能此前的唯一行为，老请求与老记录都按它解释。
// 其余取值必须是已知协议，否则直接拒绝 —— 把未知协议透传下去只会拼出一个
// 不存在的端点，报错信息还看不出是协议写错了。
func normalizeModelAuditProtocol(raw string) (string, error) {
	protocol := strings.ToLower(strings.TrimSpace(raw))
	if protocol == "" {
		return ModelAuditProtocolChatCompletions, nil
	}
	for _, allowed := range modelAuditProtocols {
		if protocol == allowed {
			return protocol, nil
		}
	}
	return "", fmt.Errorf("协议无效：%s（可选 %s）", raw, strings.Join(modelAuditProtocols, " / "))
}

// ModelAuditRun 一次降智检测。
//
// 注意：这里不保存用户提交的 API Key。凭据只活在当次检测的调用栈里，
// 落库或回传给前端都没有必要，见 migrations/249_model_audit.sql 的说明。
type ModelAuditRun struct {
	ID      int64
	UserID  int64
	BaseURL string
	ModelID string
	// Protocol 本次检测使用的上游协议；取值见 ModelAuditProtocol*，空串按 chat_completions 处理。
	Protocol string
	// ReasoningEffort 本次检测使用的思考等级；空串表示未指定。
	// 取值见 modelAuditReasoningEfforts。
	ReasoningEffort string
	Status          string
	Verdict         string
	LogicPassed     int
	LogicTotal      int
	RenderVerdict   string
	ErrorMessage    string
	DurationMs      int
	CreatedAt       time.Time
	CompletedAt     *time.Time
	Items           []ModelAuditItem
}

// ModelAuditItem 单道题的检测明细。
type ModelAuditItem struct {
	ID         int64
	RunID      int64
	TaskType   string
	TaskKey    string
	Prompt     string
	Expected   string
	Response   string
	Passed     bool
	Verdict    string
	Detail     string
	SourceHash string
	DurationMs int
	CreatedAt  time.Time
}

// ModelAuditQuestion 一道实际要跑的题。
//
// 预制题与自定义题共用这个结构：预制题的 Prompt / Expected 由服务端按 Key 填，
// 自定义题则由调用方直接给。Key 为 modelAuditCustomKey 表示这是自定义题。
type ModelAuditQuestion struct {
	TaskType string
	Key      string
	Prompt   string
	// Expected 逻辑题的期望答案（数字）。生成题不用。
	Expected string
}

// ModelAuditTaskOption 题库里的一道预制题，供前端铺下拉与「查看题目」。
type ModelAuditTaskOption struct {
	TaskType string
	Key      string
	// Title 短标题，给下拉选项用。
	Title string
	// Prompt 题目全文。
	Prompt string
	// Expected 逻辑题的期望答案；生成题为空。
	Expected string
}

// ModelAuditInput 发起一次检测的入参。
type ModelAuditInput struct {
	BaseURL string
	APIKey  string
	ModelID string
	// Protocol 上游协议；留空按 chat_completions 处理。
	// 取值见 ModelAuditProtocol*，非法值会被拒绝。
	Protocol string
	// ReasoningEffort 思考等级；留空表示请求里不带该字段。
	// 取值见 modelAuditReasoningEfforts，非法值会被拒绝。
	ReasoningEffort string
	// Tasks 指定要跑的检测项，为空表示全部。取值见 ModelAuditTaskLogic / ModelAuditTaskRender。
	// 仅在 Questions 为空时生效 —— 那时每个选中的类型跑它的第一道预制题（老调用方的行为）。
	Tasks []string
	// Questions 指定要跑的具体题目（预制题的 Key 或自定义题的正文）。
	// 非空时以它为准，Tasks 被忽略。数量上限见 modelAuditMaxLogicQuestions /
	// modelAuditMaxRenderQuestions。
	Questions []ModelAuditQuestion
}

// ModelAuditRepository 检测记录的持久化接口。
type ModelAuditRepository interface {
	CreateRun(ctx context.Context, run *ModelAuditRun) error
	CreateItems(ctx context.Context, items []ModelAuditItem) error
	FinishRun(ctx context.Context, run *ModelAuditRun) error
	GetRun(ctx context.Context, userID, runID int64) (*ModelAuditRun, error)
	ListRuns(ctx context.Context, userID int64, limit int) ([]ModelAuditRun, error)
	// CountSourceHash 统计该源码指纹在历史记录中已出现的次数，用于识别
	// 「多次检测返回同一份源码」。返回 0 表示这是首次出现。
	CountSourceHash(ctx context.Context, hash string) (int, error)
	// MarkStaleRunsFailed 把超时仍停留在 running 的任务标记为失败。
	// 进程重启会让内存中的执行协程消失，没有这一步这些记录会永远卡在 running。
	MarkStaleRunsFailed(ctx context.Context, startedBefore time.Time) error
	// HasActiveRun 判断该用户是否有仍在有效期内的执行中任务。
	// createdAfter 之前的 running 行视为进程重启的残留、不算占用名额，
	// 否则一次异常退出就会把该用户永久锁死。
	HasActiveRun(ctx context.Context, userID int64, createdAfter time.Time) (bool, error)
	// MarkRunCancelled 把指定检测标记为用户中止。
	// 第二个返回值表示这行是不是「属于自己的、仍在 running」——false 说明
	// 它已经完结或已被中止过，调用方应据此回绝，不要重复更新。
	MarkRunCancelled(ctx context.Context, userID, runID int64) (bool, error)
}
