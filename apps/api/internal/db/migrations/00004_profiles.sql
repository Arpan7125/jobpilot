-- What the user brings, for the fit score (Week 2, Day 9). One row per user,
-- created on first save. Skills are only what the user enters: the app never
-- adds a skill on their behalf.

-- +goose Up
CREATE TABLE profiles (
    user_id          uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    skills           text[] NOT NULL DEFAULT '{}',
    experience_years integer CHECK (experience_years >= 0),
    locations        text[] NOT NULL DEFAULT '{}',   -- cities the user will work in
    work_modes       text[] NOT NULL DEFAULT '{}',   -- onsite, hybrid, remote; empty = any
    min_pay          integer CHECK (min_pay >= 0),
    pay_currency     text,
    pay_period       text CHECK (pay_period IN ('hour', 'month', 'year', 'total')),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE profiles;
