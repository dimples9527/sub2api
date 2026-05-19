package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html"
	"log"
	"math/big"
	"mime"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrEmailNotConfigured    = infraerrors.ServiceUnavailable("EMAIL_NOT_CONFIGURED", "email service not configured")
	ErrInvalidVerifyCode     = infraerrors.BadRequest("INVALID_VERIFY_CODE", "invalid or expired verification code")
	ErrVerifyCodeTooFrequent = infraerrors.TooManyRequests("VERIFY_CODE_TOO_FREQUENT", "please wait before requesting a new code")
	ErrVerifyCodeMaxAttempts = infraerrors.TooManyRequests("VERIFY_CODE_MAX_ATTEMPTS", "too many failed attempts, please request a new code")

	// Password reset errors
	ErrInvalidResetToken = infraerrors.BadRequest("INVALID_RESET_TOKEN", "invalid or expired password reset token")
)

// EmailCache defines cache operations for email service
type EmailCache interface {
	GetVerificationCode(ctx context.Context, email string) (*VerificationCodeData, error)
	SetVerificationCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error
	DeleteVerificationCode(ctx context.Context, email string) error

	// Password reset token methods
	GetPasswordResetToken(ctx context.Context, email string) (*PasswordResetTokenData, error)
	SetPasswordResetToken(ctx context.Context, email string, data *PasswordResetTokenData, ttl time.Duration) error
	DeletePasswordResetToken(ctx context.Context, email string) error

	// Password reset email cooldown methods
	// Returns true if in cooldown period (email was sent recently)
	IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool
	SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error
}

// VerificationCodeData represents verification code data
type VerificationCodeData struct {
	Code      string
	Attempts  int
	CreatedAt time.Time
}

// PasswordResetTokenData represents password reset token data
type PasswordResetTokenData struct {
	Token     string
	CreatedAt time.Time
}

const (
	verifyCodeTTL         = 15 * time.Minute
	verifyCodeCooldown    = 1 * time.Minute
	maxVerifyCodeAttempts = 5

	// Password reset token settings
	passwordResetTokenTTL = 30 * time.Minute

	// Password reset email cooldown (prevent email bombing)
	passwordResetEmailCooldown = 30 * time.Second
)

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

func emailSenderDisplayName(config *SMTPConfig) string {
	if config != nil {
		if name := strings.TrimSpace(config.FromName); name != "" {
			return name
		}
		if from := strings.TrimSpace(config.From); from != "" {
			return from
		}
	}
	return "系统通知"
}

// EmailService 邮件服务
type EmailService struct {
	settingRepo SettingRepository
	cache       EmailCache
}

func smtpConfigLogSummary(config *SMTPConfig) string {
	if config == nil {
		return "host= port=0 username_set=false password_set=false from= from_name_set=false use_tls=false"
	}
	return fmt.Sprintf("host=%s port=%d username_set=%t password_set=%t from=%s from_name_set=%t use_tls=%t",
		config.Host,
		config.Port,
		strings.TrimSpace(config.Username) != "",
		strings.TrimSpace(config.Password) != "",
		config.From,
		strings.TrimSpace(config.FromName) != "",
		config.UseTLS,
	)
}

// NewEmailService 创建邮件服务实例
func NewEmailService(settingRepo SettingRepository, cache EmailCache) *EmailService {
	return &EmailService{
		settingRepo: settingRepo,
		cache:       cache,
	}
}

// GetSMTPConfig 从数据库获取SMTP配置
func (s *EmailService) GetSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	keys := []string{
		SettingKeySMTPHost,
		SettingKeySMTPPort,
		SettingKeySMTPUsername,
		SettingKeySMTPPassword,
		SettingKeySMTPFrom,
		SettingKeySMTPFromName,
		SettingKeySMTPUseTLS,
	}

	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get smtp settings: %w", err)
	}

	host := strings.TrimSpace(settings[SettingKeySMTPHost])
	if host == "" {
		return nil, ErrEmailNotConfigured
	}

	port := 587 // 默认端口
	if portStr := settings[SettingKeySMTPPort]; portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	useTLS := settings[SettingKeySMTPUseTLS] == "true"

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: strings.TrimSpace(settings[SettingKeySMTPUsername]),
		Password: strings.TrimSpace(settings[SettingKeySMTPPassword]),
		From:     strings.TrimSpace(settings[SettingKeySMTPFrom]),
		FromName: strings.TrimSpace(settings[SettingKeySMTPFromName]),
		UseTLS:   useTLS,
	}, nil
}

// SendEmail 发送邮件（使用数据库中保存的配置）
func (s *EmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	config, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return err
	}
	return s.SendEmailWithConfig(config, to, subject, body)
}

// SendEmailWithConfig 使用指定配置发送邮件
func (s *EmailService) SendEmailWithConfig(config *SMTPConfig, to, subject, body string) error {
	startedAt := time.Now()
	logger.LegacyPrintf("service.email", "[Email] SendEmailWithConfig start to=%s subject=%s %s", to, subject, smtpConfigLogSummary(config))

	from := config.From
	if config.FromName != "" {
		from = (&mail.Address{Name: config.FromName, Address: config.From}).String()
	}

	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, encodedSubject, body)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	if config.UseTLS {
		if err := s.sendMailTLS(addr, auth, config.From, to, []byte(msg), config.Host); err != nil {
			logger.LegacyPrintf("service.email", "[Email] SendEmailWithConfig failed to=%s elapsed_ms=%d error=%v", to, time.Since(startedAt).Milliseconds(), err)
			return err
		}
		logger.LegacyPrintf("service.email", "[Email] SendEmailWithConfig success to=%s elapsed_ms=%d", to, time.Since(startedAt).Milliseconds())
		return nil
	}

	if err := smtp.SendMail(addr, auth, config.From, []string{to}, []byte(msg)); err != nil {
		logger.LegacyPrintf("service.email", "[Email] SendEmailWithConfig failed to=%s elapsed_ms=%d error=%v", to, time.Since(startedAt).Milliseconds(), err)
		return err
	}
	logger.LegacyPrintf("service.email", "[Email] SendEmailWithConfig success to=%s elapsed_ms=%d", to, time.Since(startedAt).Milliseconds())
	return nil
}

// sendMailTLS 使用TLS发送邮件
func (s *EmailService) sendMailTLS(addr string, auth smtp.Auth, from, to string, msg []byte, host string) error {
	startedAt := time.Now()
	logger.LegacyPrintf("service.email", "[Email] TLS smtp dial start addr=%s host=%s from=%s to=%s", addr, host, from, to)

	tlsConfig := &tls.Config{
		ServerName: host,
		// 强制 TLS 1.2+，避免协议降级导致的弱加密风险。
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp dial failed addr=%s elapsed_ms=%d error=%v", addr, time.Since(startedAt).Milliseconds(), err)
		return fmt.Errorf("tls dial: %w", err)
	}
	defer func() { _ = conn.Close() }()
	logger.LegacyPrintf("service.email", "[Email] TLS smtp dial success addr=%s elapsed_ms=%d", addr, time.Since(startedAt).Milliseconds())

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp client creation failed host=%s error=%v", host, err)
		return fmt.Errorf("new smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err = client.Auth(auth); err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp auth failed host=%s error=%v", host, err)
		return fmt.Errorf("smtp auth: %w", err)
	}
	logger.LegacyPrintf("service.email", "[Email] TLS smtp auth success host=%s", host)

	if err = client.Mail(from); err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp MAIL FROM failed from=%s error=%v", from, err)
		return fmt.Errorf("smtp mail: %w", err)
	}
	logger.LegacyPrintf("service.email", "[Email] TLS smtp MAIL FROM accepted from=%s", from)

	if err = client.Rcpt(to); err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp RCPT TO failed to=%s error=%v", to, err)
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	logger.LegacyPrintf("service.email", "[Email] TLS smtp RCPT TO accepted to=%s", to)

	w, err := client.Data()
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp DATA command failed to=%s error=%v", to, err)
		return fmt.Errorf("smtp data: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp DATA write failed to=%s error=%v", to, err)
		return fmt.Errorf("write msg: %w", err)
	}

	err = w.Close()
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] TLS smtp DATA close failed to=%s error=%v", to, err)
		return fmt.Errorf("close writer: %w", err)
	}
	logger.LegacyPrintf("service.email", "[Email] TLS smtp message accepted to=%s elapsed_ms=%d", to, time.Since(startedAt).Milliseconds())

	// Email is sent successfully after w.Close(), ignore Quit errors
	// Some SMTP servers return non-standard responses on QUIT
	_ = client.Quit()
	return nil
}

// GenerateVerifyCode 生成6位数字验证码
func (s *EmailService) GenerateVerifyCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}
	return string(code), nil
}

// SendVerifyCode 发送验证码邮件
func (s *EmailService) SendVerifyCode(ctx context.Context, email, siteName string) error {
	startedAt := time.Now()
	logger.LegacyPrintf("service.email", "[Email] SendVerifyCode start email=%s site=%s", email, siteName)

	// 检查是否在冷却期内
	existing, err := s.cache.GetVerificationCode(ctx, email)
	if err == nil && existing != nil {
		if time.Since(existing.CreatedAt) < verifyCodeCooldown {
			logger.LegacyPrintf("service.email", "[Email] SendVerifyCode blocked by cooldown email=%s age_ms=%d cooldown_ms=%d", email, time.Since(existing.CreatedAt).Milliseconds(), verifyCodeCooldown.Milliseconds())
			return ErrVerifyCodeTooFrequent
		}
		logger.LegacyPrintf("service.email", "[Email] Existing verification code found but cooldown expired email=%s age_ms=%d", email, time.Since(existing.CreatedAt).Milliseconds())
	} else if err != nil {
		logger.LegacyPrintf("service.email", "[Email] No reusable verification code found email=%s cache_error=%v", email, err)
	}

	// 生成验证码
	code, err := s.GenerateVerifyCode()
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] Generate verification code failed email=%s error=%v", email, err)
		return fmt.Errorf("generate code: %w", err)
	}
	logger.LegacyPrintf("service.email", "[Email] Verification code generated email=%s code_length=%d", email, len(code))

	// 保存验证码到 Redis
	data := &VerificationCodeData{
		Code:      code,
		Attempts:  0,
		CreatedAt: time.Now(),
	}
	if err := s.cache.SetVerificationCode(ctx, email, data, verifyCodeTTL); err != nil {
		logger.LegacyPrintf("service.email", "[Email] Save verification code failed email=%s ttl_seconds=%d error=%v", email, int(verifyCodeTTL.Seconds()), err)
		return fmt.Errorf("save verify code: %w", err)
	}
	logger.LegacyPrintf("service.email", "[Email] Verification code saved email=%s ttl_seconds=%d", email, int(verifyCodeTTL.Seconds()))

	config, err := s.GetSMTPConfig(ctx)
	if err != nil {
		logger.LegacyPrintf("service.email", "[Email] Get SMTP config failed email=%s error=%v", email, err)
		return err
	}
	logger.LegacyPrintf("service.email", "[Email] SMTP config loaded for verification email=%s %s", email, smtpConfigLogSummary(config))

	senderName := emailSenderDisplayName(config)

	// 构建邮件内容
	subject := fmt.Sprintf("[%s] 邮箱验证码", senderName)
	body := s.buildVerifyCodeEmailBody(code, senderName)

	// 发送邮件
	if err := s.SendEmailWithConfig(config, email, subject, body); err != nil {
		logger.LegacyPrintf("service.email", "[Email] Send verification email failed email=%s elapsed_ms=%d error=%v", email, time.Since(startedAt).Milliseconds(), err)
		return fmt.Errorf("send email: %w", err)
	}

	logger.LegacyPrintf("service.email", "[Email] SendVerifyCode success email=%s elapsed_ms=%d", email, time.Since(startedAt).Milliseconds())
	return nil
}

// VerifyCode 验证验证码
func (s *EmailService) VerifyCode(ctx context.Context, email, code string) error {
	data, err := s.cache.GetVerificationCode(ctx, email)
	if err != nil || data == nil {
		return ErrInvalidVerifyCode
	}

	// 检查是否已达到最大尝试次数
	if data.Attempts >= maxVerifyCodeAttempts {
		return ErrVerifyCodeMaxAttempts
	}

	// 验证码不匹配 (constant-time comparison to prevent timing attacks)
	if subtle.ConstantTimeCompare([]byte(data.Code), []byte(code)) != 1 {
		data.Attempts++
		if err := s.cache.SetVerificationCode(ctx, email, data, verifyCodeTTL); err != nil {
			log.Printf("[Email] Failed to update verification attempt count: %v", err)
		}
		if data.Attempts >= maxVerifyCodeAttempts {
			return ErrVerifyCodeMaxAttempts
		}
		return ErrInvalidVerifyCode
	}

	// 验证成功，删除验证码
	if err := s.cache.DeleteVerificationCode(ctx, email); err != nil {
		log.Printf("[Email] Failed to delete verification code after success: %v", err)
	}
	return nil
}

// buildVerifyCodeEmailBody 构建验证码邮件HTML内容
func (s *EmailService) buildVerifyCodeEmailBody(code, siteName string) string {
	escapedSiteName := html.EscapeString(siteName)
	escapedCode := html.EscapeString(code)

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { margin: 0; padding: 0; background: #f4f1ea; color: #202738; font-family: "PingFang SC", "Microsoft YaHei", "Segoe UI", sans-serif; }
        .email-shell { width: 100%%; background: linear-gradient(180deg, #fbfaf6 0%%, #efe9db 100%%); padding: 32px 12px; }
        .container { max-width: 640px; margin: 0 auto; border-radius: 14px; overflow: hidden; background: #fffdf8; border: 1px solid rgba(173, 135, 54, 0.28); box-shadow: 0 24px 70px rgba(69, 56, 30, 0.16); }
        .header { position: relative; padding: 34px 38px 30px; text-align: left; background: linear-gradient(135deg, rgba(255,255,255,0.96), rgba(244,236,219,0.94)); border-bottom: 1px solid rgba(173, 135, 54, 0.22); }
        .brand { color: #a77b28; font-size: 13px; font-weight: 700; letter-spacing: 2px; }
        .badge { display: inline-block; margin-top: 16px; padding: 6px 12px; color: #85611e; border: 1px solid rgba(173, 135, 54, 0.34); border-radius: 999px; background: rgba(214, 177, 95, 0.11); font-size: 12px; letter-spacing: 1px; }
        h1 { margin: 16px 0 10px; color: #1f2a44; font-size: 28px; line-height: 1.28; font-weight: 800; }
        .subtitle { margin: 0; max-width: 470px; color: #5c6578; font-size: 15px; line-height: 1.75; }
        .content { padding: 36px 38px 34px; text-align: center; background: #fffdf8; }
        .label { margin: 0 0 14px; color: #9a7430; font-size: 15px; font-weight: 700; letter-spacing: 1px; }
        .code { display: inline-block; min-width: 270px; margin: 4px 0 24px; padding: 18px 28px; border: 1px solid rgba(173, 135, 54, 0.38); border-radius: 12px; background: linear-gradient(135deg, #fff8e7 0%%, #f1e5c7 100%%); color: #795415; font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace; font-size: 38px; line-height: 1; font-weight: 800; letter-spacing: 9px; box-shadow: inset 0 1px 0 rgba(255,255,255,0.8); }
        .info { margin: 0 auto; max-width: 500px; color: #5c6578; font-size: 14px; line-height: 1.8; }
        .info strong { color: #9a7430; }
        .notice { margin-top: 18px; padding: 14px 16px; border-radius: 10px; background: #fbf6ea; border: 1px solid rgba(173, 135, 54, 0.22); color: #515b70; }
        .footer { padding: 20px 30px 24px; text-align: center; color: #7a8293; font-size: 12px; line-height: 1.7; border-top: 1px solid rgba(173, 135, 54, 0.18); background: #f8f3e7; }
        @media (max-width: 520px) { .header, .content { padding-left: 24px; padding-right: 24px; } .code { min-width: 0; width: auto; font-size: 31px; letter-spacing: 6px; } h1 { font-size: 24px; } }
    </style>
</head>
<body>
    <div class="email-shell">
        <div class="container">
            <div class="header">
                <div class="brand">%s</div>
                <div class="badge">SECURE ACCESS</div>
                <h1>邮箱安全验证</h1>
                <p class="subtitle">系统已收到一次账户验证请求。请使用下方验证码完成操作，切勿转发或透露给他人。</p>
            </div>
            <div class="content">
                <p class="label">您的专属验证码</p>
                <div class="code">%s</div>
                <div class="info">
                    <p>验证码 <strong>15 分钟内有效</strong>，超时后请重新获取。</p>
                    <p class="notice">如果这不是您本人操作，可以忽略本邮件；您的账户信息不会因此发生变化。</p>
                </div>
            </div>
            <div class="footer">
                <p>这是一封系统自动发送的安全通知，请勿直接回复。</p>
            </div>
        </div>
    </div>
</body>
</html>
`, escapedSiteName, escapedCode)
}

// TestSMTPConnectionWithConfig 使用指定配置测试SMTP连接
func (s *EmailService) TestSMTPConnectionWithConfig(config *SMTPConfig) error {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	if config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: config.Host,
			// 与发送逻辑一致，显式要求 TLS 1.2+。
			MinVersion: tls.VersionTLS12,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls connection failed: %w", err)
		}
		defer func() { _ = conn.Close() }()

		client, err := smtp.NewClient(conn, config.Host)
		if err != nil {
			return fmt.Errorf("smtp client creation failed: %w", err)
		}
		defer func() { _ = client.Close() }()

		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}

		return client.Quit()
	}

	// 非TLS连接测试
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp connection failed: %w", err)
	}
	defer func() { _ = client.Close() }()

	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp authentication failed: %w", err)
	}

	return client.Quit()
}

// GeneratePasswordResetToken generates a secure 32-byte random token (64 hex characters)
func (s *EmailService) GeneratePasswordResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendPasswordResetEmail sends a password reset email with a reset link
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, email, siteName, resetURL string) error {
	var token string
	var needSaveToken bool

	// Check if token already exists
	existing, err := s.cache.GetPasswordResetToken(ctx, email)
	if err == nil && existing != nil {
		// Token exists, reuse it (allows resending email without generating new token)
		token = existing.Token
		needSaveToken = false
	} else {
		// Generate new token
		token, err = s.GeneratePasswordResetToken()
		if err != nil {
			return fmt.Errorf("generate token: %w", err)
		}
		needSaveToken = true
	}

	// Save token to Redis (only if new token generated)
	if needSaveToken {
		data := &PasswordResetTokenData{
			Token:     token,
			CreatedAt: time.Now(),
		}
		if err := s.cache.SetPasswordResetToken(ctx, email, data, passwordResetTokenTTL); err != nil {
			return fmt.Errorf("save reset token: %w", err)
		}
	}

	// Build full reset URL with URL-encoded token and email
	fullResetURL := fmt.Sprintf("%s?email=%s&token=%s", resetURL, url.QueryEscape(email), url.QueryEscape(token))

	config, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return err
	}

	senderName := emailSenderDisplayName(config)

	// Build email content
	subject := fmt.Sprintf("[%s] 密码重置请求", senderName)
	body := s.buildPasswordResetEmailBody(fullResetURL, senderName)

	// Send email
	if err := s.SendEmailWithConfig(config, email, subject, body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

// SendPasswordResetEmailWithCooldown sends password reset email with cooldown check (called by queue worker)
// This method wraps SendPasswordResetEmail with email cooldown to prevent email bombing
func (s *EmailService) SendPasswordResetEmailWithCooldown(ctx context.Context, email, siteName, resetURL string) error {
	// Check email cooldown to prevent email bombing
	if s.cache.IsPasswordResetEmailInCooldown(ctx, email) {
		log.Printf("[Email] Password reset email skipped (cooldown): %s", email)
		return nil // Silent success to prevent revealing cooldown to attackers
	}

	// Send email using core method
	if err := s.SendPasswordResetEmail(ctx, email, siteName, resetURL); err != nil {
		return err
	}

	// Set cooldown marker (Redis TTL handles expiration)
	if err := s.cache.SetPasswordResetEmailCooldown(ctx, email, passwordResetEmailCooldown); err != nil {
		log.Printf("[Email] Failed to set password reset cooldown for %s: %v", email, err)
	}

	return nil
}

// VerifyPasswordResetToken verifies the password reset token without consuming it
func (s *EmailService) VerifyPasswordResetToken(ctx context.Context, email, token string) error {
	data, err := s.cache.GetPasswordResetToken(ctx, email)
	if err != nil || data == nil {
		return ErrInvalidResetToken
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(data.Token), []byte(token)) != 1 {
		return ErrInvalidResetToken
	}

	return nil
}

// ConsumePasswordResetToken verifies and deletes the token (one-time use)
func (s *EmailService) ConsumePasswordResetToken(ctx context.Context, email, token string) error {
	// Verify first
	if err := s.VerifyPasswordResetToken(ctx, email, token); err != nil {
		return err
	}

	// Delete after verification (one-time use)
	if err := s.cache.DeletePasswordResetToken(ctx, email); err != nil {
		log.Printf("[Email] Failed to delete password reset token after consumption: %v", err)
	}
	return nil
}

// buildPasswordResetEmailBody builds the HTML content for password reset email
func (s *EmailService) buildPasswordResetEmailBody(resetURL, siteName string) string {
	escapedSiteName := html.EscapeString(siteName)
	escapedResetURL := html.EscapeString(resetURL)

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { margin: 0; padding: 0; background: #f4f1ea; color: #202738; font-family: "PingFang SC", "Microsoft YaHei", "Segoe UI", sans-serif; }
        .email-shell { width: 100%%; background: linear-gradient(180deg, #fbfaf6 0%%, #efe9db 100%%); padding: 32px 12px; }
        .container { max-width: 640px; margin: 0 auto; border-radius: 14px; overflow: hidden; background: #fffdf8; border: 1px solid rgba(173, 135, 54, 0.28); box-shadow: 0 24px 70px rgba(69, 56, 30, 0.16); }
        .header { position: relative; padding: 34px 38px 30px; text-align: left; background: linear-gradient(135deg, rgba(255,255,255,0.96), rgba(244,236,219,0.94)); border-bottom: 1px solid rgba(173, 135, 54, 0.22); }
        .brand { color: #a77b28; font-size: 13px; font-weight: 700; letter-spacing: 2px; }
        .badge { display: inline-block; margin-top: 16px; padding: 6px 12px; color: #85611e; border: 1px solid rgba(173, 135, 54, 0.34); border-radius: 999px; background: rgba(214, 177, 95, 0.11); font-size: 12px; letter-spacing: 1px; }
        h1 { margin: 16px 0 10px; color: #1f2a44; font-size: 28px; line-height: 1.28; font-weight: 800; }
        .subtitle { margin: 0; max-width: 470px; color: #5c6578; font-size: 15px; line-height: 1.75; }
        .content { padding: 36px 38px 34px; text-align: center; background: #fffdf8; }
        .button { display: inline-block; margin: 6px 0 24px; padding: 15px 34px; border-radius: 10px; background: linear-gradient(135deg, #d8b564 0%%, #a77b28 100%%); color: #ffffff; text-decoration: none; font-size: 16px; font-weight: 800; letter-spacing: 1px; box-shadow: 0 14px 30px rgba(167,123,40,0.24); }
        .info { margin: 0 auto; max-width: 500px; color: #5c6578; font-size: 14px; line-height: 1.8; }
        .info strong { color: #9a7430; }
        .notice { margin-top: 18px; padding: 14px 16px; border-radius: 10px; background: #fbf6ea; border: 1px solid rgba(173, 135, 54, 0.22); color: #515b70; }
        .fallback { margin-top: 20px; padding: 15px; border: 1px solid rgba(31, 42, 68, 0.10); border-radius: 10px; background: #f7f5ef; color: #6b7280; font-size: 12px; line-height: 1.7; text-align: left; word-break: break-all; }
        .footer { padding: 20px 30px 24px; text-align: center; color: #7a8293; font-size: 12px; line-height: 1.7; border-top: 1px solid rgba(173, 135, 54, 0.18); background: #f8f3e7; }
        @media (max-width: 520px) { .header, .content { padding-left: 24px; padding-right: 24px; } h1 { font-size: 24px; } .button { display: block; } }
    </style>
</head>
<body>
    <div class="email-shell">
        <div class="container">
            <div class="header">
                <div class="brand">%s</div>
                <div class="badge">ACCOUNT RECOVERY</div>
                <h1>密码重置请求</h1>
                <p class="subtitle">系统已为您的账户生成一次性安全链接。请确认是本人操作后继续。</p>
            </div>
            <div class="content">
                <a href="%s" class="button">重置登录密码</a>
                <div class="info">
                    <p>此安全链接 <strong>30 分钟内有效</strong>，使用后将自动失效。</p>
                    <p class="notice">如果这不是您本人发起的请求，请忽略本邮件；当前密码不会被更改。</p>
                </div>
                <div class="fallback">
                    <p>如果按钮无法打开，请复制以下链接到浏览器访问：</p>
                    <p>%s</p>
                </div>
            </div>
            <div class="footer">
                <p>这是一封系统自动发送的安全通知，请勿直接回复。</p>
            </div>
        </div>
    </div>
</body>
</html>
`, escapedSiteName, escapedResetURL, escapedResetURL)
}
