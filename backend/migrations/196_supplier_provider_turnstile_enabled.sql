-- 供应商上游登录 Turnstile 自动打码开关（默认关闭）
ALTER TABLE supplier_providers
    ADD COLUMN IF NOT EXISTS turnstile_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN supplier_providers.turnstile_enabled IS '是否在登录该上游前调用全局 2Captcha 求解 Turnstile；失败将直接导致登录失败';
