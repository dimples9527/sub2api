package service

import (
	"errors"
	"fmt"
)

// SupplierProviderAuthFailureError 表示供应商账号认证失败，而不是普通网络或基础设施错误。
type SupplierProviderAuthFailureError struct {
	Err error
}

func (e *SupplierProviderAuthFailureError) Error() string {
	if e == nil || e.Err == nil {
		return "supplier provider authentication failed"
	}
	return e.Err.Error()
}

func (e *SupplierProviderAuthFailureError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapSupplierProviderAuthFailure(err error) error {
	if err == nil {
		return nil
	}
	if IsSupplierProviderAuthFailure(err) {
		return err
	}
	return &SupplierProviderAuthFailureError{Err: err}
}

func IsSupplierProviderAuthFailure(err error) bool {
	var target *SupplierProviderAuthFailureError
	return errors.As(err, &target)
}

// SupplierProviderSessionFailureError 表示无法获取或准备供应商登录会话。
// 它不等同于账号认证失败，例如缓存、登录锁或网络基础设施异常不应自动停用供应商。
type SupplierProviderSessionFailureError struct {
	Err error
}

func (e *SupplierProviderSessionFailureError) Error() string {
	if e == nil || e.Err == nil {
		return "supplier provider session preparation failed"
	}
	return e.Err.Error()
}

func (e *SupplierProviderSessionFailureError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapSupplierProviderSessionFailure(err error) error {
	if err == nil || IsSupplierProviderSessionFailure(err) {
		return err
	}
	return &SupplierProviderSessionFailureError{Err: err}
}

func IsSupplierProviderSessionFailure(err error) bool {
	var target *SupplierProviderSessionFailureError
	return errors.As(err, &target)
}

func supplierProviderSessionFailureWithFinishError(sessionErr, finishErr error) error {
	if sessionErr == nil {
		return finishErr
	}
	if finishErr == nil {
		return sessionErr
	}
	return fmt.Errorf("%w；完成供应商成本回补运行记录失败: %v", sessionErr, finishErr)
}

const supplierProviderAuthFailureDisableMessage = "供应商登录失败，已自动停用。请检查账号、密码和打码配置，处理完成后手动重新启用。"
