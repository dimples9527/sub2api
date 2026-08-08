CREATE TABLE IF NOT EXISTS custom_platforms (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_platforms_code_active
    ON custom_platforms(code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_custom_platforms_enabled_sort
    ON custom_platforms(enabled, sort_order, id)
    WHERE deleted_at IS NULL;

INSERT INTO custom_platforms (code, name, enabled, sort_order)
SELECT 'glm', 'GLM', TRUE, 10
WHERE NOT EXISTS (
    SELECT 1 FROM custom_platforms WHERE code = 'glm' AND deleted_at IS NULL
);

INSERT INTO custom_platforms (code, name, enabled, sort_order)
SELECT 'deepseek', 'DeepSeek', TRUE, 20
WHERE NOT EXISTS (
    SELECT 1 FROM custom_platforms WHERE code = 'deepseek' AND deleted_at IS NULL
);

INSERT INTO custom_platforms (code, name, enabled, sort_order)
SELECT 'kimi', 'Kimi', TRUE, 30
WHERE NOT EXISTS (
    SELECT 1 FROM custom_platforms WHERE code = 'kimi' AND deleted_at IS NULL
);
