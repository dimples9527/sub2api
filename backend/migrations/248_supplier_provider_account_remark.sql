-- 上游账号行的专属备注（供应商账号管理页的「备注维护」）。
--
-- 为什么不复用本地账号的 accounts.notes：备注描述的是「上游这一行」本身
-- （供应商侧口径说明、临时标记等），未匹配到本地账号的行同样需要它；
-- 而 accounts.notes 在账号管理页另有语义，两者混用会导致改一处、另一处跟着变。
--
-- 不参与同步覆盖：ReplaceAccounts 的 upsert 只写它自己列出的字段，
-- 上游重新同步不会清空人工维护的备注。
ALTER TABLE supplier_provider_accounts
    ADD COLUMN IF NOT EXISTS remark TEXT NOT NULL DEFAULT '';
