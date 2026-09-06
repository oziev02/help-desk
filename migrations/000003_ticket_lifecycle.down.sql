ALTER TABLE tickets
    DROP COLUMN IF EXISTS rating,
    DROP COLUMN IF EXISTS refuse_reason,
    DROP COLUMN IF EXISTS reopen_count;

DELETE FROM roles WHERE name = 'manager';
