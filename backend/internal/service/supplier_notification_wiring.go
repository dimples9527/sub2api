package service

import "github.com/google/wire"

// ProvideSupplierNotificationDispatcher 创建并启动供应商通知投递调度器。
func ProvideSupplierNotificationDispatcher(
	repo SupplierNotificationRepository,
	sender SupplierNotificationSender,
) *SupplierNotificationDispatcher {
	dispatcher := NewSupplierNotificationDispatcher(repo, sender)
	dispatcher.Start()
	return dispatcher
}

// ProvideSupplierNotificationService 创建通知渠道、订阅和投递管理服务。
func ProvideSupplierNotificationService(
	repo SupplierNotificationRepository,
	encryptor SecretEncryptor,
	sender SupplierNotificationSender,
) *SupplierNotificationService {
	return NewSupplierNotificationService(repo, encryptor, sender)
}

var SupplierNotificationWiringSet = wire.NewSet(
	NewSupplierNotificationSender,
	wire.Bind(new(SupplierNotificationSender), new(*supplierNotificationSender)),
	ProvideSupplierNotificationDispatcher,
	ProvideSupplierNotificationService,
	wire.Bind(new(SupplierBalanceAlertDispatcher), new(*SupplierNotificationDispatcher)),
	wire.Bind(new(SupplierGroupChangeNotifier), new(*SupplierNotificationDispatcher)),
)
