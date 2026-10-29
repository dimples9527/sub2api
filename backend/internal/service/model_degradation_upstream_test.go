//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 一次完整的 Anthropic Messages 流：正文被切成两段，中间夹着 thinking 增量。
// 期望只拼 text_delta，thinking 与工具入参一律不进作品源码。
func TestExtractModelDegradationMessagesStreamConcatenatesTextDeltas(t *testing.T) {
	raw := "" +
		"event: message_start\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"role\":\"assistant\"}}\n\n" +
		"event: content_block_start\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: ping\n" +
		"data: {\"type\":\"ping\"}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"<svg xmlns=\\\"http://www.w3.org/2000/svg\\\">\"}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"先构图\"}}\n\n" +
		"event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"</svg>\"}}\n\n" +
		"event: content_block_stop\n" +
		"data: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":12}}\n\n" +
		"event: message_stop\n" +
		"data: {\"type\":\"message_stop\"}\n\n"

	content, err := extractModelDegradationMessagesStream([]byte(raw))
	require.NoError(t, err)
	require.Equal(t, `<svg xmlns="http://www.w3.org/2000/svg"></svg>`, content)
	require.NotContains(t, content, "先构图")
}

// CRLF 流必须同样能解析：只认 \n\n 会让整条流一个事件都取不出来。
func TestExtractModelDegradationMessagesStreamHandlesCRLF(t *testing.T) {
	raw := "event: content_block_delta\r\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\r\n\r\n" +
		"event: message_stop\r\n" +
		"data: {\"type\":\"message_stop\"}\r\n\r\n"

	content, err := extractModelDegradationMessagesStream([]byte(raw))
	require.NoError(t, err)
	require.Equal(t, "hello", content)
}

// 上游忽略 stream 参数、直接回完整 JSON 时，不能误判成「流式响应缺少完整结果」。
func TestExtractModelDegradationMessagesStreamAcceptsNonStreamJSON(t *testing.T) {
	raw := `{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"<svg></svg>"}]}`

	content, err := extractModelDegradationMessagesStream([]byte(raw))
	require.NoError(t, err)
	require.Equal(t, "<svg></svg>", content)
}

// 流里带 error 事件时必须报错，而不是把已有片段当成功结果。
func TestExtractModelDegradationMessagesStreamSurfacesErrorEvent(t *testing.T) {
	raw := "event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"<svg\"}}\n\n" +
		"event: error\n" +
		"data: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"Overloaded\"}}\n\n"

	_, err := extractModelDegradationMessagesStream([]byte(raw))
	require.Error(t, err)

	var upstreamErr *modelDegradationUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.NotEmpty(t, upstreamErr.Message)
	// 上游原文不得出现在诊断里（历史是共享的）。
	require.NotContains(t, upstreamErr.Message, "Overloaded")
}

// 没收到 message_stop 就是半截输出，宁可明确失败。
func TestExtractModelDegradationMessagesStreamRequiresMessageStop(t *testing.T) {
	raw := "event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"<svg>\"}}\n\n"

	_, err := extractModelDegradationMessagesStream([]byte(raw))
	require.Error(t, err)

	var upstreamErr *modelDegradationUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.Contains(t, upstreamErr.Message, "缺少完整结果")
}

// stop_reason=max_tokens 说明输出被本地上限截断：内容不完整，且重试无意义。
func TestExtractModelDegradationMessagesStreamRejectsTruncatedOutput(t *testing.T) {
	raw := "event: content_block_delta\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"<svg>\"}}\n\n" +
		"event: message_delta\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"max_tokens\"}}\n\n" +
		"event: message_stop\n" +
		"data: {\"type\":\"message_stop\"}\n\n"

	_, err := extractModelDegradationMessagesStream([]byte(raw))
	require.Error(t, err)

	var upstreamErr *modelDegradationUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.Contains(t, upstreamErr.Message, "max_tokens")
	require.False(t, upstreamErr.Retryable)
}

// 空响应要给明确诊断，不能静默返回空字符串被当成「模型没产出」。
func TestExtractModelDegradationMessagesStreamRejectsEmptyBody(t *testing.T) {
	_, err := extractModelDegradationMessagesStream([]byte("   \n"))
	require.Error(t, err)

	var upstreamErr *modelDegradationUpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	require.True(t, upstreamErr.Retryable)
}
