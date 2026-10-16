package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ModelDegradationHandler 用户侧「上游模型降智检测」handler。
//
// 凭据由用户在每次请求里提交，本层只负责转发给服务层：
// 不回显、不记录、不落库。
type ModelDegradationHandler struct {
	auditService *service.ModelDegradationService
}

// NewModelDegradationHandler 创建降智检测 handler。
func NewModelDegradationHandler(auditService *service.ModelDegradationService) *ModelDegradationHandler {
	return &ModelDegradationHandler{auditService: auditService}
}

type modelDegradationStartRequest struct {
	BaseURL string `json:"base_url" binding:"required"`
	APIKey  string `json:"api_key" binding:"required"`
	ModelID string `json:"model_id" binding:"required"`
	// Protocol 上游协议；不传按 chat_completions 处理。
	Protocol        string `json:"protocol"`
	ReasoningEffort string `json:"reasoning_effort"`
	// RenderTimeoutSeconds 生成题的单题超时（秒）；不传或 0 用服务端默认值（15 分钟）。
	RenderTimeoutSeconds int      `json:"render_timeout_seconds"`
	Tasks                []string `json:"tasks"`
	// Questions 本次要跑的具体题目。不传则按 Tasks 跑该类型下的全部预制题。
	Questions []modelDegradationQuestionRequest `json:"questions"`
}

// modelDegradationQuestionRequest 一道要跑的题。
//
// 带 key 表示选的是预制题（服务端按 key 取题目正文与期望答案）；
// key 为空则视为自定义题，正文取 prompt。
type modelDegradationQuestionRequest struct {
	TaskType string `json:"task_type"`
	Key      string `json:"key"`
	Prompt   string `json:"prompt"`
	// Expected 自定义逻辑题的期望答案（数字）；预制题与生成题都不用填。
	Expected string `json:"expected"`
}

// modelDegradationTaskOptionResponse 题库里的一道预制题。
type modelDegradationTaskOptionResponse struct {
	TaskType string `json:"task_type"`
	Key      string `json:"key"`
	Title    string `json:"title"`
	Prompt   string `json:"prompt"`
	Expected string `json:"expected"`
}

// modelDegradationModelsRequest 只带凭据与协议：模型列表来自上游，不需要模型名与检测项。
//
// 用 POST 而不是 GET：API Key 放进查询串会被各级访问日志记下来。
type modelDegradationModelsRequest struct {
	BaseURL string `json:"base_url" binding:"required"`
	APIKey  string `json:"api_key" binding:"required"`
	// Protocol 决定 /models 用哪套认证头；不传按 chat_completions 处理。
	Protocol string `json:"protocol"`
}

type modelDegradationItemResponse struct {
	TaskType   string `json:"task_type"`
	TaskKey    string `json:"task_key"`
	Prompt     string `json:"prompt"`
	Expected   string `json:"expected"`
	Response   string `json:"response"`
	Passed     bool   `json:"passed"`
	Verdict    string `json:"verdict"`
	Detail     string `json:"detail"`
	SourceHash string `json:"source_hash"`
	DurationMs int    `json:"duration_ms"`
}

type modelDegradationRunResponse struct {
	ID      int64  `json:"id"`
	BaseURL string `json:"base_url"`
	ModelID string `json:"model_id"`
	// Protocol 本次检测使用的上游协议。
	Protocol        string `json:"protocol"`
	ReasoningEffort string `json:"reasoning_effort"`
	// RenderTimeoutSeconds 本次检测生成题的单题超时（秒）。
	RenderTimeoutSeconds int                            `json:"render_timeout_seconds"`
	Status               string                         `json:"status"`
	Verdict              string                         `json:"verdict"`
	LogicPassed          int                            `json:"logic_passed"`
	LogicTotal           int                            `json:"logic_total"`
	RenderVerdict        string                         `json:"render_verdict"`
	ErrorMessage         string                         `json:"error_message"`
	DurationMs           int                            `json:"duration_ms"`
	CreatedAt            time.Time                      `json:"created_at"`
	CompletedAt          *time.Time                     `json:"completed_at,omitempty"`
	Items                []modelDegradationItemResponse `json:"items,omitempty"`
	// 共享历史：所有人的检测记录都在同一个池子里，因此需要标出「是谁跑的」。
	// 邮箱一律脱敏，只保留首字符与域名。
	OwnerEmail string `json:"owner_email"`
	// GroupID / GroupName 本次检测填的 Key 命中的平台分组；空表示没识别出分组
	// （填的是第三方上游 Key，或该 Key 未绑分组）。
	GroupID   *int64 `json:"group_id,omitempty"`
	GroupName string `json:"group_name"`
	// IsOwner 表示这条是不是当前登录者自己发起的。前端据此决定是否给「中止」入口，
	// 后端 MarkRunCancelled 另有 user_id 过滤兜底，不依赖这个字段做鉴权。
	IsOwner bool `json:"is_owner"`
}

// Start 发起一次降智检测。
// POST /api/v1/model-degradation/runs
func (h *ModelDegradationHandler) Start(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req modelDegradationStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数无效："+err.Error())
		return
	}

	run, err := h.auditService.Start(c.Request.Context(), subject.UserID, service.ModelDegradationInput{
		BaseURL:              req.BaseURL,
		APIKey:               req.APIKey,
		ModelID:              req.ModelID,
		Protocol:             req.Protocol,
		ReasoningEffort:      req.ReasoningEffort,
		RenderTimeoutSeconds: req.RenderTimeoutSeconds,
		Tasks:                req.Tasks,
		Questions:            modelDegradationQuestionsFromRequest(req.Questions),
	})
	if err != nil {
		// 同一用户已有检测在执行：这是并发冲突，不是参数错误，用 409 区分开。
		if errors.Is(err, service.ErrModelDegradationRunInProgress) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, modelDegradationRunToResponse(run, subject.UserID, false))
}

// Tasks 返回内置题库，供前端铺下拉与「查看题目」。
// GET /api/v1/model-degradation/tasks
func (h *ModelDegradationHandler) Tasks(c *gin.Context) {
	if _, ok := middleware2.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	logic, render := h.auditService.ListTasks()
	response.Success(c, gin.H{
		"logic":  modelDegradationTaskOptionsToResponse(logic),
		"render": modelDegradationTaskOptionsToResponse(render),
	})
}

// Models 用用户提交的凭据读取上游模型列表，供前端下拉候选。
// POST /api/v1/model-degradation/models
func (h *ModelDegradationHandler) Models(c *gin.Context) {
	if _, ok := middleware2.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req modelDegradationModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数无效："+err.Error())
		return
	}

	models, err := h.auditService.ListUpstreamModels(c.Request.Context(), service.ModelDegradationInput{
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Protocol: req.Protocol,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"models": models})
}

// Get 读取单次检测详情。
// GET /api/v1/model-degradation/runs/:id
func (h *ModelDegradationHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	runID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || runID <= 0 {
		response.BadRequest(c, "检测记录 ID 无效")
		return
	}

	// 共享历史：详情同样不做归属过滤，任何人都能点开看。
	run, err := h.auditService.Get(c.Request.Context(), runID)
	if err != nil {
		response.InternalError(c, "读取检测结果失败")
		return
	}
	if run == nil {
		response.NotFound(c, "检测记录不存在")
		return
	}

	response.Success(c, modelDegradationRunToResponse(run, subject.UserID, true))
}

// Cancel 中止用户自己的、仍在运行中的检测。
//
// 用 409 而不是 400：参数没问题，是「这一轮已经结束了」这个状态冲突。
// 前端据此把运行面板切回结果态，不需要额外提示。
func (h *ModelDegradationHandler) Cancel(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	runID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || runID <= 0 {
		response.BadRequest(c, "检测记录 ID 无效")
		return
	}

	if err := h.auditService.CancelRun(c.Request.Context(), subject.UserID, runID); err != nil {
		if errors.Is(err, service.ErrModelDegradationRunNotCancellable) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.InternalError(c, "中止检测失败")
		return
	}

	response.Success(c, gin.H{"cancelled": true})
}

// List 读取当前用户的检测历史。
// GET /api/v1/model-degradation/runs
func (h *ModelDegradationHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	runs, err := h.auditService.List(c.Request.Context(), limit)
	if err != nil {
		response.InternalError(c, "读取检测历史失败")
		return
	}

	items := make([]modelDegradationRunResponse, 0, len(runs))
	for i := range runs {
		items = append(items, modelDegradationRunToResponse(&runs[i], subject.UserID, false))
	}
	response.Success(c, gin.H{"items": items})
}

// maskBaseURL 只保留 scheme + host，路径与查询串一律抹掉。
//
// 共享历史会把上游地址暴露给所有人，而 base_url 是用户自己填的，
// 很可能指向私有中继或内部网关 —— 完整回显等于把别人的中转入口交出去。
func maskBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		// 解析不出来的（用户填了非 URL 的怪值）整体脱敏，至少不回显原文。
		return "***"
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + u.Host + "/***"
}

// maskEmail 保留邮箱首字符与域名，中间抹掉。
func maskEmail(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	at := strings.LastIndex(raw, "@")
	if at <= 0 {
		return "***"
	}
	local := raw[:at]
	if len(local) > 1 {
		local = local[:1] + "***"
	} else {
		local = "***"
	}
	return local + "@" + raw[at+1:]
}

func modelDegradationRunToResponse(run *service.ModelDegradationRun, viewerID int64, withItems bool) modelDegradationRunResponse {
	isOwner := viewerID > 0 && run.UserID == viewerID
	// 自己的记录看完整上游地址，别人的只留域名。
	baseURL := run.BaseURL
	if !isOwner {
		baseURL = maskBaseURL(run.BaseURL)
	}
	resp := modelDegradationRunResponse{
		ID:                   run.ID,
		BaseURL:              baseURL,
		ModelID:              run.ModelID,
		Protocol:             run.Protocol,
		ReasoningEffort:      run.ReasoningEffort,
		RenderTimeoutSeconds: run.RenderTimeoutSeconds,
		Status:               run.Status,
		Verdict:              run.Verdict,
		LogicPassed:          run.LogicPassed,
		LogicTotal:           run.LogicTotal,
		RenderVerdict:        run.RenderVerdict,
		ErrorMessage:         run.ErrorMessage,
		DurationMs:           run.DurationMs,
		CreatedAt:            run.CreatedAt,
		CompletedAt:          run.CompletedAt,
		OwnerEmail:           maskEmail(run.OwnerEmail),
		GroupID:              run.GroupID,
		GroupName:            run.GroupName,
		IsOwner:              isOwner,
	}
	if !withItems || len(run.Items) == 0 {
		return resp
	}
	resp.Items = make([]modelDegradationItemResponse, 0, len(run.Items))
	for i := range run.Items {
		item := run.Items[i]
		resp.Items = append(resp.Items, modelDegradationItemResponse{
			TaskType:   item.TaskType,
			TaskKey:    item.TaskKey,
			Prompt:     item.Prompt,
			Expected:   item.Expected,
			Response:   item.Response,
			Passed:     item.Passed,
			Verdict:    item.Verdict,
			Detail:     item.Detail,
			SourceHash: item.SourceHash,
			DurationMs: item.DurationMs,
		})
	}
	return resp
}

// modelDegradationQuestionsFromRequest 把请求里的题目列表转成服务层结构。
// 返回 nil 表示调用方没指定题目，服务层会退回「按检测项跑全部预制题」。
func modelDegradationQuestionsFromRequest(questions []modelDegradationQuestionRequest) []service.ModelDegradationQuestion {
	if len(questions) == 0 {
		return nil
	}

	items := make([]service.ModelDegradationQuestion, 0, len(questions))
	for _, question := range questions {
		items = append(items, service.ModelDegradationQuestion{
			TaskType: question.TaskType,
			Key:      question.Key,
			Prompt:   question.Prompt,
			Expected: question.Expected,
		})
	}
	return items
}

// modelDegradationTaskOptionsToResponse 把题库条目转成响应结构。
func modelDegradationTaskOptionsToResponse(options []service.ModelDegradationTaskOption) []modelDegradationTaskOptionResponse {
	items := make([]modelDegradationTaskOptionResponse, 0, len(options))
	for _, option := range options {
		items = append(items, modelDegradationTaskOptionResponse{
			TaskType: option.TaskType,
			Key:      option.Key,
			Title:    option.Title,
			Prompt:   option.Prompt,
			Expected: option.Expected,
		})
	}
	return items
}
