DO $$
DECLARE
    rec RECORD;
    alphabet CONSTANT TEXT := 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
    letters CONSTANT TEXT := 'ABCDEFGHJKLMNPQRSTUVWXYZ';
    generated_code TEXT;
BEGIN
    FOR rec IN
        SELECT id
        FROM users
        WHERE deleted_at IS NULL
          AND (
              invite_code ~ '^[0-9]+$'
              OR invite_code ~ '^[0-9A-F]{16}$'
          )
    LOOP
        LOOP
            generated_code := SUBSTRING(letters FROM (FLOOR(RANDOM() * LENGTH(letters)) + 1)::INT FOR 1);
            FOR i IN 2..8 LOOP
                generated_code := generated_code || SUBSTRING(alphabet FROM (FLOOR(RANDOM() * LENGTH(alphabet)) + 1)::INT FOR 1);
            END LOOP;

            EXIT WHEN NOT EXISTS (
                SELECT 1
                FROM users
                WHERE deleted_at IS NULL
                  AND invite_code = generated_code
                  AND id <> rec.id
            );
        END LOOP;

        UPDATE users
        SET invite_code = generated_code
        WHERE id = rec.id;
    END LOOP;
END $$;
