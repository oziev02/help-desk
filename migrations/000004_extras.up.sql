CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    building TEXT NOT NULL DEFAULT '',
    floor INT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE tickets
    ADD COLUMN room_id UUID REFERENCES rooms(id);

INSERT INTO rooms (name, building, floor) VALUES
    ('A-1-kitchen', 'A', 1),
    ('A-2-classroom', 'A', 2),
    ('B-1-restroom', 'B', 1);
