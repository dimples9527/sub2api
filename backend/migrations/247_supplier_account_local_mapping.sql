-- 上游账号 → 本地账号的「持久化绑定」。
--
-- 背景：在此之前，列表页每次都用 supplierProviderLocalAccountMatchCondition 按**名字**现算关联，
-- 而该判据的 6 个候选全部由上游「当前名」派生。上游一改密钥名，本地旧名就对不上 ——
-- 行变成未匹配、本地账号列空白，且没有任何入口能把它改回去（快捷改名挂在 canManageLocalAccount 下）。
--
-- 上游账号的 upstream_account_key（上游密钥 id）改名不变，是天然稳定键，所以把「首次匹配到的
-- 本地账号」记下来，之后改名就不再断链。分组侧早已这么做（182/183 的 supplier_provider_groups.local_group_id），
-- 这里补齐账号侧。
--
-- 与分组侧一致：ON DELETE SET NULL，本地账号被硬删时绑定自动失效（软删由 reconcile 清理）。
ALTER TABLE supplier_provider_accounts
    ADD COLUMN IF NOT EXISTS local_account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS auto_match_status VARCHAR(32) NOT NULL DEFAULT 'unmatched',
    ADD COLUMN IF NOT EXISTS matched_upstream_name VARCHAR(255) NULL;

-- 绑定方向的反查（由本地账号找上游账号，供健康守护等链路使用）。
CREATE INDEX IF NOT EXISTS idx_supplier_provider_accounts_local_account
    ON supplier_provider_accounts (local_account_id)
    WHERE local_account_id IS NOT NULL;

-- 自动绑定的扫描范围：尚未绑定的活跃行。
CREATE INDEX IF NOT EXISTS idx_supplier_provider_accounts_auto_match
    ON supplier_provider_accounts (provider_id, active)
    WHERE local_account_id IS NULL;
