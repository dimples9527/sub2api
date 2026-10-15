package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// 本文件是降智检测的上游调用层：重试策略、错误分类与流式解析。
//
// 它要解决两个实际问题，都来自线上观察：
//
//  1. 上游的瞬时失败（HTTP/2 stream 被重置、网关 502/524、连接被中途掐断）会被当成
//     「这一轮检测失败」，用户看到的是满屏上游报错而不是检测结果。这类失败重试一次
//     通常就过去了 —— 不重试等于把上游的抖动算成了模型的判定。
//
//  2. 把上游响应体原样拼进错误文案，等于把网关的返回内容贴到共享历史里：检测历史是
//     所有人可见的，而部分网关会把请求（含 Authorization 头）原样回显在 body 中。
//     因此错误一律分类成固定诊断，不回显上游 body。
//
// 参照同类实现（manxue-ai 的 retryable / upstream_failure / parse_model_response），
// 配额与退避参数与它保持一致，避免两边行为出现无法解释的差异。

const (
	// modelDegradationRetryCount 单题失败后的重试次数（不含首次调用）。
	//
	// 取 2 而不是更大：重试会真实消耗用户上游的额度（题目会重新计费），
	// 而瞬时失败绝大多数在第一次重试就恢复。
	modelDegradationRetryCount = 2
	// modelDegradationRetryMaxDelay 退避时长上限。1s / 2s / 4s → 封顶 8s。
	// 有上限是因为逻辑题的单题预算只有 60s，退避本身不能把预算吃光。
	modelDegradationRetryMaxDelay = 8 * time.Second
	// modelDegradationMaxStreamResponseBytes 流式响应的读取上限。
	//
	// 流式比非流式大得多（16MB vs 256KB）：SSE 每个事件都带完整的事件名与 JSON 信封，
	// 且生成题要吐一整个 HTML 文档，按 256KB 截断会把正常的长输出判成「响应不完整」。
	// 上限存在的意义是防止异常上游把内存拖垮，不是限制正常输出。
	modelDegradationMaxStreamResponseBytes = 16 << 20
)

// modelDegradationRetryDelay 第 attempt 次失败（attempt 从 0 起）后的退避时长。
func modelDegradationRetryDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	delay := time.Duration(1<<uint(attempt)) * time.Second
	if delay > modelDegradationRetryMaxDelay || delay <= 0 {
		return modelDegradationRetryMaxDelay
	}
	return delay
}

// modelDegradationUpstreamError 一次已分类的上游失败。
//
// 之所以要专门定义类型，是为了让重试判定不用去猜错误字符串：分类时就已经确定
// 这次失败值不值得重跑，retryable 只读 Retryable 字段。靠字符串匹配判断「要不要重试」
// 会随上游文案变化而失准，而这里的分类是我们自己写的，不会变。
type modelDegradationUpstreamError struct {
	// Message 已经是可直接展示给用户（并可存入共享历史）的固定诊断。
	Message string
	// Status HTTP 状态码；0 表示这一层没有拿到响应（网络层失败）。
	Status int
	// Retryable 这次失败是否值得重跑。
	Retryable bool
}

func (e *modelDegradationUpstreamError) Error() string { return e.Message }

// modelDegradationStatusReasons 按状态码给出的固定诊断。
//
// 与「把响应体截 300 字拼进去」的做法相反：这里一个字都不取自上游响应。
// 状态码本身已经说明问题性质，而响应体是不可信输入（见文件头第 2 点）。
var modelDegradationStatusReasons = map[int]string{
	400: "上游拒绝了请求参数，请检查模型名与协议",
	401: "上游鉴权失败，请检查 API Key",
	402: "上游账户余额不足，请检查接口账户",
	403: "上游拒绝访问，请检查 Key 权限或模型是否可用",
	404: "接口或模型不存在，请检查 Base URL 与协议",
	408: "上游请求超时，请稍后重试",
	409: "上游请求冲突，请稍后重试",
	422: "上游无法处理该请求，请检查模型名与参数",
	429: "上游限流或额度不足，请稍后重试",
	500: "上游服务异常，请稍后重试",
	502: "上游服务暂时不可用，请稍后重试",
	503: "上游服务暂时不可用，请稍后重试",
	504: "上游网关等待模型响应超时",
	// 524 不是标准 HTTP 码，是 Cloudflare 专有码：边缘在 100s 内没等到源站就自己断连。
	// 单独列出来是因为它的处置方式与 504 不同（换节点/联系服务商，而不是重试本地参数）。
	524: "上游网关等待模型响应超时",
}

// modelDegradationTimeoutPattern 上游错误码/消息里表示「超时」的特征。
// 只用于分类，不回显：匹配的是上游自己写的 code 与 message，输出的是我们固定的句子。
var modelDegradationTimeoutPattern = regexp.MustCompile(`\b(?:524|504|timeout|timed out|deadline exceeded)\b`)

// diagnoseModelDegradationStatus 把非 2xx 响应分类成固定诊断。
//
// raw 只用于读取上游自己声明的错误码（error.code / error.type / error.message），
// 不会被拼进返回的文案 —— 响应体是外部输入，可能夹带凭据。
func diagnoseModelDegradationStatus(status int, raw []byte) *modelDegradationUpstreamError {
	// 先按上游自己给的错误码分类：同样的 400，额度不足与参数非法要给用户不同提示。
	if code := extractModelDegradationErrorCode(raw); code != "" {
		if diagnosed := diagnoseModelDegradationErrorCode(code, raw); diagnosed != nil {
			return diagnosed
		}
	}

	reason, ok := modelDegradationStatusReasons[status]
	if !ok {
		if status >= 500 {
			reason = "上游服务异常，请稍后重试"
		} else {
			reason = "上游接口请求失败，请检查地址、协议与凭据"
		}
	}
	return &modelDegradationUpstreamError{
		Message:   reason,
		Status:    status,
		Retryable: modelDegradationStatusRetryable(status),
	}
}

// extractModelDegradationErrorCode 从响应体里读上游声明的错误码。
// 取不到就返回空串，交给调用方按状态码兜底。
func extractModelDegradationErrorCode(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var parsed struct {
		Code string `json:"code"`
		Type string `json:"type"`
		Err  *struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ""
	}
	if parsed.Err != nil {
		if parsed.Err.Code != "" {
			return strings.ToLower(parsed.Err.Code)
		}
		if parsed.Err.Type != "" {
			return strings.ToLower(parsed.Err.Type)
		}
	}
	if parsed.Code != "" {
		return strings.ToLower(parsed.Code)
	}
	return strings.ToLower(parsed.Type)
}

// diagnoseModelDegradationErrorCode 按上游错误码分类。
//
// 这些码各家网关写法不统一（insufficient_quota / quota_exceeded / rate_limit_exceeded /
// invalid_api_key …），只认最常见的几个；认不出的返回 nil，由状态码兜底。
func diagnoseModelDegradationErrorCode(code string, raw []byte) *modelDegradationUpstreamError {
	switch code {
	case "invalid_api_key", "invalid_apikey", "authentication_error", "unauthorized":
		return &modelDegradationUpstreamError{
			Message: "上游鉴权失败，请检查 API Key", Status: 401, Retryable: false,
		}
	case "insufficient_quota", "quota_exceeded", "billing_hard_limit_reached":
		return &modelDegradationUpstreamError{
			Message: "上游额度不足，请检查接口账户余额或限额", Status: 402, Retryable: false,
		}
	case "rate_limit_exceeded", "rate_limit", "too_many_requests":
		return &modelDegradationUpstreamError{
			Message: "上游请求限流，请稍后重试", Status: 429, Retryable: true,
		}
	case "timeout", "upstream_timeout", "gateway_timeout", "request_timeout":
		return &modelDegradationUpstreamError{
			Message: "上游网关或模型响应超时，请稍后重试或联系接口服务商", Status: 504, Retryable: true,
		}
	}

	// 错误码没命中但消息里写明了超时：有些网关只在 message 里写 "upstream timed out"。
	if modelDegradationTimeoutPattern.MatchString(strings.ToLower(extractModelDegradationErrorMessage(raw))) {
		return &modelDegradationUpstreamError{
			Message: "上游网关或模型响应超时，请稍后重试或联系接口服务商", Status: 504, Retryable: true,
		}
	}
	return nil
}

// extractModelDegradationErrorMessage 读上游错误消息，仅供分类匹配使用。
// 调用方不得把它拼进任何对外可见的文案。
func extractModelDegradationErrorMessage(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var parsed struct {
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Err     *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ""
	}
	if parsed.Err != nil && parsed.Err.Message != "" {
		return parsed.Err.Message
	}
	if parsed.Message != "" {
		return parsed.Message
	}
	return parsed.Msg
}

// modelDegradationStatusRetryable 判定状态码值不值得重跑。
//
// 可重试：408（请求超时）、429（限流）、5xx（上游自身故障）。
// 不可重试：4xx（401/403/404/400 等）—— 那是「我们发错了」，重试一百次也是错的，
// 而且会白白消耗用户额度。
func modelDegradationStatusRetryable(status int) bool {
	if status == 408 || status == 429 {
		return true
	}
	return status >= 500 && status <= 599
}

// diagnoseModelDegradationTransport 把网络层失败分类成固定诊断。
//
// 这一层的错误字符串来自传输层（Go 标准库或中间代理），同样不可信，
// 因此只做特征匹配、输出固定句子。
func diagnoseModelDegradationTransport(err error) *modelDegradationUpstreamError {
	if err == nil {
		return nil
	}

	// 超时：既包含我们自己的 deadline，也包含 net.Error 声明的超时。
	if errors.Is(err, context.DeadlineExceeded) {
		return &modelDegradationUpstreamError{
			Message: "上游响应超时，请稍后重试或检查接口服务状态", Retryable: true,
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &modelDegradationUpstreamError{
			Message: "上游响应超时，请稍后重试或检查接口服务状态", Retryable: true,
		}
	}

	// 域名解析失败。
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &modelDegradationUpstreamError{
			Message: "API 域名解析失败，请检查地址拼写或 DNS", Retryable: false,
		}
	}

	// 证书问题不可重试：重试只会再失败一次，还白白消耗额度。
	var (
		unknownAuthErr   x509.UnknownAuthorityError
		hostnameErr      x509.HostnameError
		certInvalidErr   x509.CertificateInvalidError
		constraintErr    x509.InsecureAlgorithmError
		unhandledCritErr x509.UnhandledCriticalExtension
	)
	switch {
	case errors.As(err, &unknownAuthErr), errors.As(err, &hostnameErr),
		errors.As(err, &certInvalidErr), errors.As(err, &constraintErr),
		errors.As(err, &unhandledCritErr):
		return &modelDegradationUpstreamError{
			Message: "API HTTPS 证书校验失败，请检查接口证书", Retryable: false,
		}
	}

	// 兜底按字符串特征匹配。Go 的 HTTP/2 错误来自内部包，拿不到可断言的类型，
	// 只能靠文案识别 —— 好在它的文案是稳定的。
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "http2"), strings.Contains(msg, "stream closed"),
		strings.Contains(msg, "stream error"), strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"), strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "eof"), strings.Contains(msg, "no such host"):
		return &modelDegradationUpstreamError{
			Message: "无法建立或保持与上游的连接，请检查接口服务和网络", Retryable: true,
		}
	}

	return &modelDegradationUpstreamError{
		Message: "无法建立或保持与上游的连接，请检查接口服务和网络", Retryable: true,
	}
}

// modelDegradationFailureDetail 把一次失败转成写进 item.detail 的文案。
//
// 已分类的上游失败本身就是一个完整句子，直接存；其余错误（构造请求失败、
// 候选端点都不可用等）沿用「上游请求失败：…」前缀，前端据此做二次收敛。
func modelDegradationFailureDetail(err error) string {
	if err == nil {
		return ""
	}
	var upstreamErr *modelDegradationUpstreamError
	if errors.As(err, &upstreamErr) {
		return upstreamErr.Message
	}
	return "上游请求失败：" + err.Error()
}

// diagnoseModelDegradationBodyError 2xx 响应体里带着上游错误对象。
//
// 部分网关对业务失败也回 200，只在 body 里写 error —— 这时端点是对的、只是这一轮
// 没跑成。判为可重试：换端点没有意义，重试却有可能成。
//
// 错误码与消息只用于分类，不回显。
func diagnoseModelDegradationBodyError(raw []byte) *modelDegradationUpstreamError {
	if code := extractModelDegradationErrorCode(raw); code != "" {
		if diagnosed := diagnoseModelDegradationErrorCode(code, raw); diagnosed != nil {
			return diagnosed
		}
	}
	if message := extractModelDegradationErrorMessage(raw); message != "" &&
		modelDegradationTimeoutPattern.MatchString(strings.ToLower(message)) {
		return &modelDegradationUpstreamError{
			Message: "上游网关或模型响应超时，请稍后重试或联系接口服务商", Status: http.StatusOK, Retryable: true,
		}
	}
	return &modelDegradationUpstreamError{
		Message: "上游生成失败，请稍后重试或联系接口服务商", Status: http.StatusOK, Retryable: true,
	}
}

// modelDegradationRetryable 判定一次失败值不值得重跑。
//
// 只有我们自己分类过的 modelDegradationUpstreamError 才有确定答案；
// 其余（构造请求失败、端点都不可用等）一律不重试 —— 那类错误重跑也是同样的结果。
func modelDegradationRetryable(err error) bool {
	var upstreamErr *modelDegradationUpstreamError
	if errors.As(err, &upstreamErr) {
		return upstreamErr.Retryable
	}
	return false
}

// callUpstreamWithRetry 在给定预算内调用上游，失败且值得重试时退避重跑。
//
// 关键约束：**重试不延长总预算**。attempt 0 用完整 timeout，之后的每次尝试只拿到
// 「剩余预算」（deadline - now）。否则「重试 2 次」会让单题最长耗时变成 3 倍，
// 一轮检测（逻辑题 60s + 生成题 6min）就会突破 modelDegradationTotalTimeout，
// 被整体超时掐断 —— 那比不重试更糟。
//
// 退避期间响应 ctx 取消：用户按「停止」时不必等完退避。
func (s *ModelDegradationService) callUpstreamWithRetry(
	ctx context.Context,
	input ModelDegradationInput,
	prompt string,
	timeout time.Duration,
) (string, error) {
	deadline := time.Now().Add(timeout)

	var lastErr error
	for attempt := 0; attempt <= modelDegradationRetryCount; attempt++ {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		out, err := s.callUpstream(ctx, input, prompt, remaining)
		if err == nil {
			return out, nil
		}
		lastErr = err

		delay := modelDegradationRetryDelay(attempt)
		// 预算不够再等一轮退避，就别等了 —— 剩下的时间跑一次完整请求都不够。
		if attempt >= modelDegradationRetryCount || !modelDegradationRetryable(err) || remaining <= delay {
			break
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return "", lastErr
		}
	}

	if lastErr == nil {
		// 预算一开始就是 0（timeout <= 0）：一次都没跑成。
		return "", &modelDegradationUpstreamError{Message: "上游响应超时，请稍后重试或检查接口服务状态", Retryable: true}
	}
	return "", lastErr
}

// extractModelDegradationResponsesStream 解析 responses 协议的 SSE 流。
//
// 为什么 responses 要走流式：非流式时客户端要一直 hold 住连接等上游一次性吐完，
// 生成题最多要等 6 分钟，中间任何反代 / CDN / 网关的 idle timeout 都会把连接掐断
// （表现为 HTTP/2 stream 被重置）。流式是边生成边吐帧，连接始终有流量，
// 不会被 idle 判定掐断，长输出的成功率明显更高。
//
// 只认 response.completed 作为完成标志，不做 delta 拼接：少了这一段就说明上游
// 没把这次响应跑完，此时拿到的是半截输出，判分结果不可信 —— 宁可明确失败，
// 也不要拿半截内容去判「模型是否降智」。
func extractModelDegradationResponsesStream(raw []byte) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", &modelDegradationUpstreamError{Message: "上游返回空响应，请检查接口服务状态", Retryable: true}
	}

	// 先按完整 JSON 解析：部分上游会忽略 stream 参数、仍然返回非流式响应。
	// 不先试 JSON 的话，这类上游会被误判成「流式响应缺少完整结果」——
	// 明明拿到了完整答案，却报一个看起来像网络中断的错。
	var asJSON map[string]any
	if err := json.Unmarshal([]byte(trimmed), &asJSON); err == nil {
		return extractModelDegradationResponsesOutput(asJSON)
	}

	// SSE 的分块分隔符在 CRLF 流上是 \r\n\r\n，只认 \n\n 会永远匹配不到，
	// 症状是「连上了却解析不出任何事件」。先归一化再分块。
	normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")

	var completed map[string]any
	for _, block := range strings.Split(normalized, "\n\n") {
		payload := modelDegradationSSEPayload(block)
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var event struct {
			Type     string         `json:"type"`
			Error    map[string]any `json:"error"`
			Response map[string]any `json:"response"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			// 单个事件解析失败不能直接判死：流里可能混着心跳注释。
			// 但整个流结束时仍没拿到 completed，下面会报「响应不完整」。
			continue
		}

		kind := event.Type
		if kind == "" {
			kind = modelDegradationSSEEventName(block)
		}

		if len(event.Error) > 0 || kind == "error" || kind == "response.failed" {
			return "", modelDegradationStreamFailure(event.Error, payload)
		}
		if kind == "response.incomplete" {
			return "", &modelDegradationUpstreamError{
				Message: "上游响应未完成，可能达到输出上限或被中断", Status: 200, Retryable: true,
			}
		}
		if kind == "response.completed" && event.Response != nil {
			completed = event.Response
		}
	}

	if completed == nil {
		return "", &modelDegradationUpstreamError{
			Message: "上游流式响应缺少完整结果，可能已中断", Retryable: true,
		}
	}
	return extractModelDegradationResponsesOutput(completed)
}

// modelDegradationSSEPayload 从一个 SSE 块里取出 data 字段的正文。
// 一个块里可能有多行 data:（部分实现会拆分长事件），按行拼接。
func modelDegradationSSEPayload(block string) string {
	lines := make([]string, 0, 2)
	for _, line := range strings.Split(block, "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		lines = append(lines, strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "data:")), " "))
	}
	return strings.Join(lines, "\n")
}

// modelDegradationSSEEventName 读块里的 event: 行，供没有 type 字段的事件兜底。
func modelDegradationSSEEventName(block string) string {
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "event:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
	}
	return ""
}

// modelDegradationStreamFailure 把 SSE 里的 error 事件分类成固定诊断。
// 错误码与消息只用于分类，不回显。
func modelDegradationStreamFailure(errObj map[string]any, payload string) error {
	var code string
	if errObj != nil {
		if raw, ok := errObj["code"].(string); ok {
			code = strings.ToLower(raw)
		}
	}
	if code != "" {
		if diagnosed := diagnoseModelDegradationErrorCode(code, []byte(payload)); diagnosed != nil {
			return diagnosed
		}
	}
	if errObj != nil {
		if msg, ok := errObj["message"].(string); ok && modelDegradationTimeoutPattern.MatchString(strings.ToLower(msg)) {
			return &modelDegradationUpstreamError{
				Message: "上游网关或模型响应超时，请稍后重试或联系接口服务商", Status: 504, Retryable: true,
			}
		}
	}
	return &modelDegradationUpstreamError{
		Message: "上游生成失败，请稍后重试或联系接口服务商", Retryable: true,
	}
}

// extractModelDegradationResponsesOutput 从 responses 的响应对象里取出回答正文。
//
// 与 extractModelDegradationResponsesContent 的区别：那个从完整 JSON 里取，
// 这个从已经解析出来的 map 里取（流式场景下 completed 事件的 response 字段）。
func extractModelDegradationResponsesOutput(response map[string]any) (string, error) {
	if response == nil {
		return "", &modelDegradationUpstreamError{Message: "上游流式响应缺少完整结果，可能已中断", Retryable: true}
	}

	// 上游可能把状态写成 failed / incomplete：2xx 的流里同样可能带着失败态。
	if status, ok := response["status"].(string); ok && status != "" && status != "completed" {
		if status == "incomplete" {
			return "", &modelDegradationUpstreamError{
				Message: "上游响应未完成，可能达到输出上限或被中断", Status: 200, Retryable: true,
			}
		}
		return "", &modelDegradationUpstreamError{
			Message: "上游生成失败，请稍后重试或联系接口服务商", Status: 200, Retryable: true,
		}
	}

	if errObj, ok := response["error"].(map[string]any); ok && len(errObj) > 0 {
		return "", modelDegradationStreamFailure(errObj, "")
	}

	output, _ := response["output"].([]any)
	parts := make([]string, 0, len(output))
	for _, entry := range output {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if itemType, _ := item["type"].(string); itemType != "" && itemType != "message" {
			continue
		}
		content, _ := item["content"].([]any)
		for _, piece := range content {
			part, ok := piece.(map[string]any)
			if !ok {
				continue
			}
			if partType, _ := part["type"].(string); partType != "output_text" {
				// 同一层里还会有 reasoning 之类非正文片段，只认 output_text。
				continue
			}
			if text, _ := part["text"].(string); text != "" {
				parts = append(parts, text)
			}
		}
	}
	if len(parts) == 0 {
		return "", &modelDegradationUpstreamError{
			Message: "上游响应中没有文本内容", Status: 200, Retryable: true,
		}
	}
	return strings.Join(parts, "\n"), nil
}
