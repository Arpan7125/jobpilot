-- Follow-up reminders (Week 1, Day 6). A daily job adds one when an
-- application has sat in 'applied' for 7 days with no status change.

-- +goose Up
CREATE TABLE reminders (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id uuid NOT NULL REFERENCES applications (id) ON DELETE CASCADE,
    reason         text NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    dismissed_at   timestamptz
);
-- At most one open reminder per application.
CREATE UNIQUE INDEX reminders_one_open_idx ON reminders (application_id) WHERE dismissed_at IS NULL;

-- +goose Down
DROP TABLE reminders;
