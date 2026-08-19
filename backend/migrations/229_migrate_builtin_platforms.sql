-- 将历史自定义平台配置归并到框架内置平台。
-- glm 是旧版对智谱 GLM 的业务别名，当前框架统一使用 zhipu。
UPDATE monitor_group_platform_overrides
SET actual_platform = 'zhipu', updated_at = NOW()
WHERE LOWER(TRIM(actual_platform)) = 'glm';

UPDATE supplier_local_account_platform_overrides
SET platform = 'zhipu', updated_at = NOW()
WHERE LOWER(TRIM(platform)) = 'glm';

-- 仅软删除自定义平台字典项，不修改历史用量、账单和统计记录。
UPDATE custom_platforms
SET deleted_at = COALESCE(deleted_at, NOW()), updated_at = NOW()
WHERE LOWER(TRIM(code)) IN ('glm', 'kimi', 'deepseek')
  AND deleted_at IS NULL;
