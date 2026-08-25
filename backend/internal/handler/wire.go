package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
)

// ProvideAdminHandlers creates the AdminHandlers struct
func ProvideAdminHandlers(
	dashboardHandler *admin.DashboardHandler,
	userHandler *admin.UserHandler,
	groupHandler *admin.GroupHandler,
	customPlatformHandler *admin.CustomPlatformHandler,
	accountHandler *admin.AccountHandler,
	customPlatformService service.CustomPlatformService,
	monitorGroupPlatformOverrideService service.MonitorGroupPlatformOverrideService,
	announcementHandler *admin.AnnouncementHandler,
	dataManagementHandler *admin.DataManagementHandler,
	backupHandler *admin.BackupHandler,
	oauthHandler *admin.OAuthHandler,
	openaiOAuthHandler *admin.OpenAIOAuthHandler,
	geminiOAuthHandler *admin.GeminiOAuthHandler,
	antigravityOAuthHandler *admin.AntigravityOAuthHandler,
	grokOAuthHandler *admin.GrokOAuthHandler,
	cnProviderHandler *admin.CNProviderHandler,
	proxyHandler *admin.ProxyHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	opsHandler *admin.OpsHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	usageHandler *admin.UsageHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	errorPassthroughHandler *admin.ErrorPassthroughHandler,
	tlsFingerprintProfileHandler *admin.TLSFingerprintProfileHandler,
	apiKeyHandler *admin.AdminAPIKeyHandler,
	scheduledTestHandler *admin.ScheduledTestHandler,
	channelHandler *admin.ChannelHandler,
	channelMonitorHandler *admin.ChannelMonitorHandler,
	channelMonitorTemplateHandler *admin.ChannelMonitorRequestTemplateHandler,
	contentModerationHandler *admin.ContentModerationHandler,
	promptAuditHandler *securityaudit.PromptAdminHandler,
	paymentHandler *admin.PaymentHandler,
	affiliateHandler *admin.AffiliateHandler,
	complianceHandler *admin.ComplianceHandler,
	auditLogHandler *admin.AuditLogHandler,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	supplierProviderHandler *admin.SupplierProviderHandler,
	supplierProviderAuthHandler *admin.SupplierProviderAuthHandler,
	supplierProviderTypeHandler *admin.SupplierProviderTypeHandler,
	supplierProviderSyncHandler *admin.SupplierProviderSyncHandler,
	supplierProviderRechargeHandler *admin.SupplierProviderRechargeHandler,
	supplierAutomationHandler *admin.SupplierAutomationHandler,
	supplierDashboardHandler *admin.SupplierDashboardHandler,
	supplierBalanceAlertHandler *admin.SupplierBalanceAlertHandler,
	supplierNotificationHandler *admin.SupplierNotificationHandler,
	supplierCostDeviationSettingsHandler *admin.SupplierCostDeviationSettingsHandler,
	supplierProviderCostReviewHandler *admin.SupplierProviderCostReviewHandler,
	upstreamProviderHandler *admin.UpstreamProviderHandler,
	upstreamDashboardHandler *admin.UpstreamDashboardHandler,
	upstreamManagementHandler *admin.UpstreamManagementHandler,
	upstreamAccountSyncHandler *admin.UpstreamAccountSyncHandler,
	ollamaCloudUsage *service.OllamaCloudUsageService,
) *AdminHandlers {
	accountHandler.SetUpstreamBillingProbeService(upstreamBillingProbe)
	accountHandler.SetOllamaCloudUsageService(ollamaCloudUsage)
	accountHandler.SetMonitorGroupPlatformOverrideService(monitorGroupPlatformOverrideService)
	groupHandler.SetCustomPlatformService(customPlatformService)
	groupHandler.SetMonitorGroupPlatformOverrideService(monitorGroupPlatformOverrideService)
	return &AdminHandlers{
		Dashboard:                     dashboardHandler,
		User:                          userHandler,
		Group:                         groupHandler,
		CustomPlatform:                customPlatformHandler,
		Account:                       accountHandler,
		Announcement:                  announcementHandler,
		DataManagement:                dataManagementHandler,
		Backup:                        backupHandler,
		OAuth:                         oauthHandler,
		OpenAIOAuth:                   openaiOAuthHandler,
		GeminiOAuth:                   geminiOAuthHandler,
		AntigravityOAuth:              antigravityOAuthHandler,
		GrokOAuth:                     grokOAuthHandler,
		CNProvider:                    cnProviderHandler,
		Proxy:                         proxyHandler,
		Redeem:                        redeemHandler,
		Promo:                         promoHandler,
		Setting:                       settingHandler,
		Ops:                           opsHandler,
		System:                        systemHandler,
		Subscription:                  subscriptionHandler,
		Usage:                         usageHandler,
		UserAttribute:                 userAttributeHandler,
		ErrorPassthrough:              errorPassthroughHandler,
		TLSFingerprintProfile:         tlsFingerprintProfileHandler,
		APIKey:                        apiKeyHandler,
		ScheduledTest:                 scheduledTestHandler,
		Channel:                       channelHandler,
		ChannelMonitor:                channelMonitorHandler,
		ChannelMonitorTemplate:        channelMonitorTemplateHandler,
		ContentModeration:             contentModerationHandler,
		PromptAudit:                   promptAuditHandler,
		Payment:                       paymentHandler,
		Affiliate:                     affiliateHandler,
		Compliance:                    complianceHandler,
		AuditLog:                      auditLogHandler,
		SupplierProvider:              supplierProviderHandler,
		SupplierProviderAuth:          supplierProviderAuthHandler,
		SupplierProviderType:          supplierProviderTypeHandler,
		SupplierProviderSync:          supplierProviderSyncHandler,
		SupplierProviderRecharge:      supplierProviderRechargeHandler,
		SupplierAutomation:            supplierAutomationHandler,
		SupplierDashboard:             supplierDashboardHandler,
		SupplierBalanceAlert:          supplierBalanceAlertHandler,
		SupplierNotification:          supplierNotificationHandler,
		SupplierCostDeviationSettings: supplierCostDeviationSettingsHandler,
		SupplierProviderCostReview:    supplierProviderCostReviewHandler,
		UpstreamProvider:              upstreamProviderHandler,
		UpstreamDashboard:             upstreamDashboardHandler,
		UpstreamManagement:            upstreamManagementHandler,
		UpstreamAccountSync:           upstreamAccountSyncHandler,
	}
}

func ProvideGatewayHandler(
	gatewayService *service.GatewayService,
	openAIGatewayService *service.OpenAIGatewayService,
	geminiCompatService *service.GeminiMessagesCompatService,
	antigravityGatewayService *service.AntigravityGatewayService,
	userService *service.UserService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	userMsgQueueService *service.UserMessageQueueService,
	cfg *config.Config,
	settingService *service.SettingService,
	coordinator *securityaudit.Coordinator,
) *GatewayHandler {
	h := NewGatewayHandler(gatewayService, openAIGatewayService, geminiCompatService, antigravityGatewayService,
		userService, concurrencyService, billingCacheService, usageService, apiKeyService, usageRecordWorkerPool,
		errorPassthroughService, contentModerationService, userMsgQueueService, cfg, settingService)
	h.securityAuditCoordinator = coordinator
	return h
}

func ProvideOpenAIGatewayHandler(
	gatewayService *service.OpenAIGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	opsService *service.OpsService,
	grokQuotaService *service.GrokQuotaService,
	cfg *config.Config,
	coordinator *securityaudit.Coordinator,
) *OpenAIGatewayHandler {
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, apiKeyService,
		usageRecordWorkerPool, errorPassthroughService, contentModerationService, opsService, cfg)
	h.securityAuditCoordinator = coordinator
	h.grokMediaEligibilityProber = grokQuotaService
	return h
}

func ProvideBatchImageHandler(
	batchService *service.BatchImagePublicService,
	download *service.BatchImageDownloadService,
	cleanup *service.BatchImageCleanupService,
	openAI *OpenAIGatewayHandler,
) *BatchImageHandler {
	h := NewBatchImageHandler(batchService, download, cleanup)
	h.openAI = openAI
	return h
}

// ProvideSystemHandler creates admin.SystemHandler with UpdateService
func ProvideSystemHandler(updateService *service.UpdateService, lockService *service.SystemOperationLockService) *admin.SystemHandler {
	return admin.NewSystemHandler(updateService, lockService)
}

// ProvideSettingHandler creates SettingHandler with version from BuildInfo
func ProvideSettingHandler(settingService *service.SettingService, buildInfo BuildInfo, notificationEmailService *service.NotificationEmailService) *SettingHandler {
	h := NewSettingHandler(settingService, buildInfo.Version)
	h.SetNotificationEmailService(notificationEmailService)
	return h
}

// ProvideAdminSettingHandler creates admin.SettingHandler with notification template APIs.
func ProvideAdminSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, aliyunCaptchaService *service.AliyunCaptchaService, opsService *service.OpsService, paymentConfigService *service.PaymentConfigService, paymentService *service.PaymentService, userAttributeService *service.UserAttributeService, notificationEmailService *service.NotificationEmailService, totpService *service.TotpService, userService *service.UserService) *admin.SettingHandler {
	h := admin.NewSettingHandler(settingService, emailService, turnstileService, opsService, paymentConfigService, paymentService, userAttributeService)
	h.SetNotificationEmailService(notificationEmailService)
	h.SetAliyunCaptchaService(aliyunCaptchaService)
	h.SetStepUpDeps(totpService, userService)
	return h
}

func ProvideSupplierAutomationHandler(svc *service.SupplierAutomationService) *admin.SupplierAutomationHandler {
	return admin.NewSupplierAutomationHandler(svc)
}

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	apiKeyHandler *APIKeyHandler,
	usageHandler *UsageHandler,
	redeemHandler *RedeemHandler,
	subscriptionHandler *SubscriptionHandler,
	announcementHandler *AnnouncementHandler,
	channelMonitorUserHandler *ChannelMonitorUserHandler,
	channelMonitorV2Handler *ChannelMonitorV2Handler,
	adminHandlers *AdminHandlers,
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	settingHandler *SettingHandler,
	totpHandler *TotpHandler,
	passkeyHandler *PasskeyHandler,
	paymentHandler *PaymentHandler,
	paymentWebhookHandler *PaymentWebhookHandler,
	availableChannelHandler *AvailableChannelHandler,
	modelPlazaHandler *ModelPlazaHandler,
	modelSquareHandler *ModelSquareHandler,
	asyncImageHandler *AsyncImageHandler,
	batchImageHandler *BatchImageHandler,
	_ *service.IdempotencyCoordinator,
	_ *service.IdempotencyCleanupService,
) *Handlers {
	return &Handlers{
		Auth:             authHandler,
		User:             userHandler,
		APIKey:           apiKeyHandler,
		Usage:            usageHandler,
		Redeem:           redeemHandler,
		Subscription:     subscriptionHandler,
		Announcement:     announcementHandler,
		ChannelMonitor:   channelMonitorUserHandler,
		ChannelMonitorV2: channelMonitorV2Handler,
		Admin:            adminHandlers,
		Gateway:          gatewayHandler,
		OpenAIGateway:    openaiGatewayHandler,
		Setting:          settingHandler,
		Totp:             totpHandler,
		Passkey:          passkeyHandler,
		Payment:          paymentHandler,
		PaymentWebhook:   paymentWebhookHandler,
		AvailableChannel: availableChannelHandler,
		ModelPlaza:       modelPlazaHandler,
		ModelSquare:      modelSquareHandler,
		AsyncImage:       asyncImageHandler,
		BatchImage:       batchImageHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Top-level handlers
	NewAuthHandler,
	NewUserHandler,
	NewAPIKeyHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	NewChannelMonitorUserHandler,
	NewChannelMonitorV2Handler,
	ProvideGatewayHandler,
	ProvideOpenAIGatewayHandler,
	NewTotpHandler,
	NewPasskeyHandler,
	ProvideSettingHandler,
	NewPaymentHandler,
	NewPaymentWebhookHandler,
	NewAvailableChannelHandler,
	NewModelPlazaHandler,
	NewModelSquareHandler,
	NewAsyncImageHandler,
	ProvideBatchImageHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	admin.NewUserHandler,
	admin.NewGroupHandler,
	admin.NewCustomPlatformHandler,
	admin.ProvideAccountHandler,
	admin.NewAnnouncementHandler,
	admin.NewDataManagementHandler,
	admin.NewBackupHandler,
	admin.NewOAuthHandler,
	admin.NewOpenAIOAuthHandler,
	admin.NewGeminiOAuthHandler,
	admin.NewAntigravityOAuthHandler,
	admin.NewGrokOAuthHandler,
	admin.NewCNProviderHandler,
	admin.NewProxyHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	ProvideAdminSettingHandler,
	admin.NewOpsHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUsageHandler,
	admin.NewUserAttributeHandler,
	admin.NewErrorPassthroughHandler,
	admin.NewTLSFingerprintProfileHandler,
	admin.NewAdminAPIKeyHandler,
	admin.NewScheduledTestHandler,
	admin.NewChannelHandler,
	admin.NewChannelMonitorHandler,
	admin.NewChannelMonitorRequestTemplateHandler,
	admin.NewContentModerationHandler,
	admin.NewPaymentHandler,
	admin.NewAffiliateHandler,
	admin.NewComplianceHandler,
	admin.NewAuditLogHandler,
	admin.NewSupplierProviderHandler,
	admin.NewSupplierProviderTypeHandler,
	SupplierProviderWiringSet,
	ProvideSupplierAutomationHandler,
	admin.SupplierDashboardWiringSet,
	admin.SupplierBalanceAlertWiringSet,
	admin.SupplierNotificationWiringSet,
	admin.NewSupplierCostDeviationSettingsHandler,
	admin.NewUpstreamProviderHandler,
	admin.NewUpstreamDashboardHandler,
	admin.NewUpstreamManagementHandler,
	admin.NewUpstreamAccountSyncHandler,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
