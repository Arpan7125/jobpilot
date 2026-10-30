# Posting parser eval

Put real job postings here, one per file, as `NAME.txt` (copy the full posting
text). For each one, write `NAME.expected.json` by hand from the posting, in the
same shape as the parser output (see `internal/agent/parser.go`, type `Posting`).
Only fill in what the posting actually says.

Run from `apps/api`:

    ANTHROPIC_API_KEY=... go run ./cmd/parse-eval

The parser output lands in `NAME.out.json`. The run prints which fields
mismatched per posting, plus field accuracy, required-skill recall and
precision, token totals and mean latency. Don't quote these numbers anywhere
until they come from a real run over real postings.
