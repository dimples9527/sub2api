-- 自定义平台增加颜色字段，用于列表与监控页按平台区分配色。
ALTER TABLE custom_platforms
    ADD COLUMN IF NOT EXISTS color VARCHAR(16) NOT NULL DEFAULT '#64748b';

-- 为默认种子平台补齐品牌色（仅当仍为默认色时更新，避免覆盖用户已配置的颜色）。
UPDATE custom_platforms
SET color = '#2563eb'
WHERE code = 'glm' AND deleted_at IS NULL AND color = '#64748b';

UPDATE custom_platforms
SET color = '#4f46e5'
WHERE code = 'deepseek' AND deleted_at IS NULL AND color = '#64748b';

UPDATE custom_platforms
SET color = '#db2777'
WHERE code = 'kimi' AND deleted_at IS NULL AND color = '#64748b';
