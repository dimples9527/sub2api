package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"go.uber.org/zap"
)

const (
	// 逻辑题的单题超时。回答只有一句话，60s 足够。
	modelAuditPerTaskTimeout = 60 * time.Second
	// 生成题的单题超时。这题要吐一整个 HTML 文档，实测上游 60s 根本出不来，
	// 给到 6 分钟才保险。
	modelAuditRenderTaskTimeout = 6 * time.Minute
	// 单次检测的整体上限 = 1 道逻辑题 + 1 道生成题（每类各一道，见
	// modelAuditMaxLogicQuestions / modelAuditMaxRenderQuestions），
	// 再加 1 分钟收尾余量。必须严格大于各题上限之和，否则生成题
	// 会被整体上限提前掐断。
	modelAuditTotalTimeout = modelAuditPerTaskTimeout + modelAuditRenderTaskTimeout + time.Minute
	// 同时执行的检测任务数上限，避免批量发起打满上游与本地连接。
	modelAuditMaxConcurrent = 4
	// 上游响应体读取上限，防止异常上游返回超大内容拖垮内存。
	modelAuditMaxResponseBytes = 256 << 10
	// 错误信息里回显上游响应片段的长度上限。
	modelAuditErrorSnippetLimit = 300
	// 列表查询默认与最大条数。
	modelAuditDefaultListLimit = 20
	modelAuditMaxListLimit     = 100
	// 判定 running 任务已失效的宽限期。
	modelAuditStaleGrace = time.Minute
	// 读取上游模型列表的超时。这个请求比检测题轻得多。
	modelAuditModelsTimeout = 20 * time.Second
	// Anthropic Messages 必须显式声明 API 版本，缺失会直接 400。
	modelAuditAnthropicVersion = "2023-06-01"
	// Anthropic Messages 的 max_tokens 是必填项。检测题的回答都很短，给足即可。
	modelAuditAnthropicMaxTokens = 4096
	// 各协议在 Base URL 之后要拼的终端路径。
	modelAuditChatCompletionsPath = "/chat/completions"
	modelAuditResponsesPath       = "/responses"
	modelAuditMessagesPath        = "/messages"
	// 模型列表路径。OpenAI 与 Anthropic 都是 {base}/v1/models。
	modelAuditModelsPath = "/models"
)

// modelAuditTerminalPaths 已知的终端路径，用来剥掉用户可能直接粘进来的完整端点。
// 顺序无关：这几个后缀互不为前缀。
var modelAuditTerminalPaths = []string{
	modelAuditChatCompletionsPath,
	modelAuditResponsesPath,
	modelAuditMessagesPath,
	modelAuditModelsPath,
}

// modelAuditEndpointPaths 依据用户填的 Base URL 与协议终端路径给出候选端点，按可能性从高到低。
//
// 用户通常只填一个域名（连 /v1 都不带），而各家网关的路径约定并不一致：
// OpenAI 与 Anthropic 的官方端点都是 {base}/v1/{path}，但也有网关只暴露 {base}/{path}。
// 因此先做归一化 —— 路径里没有版本段就补 /v1 —— 再给出「不补 /v1」的兜底候选，
// 由调用方在 404/405 时换下一个候选重试。
//
// 这条兜底不是可有可无的：光靠归一化，用户没有任何办法表达「这个网关就是不要 /v1」。
func modelAuditEndpointPaths(baseURL, terminalPath string) []string {
	raw := strings.TrimSpace(baseURL)

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return []string{strings.TrimRight(raw, "/") + terminalPath}
	}

	// 丢掉 query / fragment：端点路径必须拼在它们前面。
	// 同时剥掉用户可能粘进来的完整终端路径（…/v1/chat/completions），
	// 这样中途改协议也不用重新填地址。
	path := strings.TrimRight(parsed.Path, "/")
	for _, known := range modelAuditTerminalPaths {
		if strings.HasSuffix(path, known) {
			path = strings.TrimRight(strings.TrimSuffix(path, known), "/")
			break
		}
	}

	base := parsed.Scheme + "://" + parsed.Host + path

	// 地址里已经带了版本段（…/v1、…/openai/v1）就不再自作主张补一次。
	if modelAuditPathHasVersion(path) {
		return []string{base + terminalPath}
	}
	return []string{base + "/v1" + terminalPath, base + terminalPath}
}

// modelAuditPathHasVersion 判断路径最后一段是否是版本段（v1 / v2 / v1beta / v1alpha …）。
func modelAuditPathHasVersion(path string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	last := segments[len(segments)-1]
	if len(last) < 2 || (last[0] != 'v' && last[0] != 'V') {
		return false
	}
	return last[1] >= '0' && last[1] <= '9'
}

// modelAuditPathMissing 判断状态码是否表示「这个路径不存在」。
// 只有这种情况才值得换下一个候选端点重试。
func modelAuditPathMissing(status int) bool {
	return status == http.StatusNotFound || status == http.StatusMethodNotAllowed
}

// ModelAuditService 上游模型降智检测服务。
//
// 凭据（API Key）由用户在每次检测时提交，只存在于当次调用栈中：
// 不落库、不回传、不进日志。
type ModelAuditService struct {
	repo ModelAuditRepository
	// sem 限制并发执行的检测任务数；拿不到名额的任务排队等待而不是直接失败。
	sem chan struct{}
	// cancels 登记每轮执行协程的取消函数：runID → context.CancelFunc。
	// 中止时既断掉正在发出的上游请求，也让 execute 的循环提前收尾；
	// 直接改库状态是不够的 —— 协程不知道状态变了，还会把剩下的题继续跑完。
	cancels sync.Map
}

// NewModelAuditService 创建降智检测服务。
func NewModelAuditService(repo ModelAuditRepository) *ModelAuditService {
	return &ModelAuditService{
		repo: repo,
		sem:  make(chan struct{}, modelAuditMaxConcurrent),
	}
}

// Start 创建一次检测任务并异步执行，立即返回任务记录供前端轮询。
func (s *ModelAuditService) Start(ctx context.Context, userID int64, input ModelAuditInput) (*ModelAuditRun, error) {
	normalized, err := s.validateInput(input)
	if err != nil {
		return nil, err
	}

	// 同一用户同一时间只允许一轮检测。
	//
	// 执行名额（modelAuditMaxConcurrent）是进程级共享的，而单轮最长能占用它
	// modelAuditTotalTimeout；放开并发提交的话，多出来的请求不会失败，而是变成
	// 无上限的 goroutine 排队（每个排队协程还攥着用户提交的 API Key）。
	//
	// 只看「有效期内的」running：进程上次异常退出留下的行不算占用，否则一次崩溃
	// 就会把该用户永久锁死。这里没有事务，两个同时到达的请求理论上可能都通过检查
	// —— 代价只是退回「两轮并行」，比加唯一索引动迁移划算。
	active, err := s.repo.HasActiveRun(ctx, userID, time.Now().Add(-(modelAuditTotalTimeout + modelAuditStaleGrace)))
	if err != nil {
		return nil, err
	}
	if active {
		return nil, ErrModelAuditRunInProgress
	}

	run := &ModelAuditRun{
		UserID:          userID,
		BaseURL:         normalized.BaseURL,
		ModelID:         normalized.ModelID,
		Protocol:        normalized.Protocol,
		ReasoningEffort: normalized.ReasoningEffort,
		Status:          ModelAuditStatusRunning,
		Verdict:         ModelAuditVerdictUnknown,
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	// 刻意使用 Background：检测要在请求返回后继续跑完，
	// 不能随用户那次 HTTP 请求的 ctx 一起被取消。
	go s.execute(context.Background(), run, normalized)

	return run, nil
}

// Get 读取单次检测详情。返回 nil 表示记录不存在或不属于该用户。
func (s *ModelAuditService) Get(ctx context.Context, userID, runID int64) (*ModelAuditRun, error) {
	run, err := s.repo.GetRun(ctx, userID, runID)
	if err != nil || run == nil {
		return run, err
	}

	// 进程重启会让执行协程消失，这类记录会永远停在 running。
	// 读取时兜底标记，避免前端一直转圈。
	if run.Status == ModelAuditStatusRunning && time.Since(run.CreatedAt) > modelAuditTotalTimeout+modelAuditStaleGrace {
		if err := s.repo.MarkStaleRunsFailed(ctx, time.Now().Add(-(modelAuditTotalTimeout + modelAuditStaleGrace))); err != nil {
			logger.FromContext(ctx).Warn("model_audit.mark_stale_failed", zap.Error(err))
		}
		run.Status = ModelAuditStatusFailed
		run.ErrorMessage = "检测超时或服务重启导致中断"
	}
	return run, nil
}

// List 读取用户的检测历史。
func (s *ModelAuditService) List(ctx context.Context, userID int64, limit int) ([]ModelAuditRun, error) {
	if limit <= 0 {
		limit = modelAuditDefaultListLimit
	}
	if limit > modelAuditMaxListLimit {
		limit = modelAuditMaxListLimit
	}
	return s.repo.ListRuns(ctx, userID, limit)
}

// validateModelAuditCredentials 校验并归一化用户提交的上游凭据。
//
// 强制 HTTPS 且拒绝私网/回环地址：这个 URL 由服务端发起请求，
// 不校验等于给用户开了一个内网探测入口。
func validateModelAuditCredentials(rawBaseURL, rawAPIKey string) (string, string, error) {
	baseURL := strings.TrimSpace(rawBaseURL)
	apiKey := strings.TrimSpace(rawAPIKey)
	if baseURL == "" {
		return "", "", fmt.Errorf("请填写上游 Base URL")
	}
	if apiKey == "" {
		return "", "", fmt.Errorf("请填写上游 API Key")
	}

	validated, err := urlvalidator.ValidateHTTPURL(baseURL, false, urlvalidator.ValidationOptions{AllowPrivate: false})
	if err != nil {
		return "", "", fmt.Errorf("Base URL 无效：%w", err)
	}
	return validated, apiKey, nil
}

// validateInput 校验并归一化用户提交的入参，返回可直接使用的入参副本。
func (s *ModelAuditService) validateInput(input ModelAuditInput) (ModelAuditInput, error) {
	modelID := strings.TrimSpace(input.ModelID)
	if modelID == "" {
		return ModelAuditInput{}, fmt.Errorf("请填写模型名称")
	}

	baseURL, apiKey, err := validateModelAuditCredentials(input.BaseURL, input.APIKey)
	if err != nil {
		return ModelAuditInput{}, err
	}

	protocol, err := normalizeModelAuditProtocol(input.Protocol)
	if err != nil {
		return ModelAuditInput{}, err
	}

	effort, err := normalizeModelAuditReasoningEffort(input.ReasoningEffort)
	if err != nil {
		return ModelAuditInput{}, err
	}

	// 题目在入参校验阶段就定下来：这样「题号不存在 / 没填期望答案」这类错误
	// 会在建 run 记录之前以 400 返回，而不是变成一个跑完才发现失败的记录。
	questions, err := resolveModelAuditQuestions(input.Tasks, input.Questions)
	if err != nil {
		return ModelAuditInput{}, err
	}

	return ModelAuditInput{
		BaseURL:         baseURL,
		APIKey:          apiKey,
		ModelID:         modelID,
		Protocol:        protocol,
		ReasoningEffort: effort,
		Tasks:           input.Tasks,
		Questions:       questions,
	}, nil
}

// ListTasks 返回内置题库，供前端铺下拉与「查看题目」。
//
// 题库是编译期常量、与用户无关，所以不需要凭据也不需要 ctx。
func (s *ModelAuditService) ListTasks() (logic, render []ModelAuditTaskOption) {
	return ModelAuditLogicTaskOptions(), ModelAuditRenderTaskOptions()
}

// execute 串行跑完全部检测项并落库。
func (s *ModelAuditService) execute(ctx context.Context, run *ModelAuditRun, input ModelAuditInput) {
	defer func() {
		if r := recover(); r != nil {
			logger.FromContext(ctx).Error("model_audit.panic", zap.Any("recover", r), zap.Int64("run_id", run.ID))
			run.Status = ModelAuditStatusFailed
			run.ErrorMessage = "检测执行异常"
			if err := s.repo.FinishRun(context.Background(), run); err != nil {
				logger.FromContext(ctx).Warn("model_audit.finish_failed", zap.Error(err))
			}
		}
	}()

	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	ctx, cancel := context.WithTimeout(ctx, modelAuditTotalTimeout)
	defer cancel()

	// 再包一层可取消的 ctx 并登记，用户按「停止」时能在下一道题之前收尾。
	// 为什么不在 runLogicTask 内部逐个判断：那些函数拿到 ctx 就会用，
	// 中止只需要保证「不再发起新请求」，在循环入口判断一次就够。
	ctx, cancelRun := context.WithCancel(ctx)
	s.cancels.Store(run.ID, cancelRun)
	defer func() {
		cancelRun()
		s.cancels.Delete(run.ID)
	}()

	started := time.Now()
	items := make([]ModelAuditItem, 0, len(input.Questions))

	// 每道题一跑完就落库，而不是全部跑完再一次性写。
	// 前端是按固定间隔轮询 /runs/:id 的，只有边跑边写才能逐题看到进度；
	// 顺带的好处是中途超时或失败时，已经跑完的题仍然查得到。
	record := func(item ModelAuditItem) {
		item.RunID = run.ID
		items = append(items, item)
		if err := s.repo.CreateItems(ctx, []ModelAuditItem{item}); err != nil {
			logger.FromContext(ctx).Warn("model_audit.create_item_failed",
				zap.Error(err), zap.Int64("run_id", run.ID), zap.String("task_key", item.TaskKey))
		}
	}

	// 已中止就不再起新请求：已经跑完的题照样算结果，这一轮标记成 cancelled。
	cancelled := false

	// 要跑哪些题在 validateInput 里就解析好了（预制题按 Key 展开、自定义题按正文），
	// 这里只负责按顺序跑；顺序是逻辑题在前、生成题在后。
	for _, question := range input.Questions {
		if ctx.Err() != nil {
			cancelled = true
			break
		}
		if question.TaskType == ModelAuditTaskLogic {
			record(s.runLogicTask(ctx, input, question))
			continue
		}
		record(s.runRenderTask(ctx, input, question))
	}

	run.DurationMs = int(time.Since(started).Milliseconds())
	run.LogicPassed, run.LogicTotal, run.RenderVerdict = summarizeModelAuditItems(items)
	if cancelled {
		// 中止是「用户主动结束」，verdict 不该跟着算成 normal / degraded
		// 和历史记录里的自动判定混在一起；已跑题的结果仍在明细里看得到。
		run.Status = ModelAuditStatusCancelled
		run.Verdict = ModelAuditVerdictUnknown
		run.ErrorMessage = ModelAuditCancelledMessage
	} else {
		run.Verdict = summarizeModelAuditVerdict(run.LogicPassed, run.LogicTotal, run.RenderVerdict, items)
		run.Status = ModelAuditStatusCompleted
	}

	if err := s.repo.FinishRun(ctx, run); err != nil {
		logger.FromContext(ctx).Warn("model_audit.finish_failed", zap.Error(err), zap.Int64("run_id", run.ID))
	}
}

// CancelRun 中止用户自己的一轮检测。
//
// 成功路径是「先断请求、再收尾」：cancel 让 execute 的 ctx 立刻取消，
// 正在进行中的上游调用随之失败，循环在下一道题前跳出，最后由 FinishRun
// 把状态写成 cancelled。落库用带 status = running 的条件更新，重复点停止
// （或点到已完结的记录）只会拿到 ErrModelAuditRunNotCancellable，不会覆盖结果。
//
// 取消函数不在进程内时（服务重启后残留的 running 行），退化成只把库里的行
// 标记掉 —— 那种情况下本来也没有协程在跑。
func (s *ModelAuditService) CancelRun(ctx context.Context, userID, runID int64) error {
	cancelled, err := s.repo.MarkRunCancelled(ctx, userID, runID)
	if err != nil {
		return err
	}
	if !cancelled {
		return ErrModelAuditRunNotCancellable
	}

	if v, loaded := s.cancels.LoadAndDelete(runID); loaded {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
	return nil
}

// runLogicTask 跑一道逻辑题。
func (s *ModelAuditService) runLogicTask(ctx context.Context, input ModelAuditInput, question ModelAuditQuestion) ModelAuditItem {
	item := ModelAuditItem{
		TaskType: ModelAuditTaskLogic,
		TaskKey:  question.Key,
		Prompt:   question.Prompt,
		Expected: question.Expected,
	}

	started := time.Now()
	response, err := s.callUpstream(ctx, input, question.Prompt, modelAuditPerTaskTimeout)
	item.DurationMs = int(time.Since(started).Milliseconds())

	if err != nil {
		item.Verdict = ModelAuditItemUnknown
		item.Detail = "上游请求失败：" + err.Error()
		return item
	}

	item.Response = response
	passed, detail := judgeModelAuditLogic(response, question.Expected)
	item.Passed = passed
	item.Detail = detail
	if passed {
		item.Verdict = ModelAuditItemOK
	} else {
		item.Verdict = ModelAuditItemDegraded
	}
	return item
}

// runRenderTask 跑生成题，并对源码指纹做查重。
func (s *ModelAuditService) runRenderTask(ctx context.Context, input ModelAuditInput, question ModelAuditQuestion) ModelAuditItem {
	item := ModelAuditItem{
		TaskType: ModelAuditTaskRender,
		TaskKey:  question.Key,
		Prompt:   question.Prompt,
	}

	started := time.Now()
	response, err := s.callUpstream(ctx, input, question.Prompt, modelAuditRenderTaskTimeout)
	item.DurationMs = int(time.Since(started).Milliseconds())

	if err != nil {
		item.Verdict = ModelAuditItemUnknown
		item.Detail = "上游请求失败：" + err.Error()
		return item
	}

	item.Response = response
	// 生成题不再做降智判定：只生成、展示作品，是否「降智」交由用户自己看作品判断。
	item.Verdict = ModelAuditItemGenerated
	item.Passed = false
	return item
}

// modelAuditUpstreamRequest 一次上游调用的完整描述。
type modelAuditUpstreamRequest struct {
	// Paths 候选端点，按可能性从高到低；调用方在 404/405 时换下一个。
	Paths   []string
	Headers map[string]string
	Body    []byte
	// Extract 从 2xx 响应体里取出回答正文。
	//
	// 它同时充当「这个端点确实在按本协议服务」的判据：部分网关（实测有）对未知
	// 路由也回 HTTP 200，只在 body 里写 {"code":404,...}。只看状态码会把这种
	// 错误响应当成成功，所以取不出正文时要当作该候选不可用，继续试下一个。
	Extract func(raw []byte) (string, error)
}

// buildModelAuditUpstreamRequest 按协议构造一次上游调用。
//
// 端点、认证头、请求体、响应体形状在三种协议之间都不一样，差异全部收敛在这里，
// callUpstream 只负责发请求与读响应：
//   - chat_completions：POST {base}/v1/chat/completions，Authorization: Bearer，取 choices[0].message.content
//   - responses：       POST {base}/v1/responses，Authorization: Bearer，取 output[].content[].text
//   - messages：        POST {base}/v1/messages，x-api-key + anthropic-version，取 content[].text
//
// {base} 由 modelAuditEndpointPaths 归一化：用户只填域名时自动补 /v1。
func buildModelAuditUpstreamRequest(input ModelAuditInput, prompt string) (modelAuditUpstreamRequest, error) {
	headers := map[string]string{"Content-Type": "application/json"}

	var (
		terminalPath string
		payload      map[string]any
		extract      func(raw []byte) (string, error)
	)

	switch input.Protocol {
	case ModelAuditProtocolMessages:
		terminalPath = modelAuditMessagesPath
		headers["x-api-key"] = input.APIKey
		headers["anthropic-version"] = modelAuditAnthropicVersion
		payload = map[string]any{
			"model":      input.ModelID,
			"max_tokens": modelAuditAnthropicMaxTokens,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		// Anthropic 没有 reasoning_effort：它用 thinking.budget_tokens 表达推理深度，
		// 档位语义与 OpenAI 的 low/medium/high 不可换算；而且多传一个它不认识的字段
		// 会直接 400（extra inputs are not permitted）。因此这里刻意不发该字段。
		extract = extractModelAuditMessagesContent

	case ModelAuditProtocolResponses:
		terminalPath = modelAuditResponsesPath
		headers["Authorization"] = "Bearer " + input.APIKey
		payload = map[string]any{
			"model":  input.ModelID,
			"input":  prompt,
			"stream": false,
		}
		if input.ReasoningEffort != "" {
			payload["reasoning"] = map[string]string{"effort": input.ReasoningEffort}
		}
		extract = extractModelAuditResponsesContent

	default: // ModelAuditProtocolChatCompletions
		terminalPath = modelAuditChatCompletionsPath
		headers["Authorization"] = "Bearer " + input.APIKey
		payload = map[string]any{
			"model": input.ModelID,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream": false,
		}
		// 未指定推理程度时不带该字段：上游未必接受这个参数，
		// 无谓地传一个值只会把「上游不支持」变成一次莫名其妙的 400。
		if input.ReasoningEffort != "" {
			payload["reasoning_effort"] = input.ReasoningEffort
		}
		extract = extractModelAuditChatContent
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return modelAuditUpstreamRequest{}, fmt.Errorf("构造请求体失败")
	}

	return modelAuditUpstreamRequest{
		Paths:   modelAuditEndpointPaths(input.BaseURL, terminalPath),
		Headers: headers,
		Body:    encoded,
		Extract: extract,
	}, nil
}

// requestModelAuditUpstream 依次尝试候选端点，返回第一个「路径存在且内容可用」的响应体。
//
// 判定「这个候选不行」分两种：
//   - 状态码 404/405：路径不存在，换下一个；
//   - 2xx 但 accept 解析失败：路径存在、内容却不是本协议该有的东西
//     （有些网关对未知路由也回 200），同样换下一个。
//
// 其余状态码（401/400/429/5xx）说明路径是通的且上游真的在拒绝我们，
// 换下一个候选只会掩盖真正的原因，因此立刻返回。
// 全部候选都不可用时，返回**第一个**候选的错误 —— 那是我们最相信的路径。
func (s *ModelAuditService) requestModelAuditUpstream(
	ctx context.Context,
	method string,
	paths []string,
	headers map[string]string,
	body []byte,
	accept func(raw []byte) error,
	timeout time.Duration,
) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// ValidateResolvedIP 打开 DNS 解析后的地址校验，堵住 DNS rebinding：
	// 只校验 URL 字面量的话，攻击者可以用一个解析到内网的域名绕过。
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:            timeout,
		ValidateResolvedIP: true,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 客户端失败")
	}

	var firstErr error
	for _, endpoint := range paths {
		// 每次都要新建 reader：上一次尝试已经把 body 读掉了。
		var reader io.Reader
		if len(body) > 0 {
			reader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(reqCtx, method, endpoint, reader)
		if err != nil {
			return nil, fmt.Errorf("构造请求失败")
		}
		for name, value := range headers {
			req.Header.Set(name, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			// 网络层失败换路径没意义，直接返回。
			return nil, fmt.Errorf("请求上游失败：%s", err.Error())
		}

		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, modelAuditMaxResponseBytes))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("读取上游响应失败")
		}

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			if accept == nil {
				return raw, nil
			}
			acceptErr := accept(raw)
			if acceptErr == nil {
				return raw, nil
			}
			if firstErr == nil {
				firstErr = acceptErr
			}
			continue
		}

		callErr := modelAuditUpstreamStatusError(resp.StatusCode, raw)
		if firstErr == nil {
			firstErr = callErr
		}
		if !modelAuditPathMissing(resp.StatusCode) {
			return nil, callErr
		}
	}

	if firstErr == nil {
		return nil, fmt.Errorf("上游没有可用的端点")
	}
	if len(paths) > 1 {
		return nil, fmt.Errorf("%w（已尝试 %s）", firstErr, strings.Join(paths, "、"))
	}
	return nil, firstErr
}

// modelAuditUpstreamStatusError 把非 2xx 响应翻译成给用户看的错误。
func modelAuditUpstreamStatusError(status int, raw []byte) error {
	snippet := logredact.RedactText(
		truncateString(string(raw), modelAuditErrorSnippetLimit),
		"api_key", "authorization", "x-api-key",
	)
	return fmt.Errorf("上游返回 HTTP %d：%s", status, snippet)
}

// callUpstream 按所选协议向用户提交的上游发起一次调用。
// timeout 是这道题的单题上限：生成题要吐一整个 HTML 文档，比逻辑题慢得多。
func (s *ModelAuditService) callUpstream(ctx context.Context, input ModelAuditInput, prompt string, timeout time.Duration) (string, error) {
	call, err := buildModelAuditUpstreamRequest(input, prompt)
	if err != nil {
		return "", err
	}

	// 用 Extract 兼作候选可用性判据：取不出正文说明这个端点并不是在按本协议服务。
	raw, err := s.requestModelAuditUpstream(
		ctx,
		http.MethodPost,
		call.Paths,
		call.Headers,
		call.Body,
		func(raw []byte) error {
			_, extractErr := call.Extract(raw)
			return extractErr
		},
		timeout,
	)
	if err != nil {
		return "", err
	}
	return call.Extract(raw)
}

// ListUpstreamModels 用用户提交的凭据读取上游的模型列表，供前端下拉候选。
//
// 候选必须来自被检测的那个上游：本功能的 Base URL 是用户任意填的，
// 跟本站模型广场里的模型往往对不上，拿广场列表去填只会选到上游没有的模型。
// 请求由服务端发起（浏览器直连第三方上游会被 CORS 挡掉），因此沿用与检测
// 相同的地址校验与 DNS 解析后校验，避免变成内网探测入口。
// 端点同样交给 modelAuditEndpointPaths 归一化，用户只填域名也能取到列表。
func (s *ModelAuditService) ListUpstreamModels(ctx context.Context, input ModelAuditInput) ([]string, error) {
	baseURL, apiKey, err := validateModelAuditCredentials(input.BaseURL, input.APIKey)
	if err != nil {
		return nil, err
	}

	protocol, err := normalizeModelAuditProtocol(input.Protocol)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	// Anthropic 的 /models 用 x-api-key 认证且必须带 anthropic-version；
	// 两家的响应都是 {data:[{id}]}，所以解析逻辑可以共用。
	if protocol == ModelAuditProtocolMessages {
		headers["x-api-key"] = apiKey
		headers["anthropic-version"] = modelAuditAnthropicVersion
	} else {
		headers["Authorization"] = "Bearer " + apiKey
	}

	// 与检测题共用端点归一化与候选重试：用户只填域名（不带 /v1）时也能读到模型列表。
	// 「取不到模型列表」也算该候选不可用 —— 有些网关对未知路由回 200 + 一个错误 JSON。
	raw, err := s.requestModelAuditUpstream(
		ctx,
		http.MethodGet,
		modelAuditEndpointPaths(baseURL, modelAuditModelsPath),
		headers,
		nil,
		func(raw []byte) error {
			models, extractErr := extractModelAuditModelIDs(raw)
			if extractErr != nil {
				return extractErr
			}
			if len(models) == 0 {
				return fmt.Errorf("上游没有返回任何模型")
			}
			return nil
		},
		modelAuditModelsTimeout,
	)
	if err != nil {
		return nil, err
	}

	models, err := extractModelAuditModelIDs(raw)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("上游没有返回任何模型")
	}
	return models, nil
}

// extractModelAuditModelIDs 从上游 /models 响应里取出模型 ID。
//
// 兼容三种常见形状：{data:[{id}]}（OpenAI 标准）、{models:[{id}]}、裸数组 [{id}] 或 ["id"]。
func extractModelAuditModelIDs(raw []byte) ([]string, error) {
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("上游响应不是合法 JSON")
	}

	switch value := probe.(type) {
	case []any:
		return collectModelAuditModelIDs(value), nil
	case map[string]any:
		if errObj, ok := value["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok && strings.TrimSpace(msg) != "" {
				return nil, fmt.Errorf("上游返回错误：%s", truncateString(msg, modelAuditErrorSnippetLimit))
			}
		}
		for _, key := range []string{"data", "models"} {
			if list, ok := value[key].([]any); ok {
				return collectModelAuditModelIDs(list), nil
			}
		}
		// 既没有列表也没有标准 error 对象时，把上游自己的 message 透出来。
		// 有些网关对未知路由回 200 + {"code":404,"message":"未找到路由: /v1/models"}，
		// 只报「没有模型列表」会把真正的原因（路径不对）藏起来。
		for _, key := range []string{"message", "msg"} {
			if msg, ok := value[key].(string); ok && strings.TrimSpace(msg) != "" {
				return nil, fmt.Errorf("上游返回错误：%s", truncateString(msg, modelAuditErrorSnippetLimit))
			}
		}
	}
	return nil, fmt.Errorf("上游响应里没有模型列表")
}

// collectModelAuditModelIDs 从条目里挑出模型 ID，去重且保持原顺序。
func collectModelAuditModelIDs(list []any) []string {
	ids := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, entry := range list {
		var id string
		switch item := entry.(type) {
		case string:
			id = item
		case map[string]any:
			if s, ok := item["id"].(string); ok {
				id = s
			} else if s, ok := item["name"].(string); ok {
				id = s
			}
		}

		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// extractModelAuditChatContent 从 OpenAI 兼容（/chat/completions）响应中取出回答正文。
func extractModelAuditChatContent(raw []byte) (string, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("上游响应不是合法 JSON")
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return "", fmt.Errorf("上游返回错误：%s", truncateString(parsed.Error.Message, modelAuditErrorSnippetLimit))
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("上游响应中没有 choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// extractModelAuditResponsesContent 从 OpenAI Responses（/responses）响应中取出回答正文。
//
// 正文藏在 output[].content[] 的 output_text 片段里，一条回答可能被切成多段；
// 同一层里还会有 reasoning 之类非正文的片段，只认 output_text。
func extractModelAuditResponsesContent(raw []byte) (string, error) {
	var parsed struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("上游响应不是合法 JSON")
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return "", fmt.Errorf("上游返回错误：%s", truncateString(parsed.Error.Message, modelAuditErrorSnippetLimit))
	}

	parts := make([]string, 0, len(parsed.Output))
	for _, entry := range parsed.Output {
		for _, part := range entry.Content {
			if part.Type == "output_text" && part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("上游响应中没有 output_text")
	}
	return strings.Join(parts, "\n"), nil
}

// extractModelAuditMessagesContent 从 Anthropic Messages（/messages）响应中取出回答正文。
//
// 正文是 content[] 里 type=text 的片段；开了 extended thinking 时同一层还会有
// thinking 片段，只认 text。
func extractModelAuditMessagesContent(raw []byte) (string, error) {
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("上游响应不是合法 JSON")
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return "", fmt.Errorf("上游返回错误：%s", truncateString(parsed.Error.Message, modelAuditErrorSnippetLimit))
	}

	parts := make([]string, 0, len(parsed.Content))
	for _, part := range parsed.Content {
		if part.Type == "text" && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("上游响应中没有 text 内容")
	}
	return strings.Join(parts, "\n"), nil
}

// normalizeModelAuditTasks 解析要执行的检测项，为空表示全部执行。
func normalizeModelAuditTasks(tasks []string) map[string]bool {
	selected := make(map[string]bool, 2)
	for _, task := range tasks {
		switch strings.TrimSpace(task) {
		case ModelAuditTaskLogic:
			selected[ModelAuditTaskLogic] = true
		case ModelAuditTaskRender:
			selected[ModelAuditTaskRender] = true
		}
	}
	if len(selected) == 0 {
		selected[ModelAuditTaskLogic] = true
		selected[ModelAuditTaskRender] = true
	}
	return selected
}

// summarizeModelAuditItems 汇总逻辑题通过数与生成题判定。
func summarizeModelAuditItems(items []ModelAuditItem) (logicPassed, logicTotal int, renderVerdict string) {
	for i := range items {
		switch items[i].TaskType {
		case ModelAuditTaskLogic:
			logicTotal++
			if items[i].Passed {
				logicPassed++
			}
		case ModelAuditTaskRender:
			renderVerdict = items[i].Verdict
		}
	}
	return logicPassed, logicTotal, renderVerdict
}

// summarizeModelAuditVerdict 汇总单次检测的整体判定。
//
// 逻辑题仍按「答对 / 答错」判定是否降智；生成题不再判定（只展示作品），
// 因此不构成 degraded 也不计入通过率。只有当逻辑题可判定时整体才有 normal / degraded，
// 仅跑了生成题（未做智商判定）时整体无从判定，记 unknown。
func summarizeModelAuditVerdict(logicPassed, logicTotal int, renderVerdict string, items []ModelAuditItem) string {
	responded := 0
	for i := range items {
		if items[i].Verdict != ModelAuditItemUnknown {
			responded++
		}
	}
	if responded == 0 {
		return ModelAuditVerdictUnknown
	}
	if logicTotal > 0 && logicPassed < logicTotal {
		return ModelAuditVerdictDegraded
	}
	// 生成题不再判定，renderVerdict 恒为 generated，不参与整体判定。
	if logicTotal == 0 {
		return ModelAuditVerdictUnknown
	}
	return ModelAuditVerdictNormal
}
