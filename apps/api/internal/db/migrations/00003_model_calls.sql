-- +goose Up
-- One row per LLM request: what it cost, how long it took, whether it worked.
CREATE TABLE model_calls (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    job_id        uuid REFERENCES jobs (id) ON DELETE SET NULL,
    feature       text NOT NULL,          -- e.g. 'parse'
    model         text NOT NULL,
    input_tokens  bigint NOT NULL DEFAULT 0,
    output_tokens bigint NOT NULL DEFAULT 0,
    latency_ms    integer NOT NULL,
    ok            boolean NOT NULL,
    error         text,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX model_calls_user_created_idx ON model_calls (user_id, created_at DESC);

ALTER TABLE jobs ADD COLUMN parsed_at timestamptz;

-- +goose Down
ALTER TABLE jobs DROP COLUMN parsed_at;
DROP TABLE model_calls;
