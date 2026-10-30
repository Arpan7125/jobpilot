-- Resume tailoring (Week 2, Day 10). The resume lives on the profile as plain
-- text. A tailoring is one set of suggested rewordings for one job, each
-- waiting for the user's approve or reject. Nothing leaves the app.

-- +goose Up
ALTER TABLE profiles ADD COLUMN resume text NOT NULL DEFAULT '';

CREATE TABLE tailorings (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      uuid NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    resume      text NOT NULL,   -- the resume the suggestions were made against
    suggestions jsonb NOT NULL,  -- [{original, rewritten, reason, status}]
    rejected    jsonb NOT NULL DEFAULT '[]',  -- suggestions the guards dropped, with why
    relevant    jsonb NOT NULL DEFAULT '[]',  -- resume lines most relevant to the job
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tailorings_job_created_idx ON tailorings (job_id, created_at DESC);

-- +goose Down
DROP TABLE tailorings;
ALTER TABLE profiles DROP COLUMN resume;
