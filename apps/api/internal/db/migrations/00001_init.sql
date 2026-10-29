-- JobPilot schema (Week 1, Day 1).
--
-- users 1─* jobs 1─1 applications 1─* application_status_history
--
-- A job is the posting you pasted. An application is your pursuit of it and
-- holds the current status. Every status change also writes a history row,
-- which is what the Week 3 funnel and time-between-stages views read from.

-- +goose Up
CREATE TYPE application_status AS ENUM (
    'saved',      -- posting saved, not applied yet
    'applied',
    'replied',    -- any response from the company
    'interview',
    'offer',
    'rejected',
    'withdrawn'   -- you pulled out
);

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE,  -- stored lowercased by the API
    password_hash text NOT NULL,
    name          text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE jobs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    company     text NOT NULL,
    title       text NOT NULL,
    location    text,
    url         text,
    platform    text,            -- LinkedIn, Internshala, referral... (Week 3 filter)
    description text NOT NULL,   -- the raw posting text
    salary_min  integer CHECK (salary_min >= 0),
    salary_max  integer CHECK (salary_max >= 0),
    currency    char(3),
    parsed      jsonb,           -- Week 2 parser output
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (salary_min IS NULL OR salary_max IS NULL OR salary_min <= salary_max)
);
CREATE INDEX jobs_user_id_created_at_idx ON jobs (user_id, created_at DESC);

CREATE TABLE applications (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id            uuid NOT NULL UNIQUE REFERENCES jobs (id) ON DELETE CASCADE,
    status            application_status NOT NULL DEFAULT 'saved',
    applied_at        timestamptz,
    -- Last status change; the Day 6 reminder job flags 'applied' rows older than 7 days.
    status_changed_at timestamptz NOT NULL DEFAULT now(),
    notes             text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX applications_status_changed_idx ON applications (status, status_changed_at);

CREATE TABLE application_status_history (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id uuid NOT NULL REFERENCES applications (id) ON DELETE CASCADE,
    from_status    application_status,  -- null on the first row
    to_status      application_status NOT NULL,
    changed_at     timestamptz NOT NULL DEFAULT now(),
    note           text
);
CREATE INDEX application_status_history_app_idx ON application_status_history (application_id, changed_at);

-- +goose Down
DROP TABLE application_status_history;
DROP TABLE applications;
DROP TABLE jobs;
DROP TABLE users;
DROP TYPE application_status;
