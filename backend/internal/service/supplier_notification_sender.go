package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	supplierNotificationSendTimeout = 15 * time.Second
	supplierNotificationMaxResponse = 8192
)

type supplierNotificationHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type supplierNotificationSender struct {
	encryptor  SecretEncryptor
	httpClient supplierNotificationHTTPClient
	timeout    time.Duration
}

func NewSupplierNotificationSender(encryptor SecretEncryptor) *supplierNotificationSender {
	return &supplierNotificationSender{
		encryptor:  encryptor,
		httpClient: &http.Client{Timeout: supplierNotificationSendTimeout},
		timeout:    supplierNotificationSendTimeout,
	}
}

func (s *supplierNotificationSender) Send(ctx context.Context, channel SupplierNotificationChannel, payload SupplierNotificationEventPayload) (SupplierNotificationSendResult, error) {
	if s == nil || s.encryptor == nil || channel.ID < 0 {
		return SupplierNotificationSendResult{}, ErrSupplierNotificationInvalid
	}
	configJSON, err := s.decrypt(channel.ConfigEncrypted)
	if err != nil {
		return SupplierNotificationSendResult{}, err
	}
	proxy, err := s.decryptProxy(channel.ProxyEncrypted)
	if err != nil {
		return SupplierNotificationSendResult{}, err
	}

	switch channel.ChannelType {
	case SupplierNotificationChannelFeishu:
		var config SupplierNotificationFeishuConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return SupplierNotificationSendResult{}, fmt.Errorf("解析飞书通知配置失败: %w", err)
		}
		if err := validateFeishuConfig(config); err != nil {
			return SupplierNotificationSendResult{}, err
		}
		return s.sendFeishu(ctx, config, proxy, payload)
	case SupplierNotificationChannelEmail:
		var config SupplierNotificationEmailConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return SupplierNotificationSendResult{}, fmt.Errorf("解析邮件通知配置失败: %w", err)
		}
		if err := validateEmailConfig(config); err != nil {
			return SupplierNotificationSendResult{}, err
		}
		return s.sendEmail(ctx, config, proxy, payload)
	default:
		return SupplierNotificationSendResult{}, fmt.Errorf("不支持的通知渠道类型: %s", channel.ChannelType)
	}
}

func (s *supplierNotificationSender) decrypt(ciphertext string) ([]byte, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return []byte("{}"), nil
	}
	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("解密通知配置失败: %w", err)
	}
	return []byte(plaintext), nil
}

func (s *supplierNotificationSender) decryptProxy(ciphertext string) (*SupplierNotificationProxyConfig, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return nil, nil
	}
	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("解密通知代理配置失败: %w", err)
	}
	var proxy SupplierNotificationProxyConfig
	if err := json.Unmarshal([]byte(plaintext), &proxy); err != nil {
		return nil, fmt.Errorf("解析通知代理配置失败: %w", err)
	}
	if strings.TrimSpace(proxy.URL) == "" {
		return nil, nil
	}
	if err := validateProxyConfig(proxy); err != nil {
		return nil, err
	}
	return &proxy, nil
}

func validateFeishuConfig(config SupplierNotificationFeishuConfig) error {
	webhook := strings.TrimSpace(config.WebhookURL)
	parsed, err := url.Parse(webhook)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ErrSupplierNotificationInvalid
	}
	return nil
}

func validateEmailConfig(config SupplierNotificationEmailConfig) error {
	if strings.TrimSpace(config.Host) == "" || config.Port < 1 || config.Port > 65535 {
		return ErrSupplierNotificationInvalid
	}
	if strings.TrimSpace(config.From) == "" || len(config.To) == 0 {
		return ErrSupplierNotificationInvalid
	}
	if _, err := mail.ParseAddress(config.From); err != nil {
		return fmt.Errorf("邮件发件人地址无效: %w", err)
	}
	for _, recipient := range config.To {
		if _, err := mail.ParseAddress(recipient); err != nil {
			return fmt.Errorf("邮件收件人地址无效: %w", err)
		}
	}
	if config.Username == "" && config.Password != "" {
		return ErrSupplierNotificationInvalid
	}
	return nil
}

func validateProxyConfig(config SupplierNotificationProxyConfig) error {
	parsed, err := url.Parse(strings.TrimSpace(config.URL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ErrSupplierNotificationInvalid
	}
	return nil
}

func (s *supplierNotificationSender) sendFeishu(ctx context.Context, config SupplierNotificationFeishuConfig, proxy *SupplierNotificationProxyConfig, payload SupplierNotificationEventPayload) (SupplierNotificationSendResult, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body := map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": supplierNotificationMessage(payload)},
	}
	if secret := strings.TrimSpace(config.Secret); secret != "" {
		stringToSign := timestamp + "\n" + secret
		mac := hmac.New(sha256.New, []byte(stringToSign))
		body["timestamp"] = timestamp
		body["sign"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return SupplierNotificationSendResult{}, fmt.Errorf("编码飞书通知请求失败: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(config.WebhookURL), bytes.NewReader(rawBody))
	if err != nil {
		return SupplierNotificationSendResult{}, ErrSupplierNotificationInvalid
	}
	request.Header.Set("Content-Type", "application/json")
	client := s.httpClient
	if proxy != nil {
		client, err = s.httpClientWithProxy(proxy)
		if err != nil {
			return SupplierNotificationSendResult{}, err
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return SupplierNotificationSendResult{}, errors.New("飞书通知请求失败")
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, supplierNotificationMaxResponse))
	if readErr != nil {
		return SupplierNotificationSendResult{HTTPStatus: response.StatusCode}, errors.New("读取飞书通知响应失败")
	}
	safeBody := sanitizeSupplierNotificationText(string(responseBody), config.Secret)
	result := SupplierNotificationSendResult{HTTPStatus: response.StatusCode, ResponseBody: safeBody}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("飞书通知返回 HTTP 状态码 %d: %s", response.StatusCode, safeBody)
	}
	var business struct {
		Code *int   `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(responseBody, &business); err == nil && business.Code != nil && *business.Code != 0 {
		return result, fmt.Errorf("飞书通知业务失败 code=%d: %s", *business.Code, sanitizeSupplierNotificationText(business.Msg, config.Secret))
	}
	return result, nil
}

func (s *supplierNotificationSender) httpClientWithProxy(proxy *SupplierNotificationProxyConfig) (supplierNotificationHTTPClient, error) {
	if proxy == nil {
		return s.httpClient, nil
	}
	proxyURL, err := url.Parse(strings.TrimSpace(proxy.URL))
	if err != nil || proxyURL.Host == "" {
		return nil, ErrSupplierNotificationInvalid
	}
	if proxy.Username != "" {
		proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	timeout := s.timeout
	if timeout <= 0 {
		timeout = supplierNotificationSendTimeout
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

func (s *supplierNotificationSender) sendEmail(ctx context.Context, config SupplierNotificationEmailConfig, proxy *SupplierNotificationProxyConfig, payload SupplierNotificationEventPayload) (SupplierNotificationSendResult, error) {
	if err := ctx.Err(); err != nil {
		return SupplierNotificationSendResult{}, err
	}
	target := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	conn, err := dialSupplierNotificationSMTP(ctx, target, proxy)
	if err != nil {
		return SupplierNotificationSendResult{}, errors.New("连接 SMTP 服务器失败")
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return SupplierNotificationSendResult{}, errors.New("创建 SMTP 客户端失败")
	}
	defer client.Close()
	if config.StartTLS {
		supported, _ := client.Extension("STARTTLS")
		if !supported {
			return SupplierNotificationSendResult{}, errors.New("SMTP 服务器不支持 STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return SupplierNotificationSendResult{}, errors.New("SMTP STARTTLS 握手失败")
		}
	}
	if config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host)); err != nil {
			return SupplierNotificationSendResult{}, errors.New("SMTP 身份认证失败")
		}
	}
	if err := client.Mail(config.From); err != nil {
		return SupplierNotificationSendResult{}, errors.New("SMTP 发件人设置失败")
	}
	for _, recipient := range config.To {
		if err := client.Rcpt(recipient); err != nil {
			return SupplierNotificationSendResult{}, errors.New("SMTP 收件人设置失败")
		}
	}
	writer, err := client.Data()
	if err != nil {
		return SupplierNotificationSendResult{}, errors.New("SMTP 打开邮件内容失败")
	}
	message := formatSupplierNotificationEmail(config, payload)
	if _, err := io.Copy(writer, strings.NewReader(message)); err != nil {
		_ = writer.Close()
		return SupplierNotificationSendResult{}, errors.New("SMTP 写入邮件内容失败")
	}
	if err := writer.Close(); err != nil {
		return SupplierNotificationSendResult{}, errors.New("SMTP 发送邮件内容失败")
	}
	if err := client.Quit(); err != nil {
		return SupplierNotificationSendResult{}, errors.New("SMTP 关闭连接失败")
	}
	return SupplierNotificationSendResult{HTTPStatus: http.StatusOK}, nil
}

func dialSupplierNotificationSMTP(ctx context.Context, target string, proxy *SupplierNotificationProxyConfig) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: supplierNotificationSendTimeout}
	if proxy == nil || strings.TrimSpace(proxy.URL) == "" {
		return dialer.DialContext(ctx, "tcp", target)
	}
	proxyURL, err := url.Parse(strings.TrimSpace(proxy.URL))
	if err != nil || proxyURL.Host == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https") {
		return nil, ErrSupplierNotificationInvalid
	}
	conn, err := dialer.DialContext(ctx, "tcp", proxyURL.Host)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: target}, Host: target, Header: make(http.Header)}
	if proxy.Username != "" {
		request.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(proxy.Username+":"+proxy.Password)))
	}
	if err := request.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("HTTP 代理连接 SMTP 失败，状态码 %d", response.StatusCode)
	}
	return conn, nil
}

func supplierNotificationEventTypeText(eventType string) string {
	switch eventType {
	case SupplierBalanceAlertEventRecovered:
		return "余额恢复"
	case SupplierCostAlertEventOverrun:
		return "成本超额"
	case SupplierCostAlertEventRecovered:
		return "成本恢复"
	case SupplierGroupChangeEventType:
		return "分组变化"
	default:
		return "余额不足"
	}
}

func supplierNotificationMessage(payload SupplierNotificationEventPayload) string {
	if payload.EventType == SupplierGroupChangeEventType {
		return supplierGroupChangeNotificationMessage(payload)
	}

	lines := []string{
		"供应商通知",
		"类型: " + supplierNotificationEventTypeText(payload.EventType),
		"供应商: " + payload.ProviderName,
	}
	switch payload.EventType {
	case SupplierCostAlertEventOverrun, SupplierCostAlertEventRecovered:
		lines = append(lines,
			"超额金额: "+payload.Balance.String(),
			"触发阈值: "+payload.Threshold.String())
	default:
		lines = append(lines,
			"余额: "+payload.Balance.String(),
			"阈值: "+payload.Threshold.String())
	}
	lines = append(lines, "时间: "+payload.ObservedAt.Format(time.RFC3339))
	return strings.Join(lines, "\n")
}

func supplierGroupChangeNotificationMessage(payload SupplierNotificationEventPayload) string {
	lines := []string{"供应商「" + payload.ProviderName + "」分组发生变化"}
	if payload.GroupChanges == nil {
		return strings.Join(lines, "\n")
	}

	appendChanges := func(title string, changes []SupplierProviderGroupChange, format func(SupplierProviderGroupChange) string) {
		if len(changes) == 0 {
			return
		}
		lines = append(lines, "", title)
		for _, change := range changes {
			lines = append(lines, "- "+format(change))
		}
	}
	appendChanges("新增分组：", payload.GroupChanges.Added, func(change SupplierProviderGroupChange) string {
		return change.UpstreamKey + "，倍率 " + formatSupplierNotificationRate(change.NewRateMultiplier)
	})
	appendChanges("删除分组：", payload.GroupChanges.Removed, func(change SupplierProviderGroupChange) string {
		return change.UpstreamKey + "，原倍率 " + formatSupplierNotificationRate(change.OldRateMultiplier)
	})
	appendChanges("倍率变化：", payload.GroupChanges.RateChanged, func(change SupplierProviderGroupChange) string {
		return change.UpstreamKey + "：" + formatSupplierNotificationRate(change.OldRateMultiplier) + " → " + formatSupplierNotificationRate(change.NewRateMultiplier)
	})
	appendChanges("名称变化：", payload.GroupChanges.NameChanged, func(change SupplierProviderGroupChange) string {
		return change.UpstreamKey + "：" + change.OldName + " → " + change.NewName
	})
	return strings.Join(lines, "\n")
}

func formatSupplierNotificationRate(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatSupplierNotificationEmail(config SupplierNotificationEmailConfig, payload SupplierNotificationEventPayload) string {
	subject := "[供应商] " + supplierNotificationEventTypeText(payload.EventType)
	to := make([]string, 0, len(config.To))
	for _, item := range config.To {
		to = append(to, item)
	}
	return "From: " + config.From + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + supplierNotificationMessage(payload) + "\r\n"
}

func sanitizeSupplierNotificationText(value string, secrets ...string) string {
	result := value
	for _, secret := range secrets {
		if secret != "" {
			result = strings.ReplaceAll(result, secret, "[已隐藏]")
		}
	}
	return truncateSupplierNotificationText(result, supplierNotificationMaxResponse)
}

func truncateSupplierNotificationText(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
