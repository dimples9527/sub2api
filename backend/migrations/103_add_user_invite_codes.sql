ALTER TABLE users
    ADD COLUMN IF NOT EXISTS invite_code VARCHAR(16);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS invited_by_id BIGINT;

UPDATE users
SET invite_code = UPPER(LPAD(TO_HEX(id), 16, '0'))
WHERE invite_code IS NULL OR BTRIM(invite_code) = '';

ALTER TABLE users
    ALTER COLUMN invite_code SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_invited_by_id_fkey'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_invited_by_id_fkey
            FOREIGN KEY (invited_by_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS users_invite_code_unique_active
    ON users(invite_code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_invited_by_id
    ON users(invited_by_id)
    WHERE deleted_at IS NULL AND invited_by_id IS NOT NULL;
