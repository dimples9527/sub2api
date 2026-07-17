-- Keep the legacy users.invite_code column when it exists, but make it non-blocking.
-- Current invitation registration state is stored through redeem_codes usage.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'users'
          AND column_name = 'invite_code'
    ) THEN
        ALTER TABLE users
            ALTER COLUMN invite_code DROP NOT NULL,
            ALTER COLUMN invite_code SET DEFAULT '';
    END IF;
END $$;
