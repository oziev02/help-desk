ALTER TABLE tickets
    ADD COLUMN rating INT CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),
    ADD COLUMN refuse_reason TEXT,
    ADD COLUMN reopen_count INT NOT NULL DEFAULT 0;

INSERT INTO roles (name) VALUES ('manager')
ON CONFLICT (name) DO NOTHING;
