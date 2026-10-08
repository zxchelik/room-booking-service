-- +goose Up

CREATE EXTENSION IF NOT EXISTS btree_gist;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('admin', 'user');
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'booking_status') THEN
        CREATE TYPE booking_status AS ENUM ('active', 'cancelled');
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NULL,
    capacity INTEGER NULL CHECK (capacity IS NULL OR capacity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL UNIQUE REFERENCES rooms(id) ON DELETE CASCADE,
    days_of_week SMALLINT[] NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedules_has_days CHECK (COALESCE(array_length(days_of_week, 1), 0) > 0),
    CONSTRAINT schedules_time_bounds CHECK (start_time < end_time)
);

CREATE TABLE IF NOT EXISTS slots (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT slots_time_bounds CHECK (start_time < end_time)
);

CREATE UNIQUE INDEX IF NOT EXISTS slots_room_start_end_unique_idx
    ON slots(room_id, start_time, end_time);

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'slots_no_overlap'
    ) THEN
        ALTER TABLE slots
            ADD CONSTRAINT slots_no_overlap
            EXCLUDE USING gist (
                room_id WITH =,
                tstzrange(start_time, end_time, '[)') WITH &&
        );
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY,
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status booking_status NOT NULL,
    conference_link TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS bookings_one_active_slot_idx
    ON bookings(slot_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS bookings_user_created_idx
    ON bookings(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS bookings_status_created_idx
    ON bookings(status, created_at DESC);

CREATE INDEX IF NOT EXISTS slots_room_start_idx
    ON slots(room_id, start_time);

CREATE INDEX IF NOT EXISTS slots_start_idx
    ON slots(start_time);

-- +goose Down

DROP INDEX IF EXISTS slots_start_idx;
DROP INDEX IF EXISTS slots_room_start_idx;
DROP INDEX IF EXISTS bookings_status_created_idx;
DROP INDEX IF EXISTS bookings_user_created_idx;
DROP INDEX IF EXISTS bookings_one_active_slot_idx;
DROP TABLE IF EXISTS bookings;
ALTER TABLE slots DROP CONSTRAINT IF EXISTS slots_no_overlap;
DROP INDEX IF EXISTS slots_room_start_end_unique_idx;
DROP TABLE IF EXISTS slots;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS booking_status;
DROP TYPE IF EXISTS user_role;
