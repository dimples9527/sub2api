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
	ModelDegradationStatusRunning   = "running"
	ModelDegradationStatusCompleted = "completed"
	ModelDegradationStatusFailed    = "failed"
	// ModelDegradationStatusCancelled 用户手动中止。与 failed 分开是为了让前端
	// 能区分「你自己按了停止」和「上游出错了」，二者的提示完全不同。
	ModelDegradationStatusCancelled = "cancelled"
)

// ErrModelDegradationRunInProgress 该用户已有一轮检测在执行。
//
// 执行名额是进程级共享的（modelDegradationMaxConcurrent），而单轮最长能占用它
// modelDegradationTotalTimeout；不拦并发提交的话，多出来的请求不会失败，而是变成
// 无上限的 goroutine 排队 —— 每个排队协程还攥着用户提交的 API Key。
var ErrModelDegradationRunInProgress = errors.New("已有一轮检测正在进行，请等它结束后再发起")

// ErrModelDegradationRunNotCancellable 要中止的检测已经不在运行中。
//
// 由 MarkRunCancelled 的条件更新保证：WHERE 里带 status = running，
// 已完结或已被中止过一次的行不会被二次更新。
var ErrModelDegradationRunNotCancellable = errors.New("该检测已结束，无需中止")

// ModelDegradationCancelledMessage 中止原因。仓储的条件更新与 execute 收尾都写
// 这一句，两处指向同一个常量，避免前端看到两种说法。
const ModelDegradationCancelledMessage = "已手动中止"

// 思考等级（reasoning_effort）的可选档位，从浅到深。
//
// 取值与源站 manxue.ai 的下拉一致，也落在本仓既有的 REASONING_EFFORT_LEVELS
// （frontend/src/constants/channel.ts）子集内。空串表示不指定。
var modelDegradationReasoningEfforts = []string{"low", "medium", "high", "xhigh", "max"}

// normalizeModelDegradationReasoningEffort 归一化思考等级。
//
// 空串合法（表示请求里不带该字段）；其余取值必须是已知档位，
// 否则直接拒绝 —— 把任意字符串透传给上游只会换回一个难懂的上游报错。
func normalizeModelDegradationReasoningEffort(raw string) (string, error) {
	effort := strings.ToLower(strings.TrimSpace(raw))
	if effort == "" {
		return "", nil
	}
	for _, allowed := range modelDegradationReasoningEfforts {
		if effort == allowed {
			return effort, nil
		}
	}
	return "", fmt.Errorf("思考等级无效：%s（可选 %s）", raw, strings.Join(modelDegradationReasoningEfforts, " / "))
}

// 上游协议。三种协议的端点、认证头、请求体与响应体形状都不一样，
// 因此必须随检测记录一起存下来，否则无法解释「同一个模型换了协议之后判定变了」。
const (
	// ModelDegradationProtocolChatCompletions OpenAI 兼容的 /chat/completions。
	// 也是本功能在支持多协议之前的唯一协议，因此是默认值。
	ModelDegradationProtocolChatCompletions = "chat_completions"
	// ModelDegradationProtocolResponses OpenAI Responses API（/responses），源站 manxue.ai 的默认协议。
	ModelDegradationProtocolResponses = "responses"
	// ModelDegradationProtocolMessages Anthropic Messages API（/messages），Claude 官方端点用这个。
	ModelDegradationProtocolMessages = "messages"
)

// modelDegradationProtocols 协议可选值，顺序即前端下拉顺序。
var modelDegradationProtocols = []string{
	ModelDegradationProtocolChatCompletions,
	ModelDegradationProtocolResponses,
	ModelDegradationProtocolMessages,
}

// normalizeModelDegradationProtocol 归一化协议。
//
// 空串落到 chat_completions：这是本功能此前的唯一行为，老请求与老记录都按它解释。
// 其余取值必须是已知协议，否则直接拒绝 —— 把未知协议透传下去只会拼出一个
// 不存在的端点，报错信息还看不出是协议写错了。
func normalizeModelDegradationProtocol(raw string) (string, error) {
	protocol := strings.ToLower(strings.TrimSpace(raw))
	if protocol == "" {
		return ModelDegradationProtocolChatCompletions, nil
	}
	for _, allowed := range modelDegradationProtocols {
		if protocol == allowed {
			return protocol, nil
		}
	}
	return "", fmt.Errorf("协议无效：%s（可选 %s）", raw, strings.Join(modelDegradationProtocols, " / "))
}

// ModelDegradationRun 一次降智检测。
//
// 注意：这里不保存用户提交的 API Key。凭据只活在当次检测的调用栈里，
// 落库或回传给前端都没有必要，见 migrations/249_model_audit.sql 的说明。
type ModelDegradationRun struct {
	ID     int64
	UserID int64
	// OwnerEmail 发起者邮箱，由仓储层 LEFT JOIN users 带出，只为共享历史里展示用
	// （对外一律脱敏后再出网）。它不落库，CreateRun 不会写这个字段。
	OwnerEmail string
	// GroupID / GroupName 本次检测填的 Key 若命中发起者本人名下的平台 Key，
	// 就记下它绑定的分组。命中不了（填的是第三方上游 Key）时 GroupID 为 nil、GroupName 为空。
	// GroupName 是发起时的名字快照 —— 分组会被改名或删除，历史记录要保留当时的语义。
	GroupID   *int64
	GroupName string
	BaseURL   string
	ModelID   string
	// Protocol 本次检测使用的上游协议；取值见 ModelDegradationProtocol*，空串按 chat_completions 处理。
	Protocol string
	// ReasoningEffort 本次检测使用的思考等级；空串表示未指定。
	// 取值见 modelDegradationReasoningEfforts。
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
	Items           []ModelDegradationItem
}

// ModelDegradationItem 单道题的检测明细。
type ModelDegradationItem struct {
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

// ModelDegradationQuestion 一道实际要跑的题。
//
// 预制题与自定义题共用这个结构：预制题的 Prompt / Expected 由服务端按 Key 填，
// 自定义题则由调用方直接给。Key 为 modelDegradationCustomKey 表示这是自定义题。
type ModelDegradationQuestion struct {
	TaskType string
	Key      string
	Prompt   string
	// Expected 逻辑题的期望答案（数字）。生成题不用。
	Expected string
}

// ModelDegradationTaskOption 题库里的一道预制题，供前端铺下拉与「查看题目」。
type ModelDegradationTaskOption struct {
	TaskType string
	Key      string
	// Title 短标题，给下拉选项用。
	Title string
	// Prompt 题目全文。
	Prompt string
	// Expected 逻辑题的期望答案；生成题为空。
	Expected string
}

// ModelDegradationInput 发起一次检测的入参。
type ModelDegradationInput struct {
	BaseURL string
	APIKey  string
	ModelID string
	// Protocol 上游协议；留空按 chat_completions 处理。
	// 取值见 ModelDegradationProtocol*，非法值会被拒绝。
	Protocol string
	// ReasoningEffort 思考等级；留空表示请求里不带该字段。
	// 取值见 modelDegradationReasoningEfforts，非法值会被拒绝。
	ReasoningEffort string
	// Tasks 指定要跑的检测项，为空表示全部。取值见 ModelDegradationTaskLogic / ModelDegradationTaskRender。
	// 仅在 Questions 为空时生效 —— 那时每个选中的类型跑它的第一道预制题（老调用方的行为）。
	Tasks []string
	// Questions 指定要跑的具体题目（预制题的 Key 或自定义题的正文）。
	// 非空时以它为准，Tasks 被忽略。数量上限见 modelDegradationMaxLogicQuestions /
	// modelDegradationMaxRenderQuestions。
	Questions []ModelDegradationQuestion
}

// ModelDegradationRepository 检测记录的持久化接口。
type ModelDegradationRepository interface {
	CreateRun(ctx context.Context, run *ModelDegradationRun) error
	CreateItems(ctx context.Context, items []ModelDegradationItem) error
	FinishRun(ctx context.Context, run *ModelDegradationRun) error
	// GetRun / ListRuns 是共享历史：不带 user_id 过滤，任何人都能读到全部检测记录。
	// 读取放开、写入不放开 —— 中止仍由 MarkRunCancelled 的 user_id 过滤拦着。
	GetRun(ctx context.Context, runID int64) (*ModelDegradationRun, error)
	ListRuns(ctx context.Context, limit int) ([]ModelDegradationRun, error)
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
	// LookupAPIKeyGroup 用检测请求里填的明文 Key 反查平台 API Key 绑定的分组，
	// 只认该用户名下的 Key。命中返回 groupID（可为 nil，表示该 Key 未绑分组）与分组名；
	// 未命中返回 (nil, "", nil) —— 填的是第三方上游 Key 属于常态，不算错误。
	LookupAPIKeyGroup(ctx context.Context, userID int64, apiKey string) (*int64, string, error)
	// DeleteRunsOlderThan 物理删除 created_at 早于 cutoff 的历史记录，单次最多 batchSize 条。
	// 返回本次实际删掉的行数（0 表示已经清完）。仍在跑的那一条不删。
	DeleteRunsOlderThan(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
	// DeleteRunsBeyondKeep 只保留最新的 keep 条，超出的按时间倒序物理删除，
	// 单次最多 batchSize 条。天数闸之外的条数兜底。
	DeleteRunsBeyondKeep(ctx context.Context, keep, batchSize int) (int64, error)
}

// ModelDegradationHistoryMaintainer 是每日运维清理任务需要的检测历史清理能力。
// 检测历史是共享池子、且题目明细里存着整份模型输出（单条最多 256KB），
// 不清理会一直涨，所以挂进 OpsCleanupService 的 cron（与模型监控历史同一套路）。
type ModelDegradationHistoryMaintainer interface {
	PruneHistory(ctx context.Context) error
}
