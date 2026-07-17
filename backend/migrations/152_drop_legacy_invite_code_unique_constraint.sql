-- Drop uniqueness on legacy users.invite_code if it exists.
-- With DEFAULT '', multiple users would violate uniqueness on registration.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND constraint_name = 'users_invite_code_key'
          AND constraint_type = 'UNIQUE'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_invite_code_key;
    END IF;
END $$;

DROP INDEX IF EXISTS users_invite_code_unique_active;
