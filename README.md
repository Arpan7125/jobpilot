# JobPilot

Job application tracker with an AI agent that scores job fit and drafts tailored
resumes for your approval. Built over four weeks; this is Week 1 (full-stack foundation).

**Live:** https://jobpilot-web-zj21.onrender.com (API health:
https://jobpilot-api-m9bp.onrender.com/health). Hosted on Render's free plan, so
the first request after a quiet spell takes up to a minute while it wakes.

## Stack

- `apps/api`: Go 1.26, chi, pgx, goose migrations, JWT (HS256) + bcrypt
- `apps/web`: Next.js 16 (App Router, TypeScript, Tailwind)
- PostgreSQL 16

Schema design: [docs/schema.md](docs/schema.md).

![Status board](docs/screenshots/board.png)

## How auth works

The Go API issues a 24-hour HS256 JWT on sign-up and login. The Next.js app
never hands that token to browser JavaScript: server actions call the API and
store the token in an `httpOnly`, `SameSite=Lax` cookie (`Secure` in
production), then forward it as a bearer token on every server-side API call.
An expired or missing token sends you back to `/login`.

## Run locally

Needs Go 1.26+, Node 22+ and Docker (or any Postgres 16 on `localhost:5432`).

```bash
npm run db:up                         # Postgres (creates jobpilot and jobpilot_test)
cd apps/api
cp .env.example .env                  # then set JWT_SECRET: openssl rand -hex 32
make run                              # http://localhost:4000, migrates on start
make test                             # unit + API tests against jobpilot_test

cd ../.. && npm install && npm run dev:web   # http://localhost:3000
```

The web app reads `API_URL` (default `http://localhost:4000`). With both apps
running, `node apps/web/e2e/smoke.cjs` drives the main flows in a real browser
(sign up, add, move on the board, edit, delete, log out, log in).

## Deployment

Render, Singapore region, free plan: a Go web service (`apps/api`), a Node web
service (`apps/web`) and Postgres 16. Both services deploy on every push.
Secrets (`DATABASE_URL`, `JWT_SECRET`, `ANTHROPIC_API_KEY`) live in Render's environment settings,
never in the repo. The API runs its migrations on startup. `render.yaml`
describes the same setup as a Blueprint.

## AI features (Week 2)

The AI features call Claude (`claude-opus-5-5`) through the official Go SDK.
They are optional: without `ANTHROPIC_API_KEY` the rest of the app works and
the AI endpoints return `503`.

- **Posting parser.** "Parse posting" on a job page sends the posting text to
  Claude with a JSON schema (structured outputs), so the reply always has the
  same shape: role, company, seniority, employment type, required and
  nice-to-have skills, years of experience, pay, locations, work mode and
  degree. The prompt says to report only what the posting states; anything
  missing comes back as null, empty or "unknown". Server-side fallbacks are on,
  so if the model declines, the API retries on a fallback model in the same call.
- **Fit score and hard filters.** Plain code, not a model call, so the same
  inputs always give the same score. The profile page holds your skills,
  years of experience, locations, accepted work modes and minimum pay. Each
  parsed job gets a 0 to 100 score: required skills 60, nice-to-have skills 10,
  experience 20, location 10. A part only counts when both the posting and the
  profile state it. Common spellings match (Golang and Go, Postgres and
  PostgreSQL). Hard filters flag a job whose stated pay is below your minimum
  (same currency and period only, never converted), whose work mode you don't
  accept, or that asks for more than 2 years beyond your experience. Unknown
  values never trigger a filter. The score is computed on read, so it follows
  profile edits.
- **Resume tailoring with approval.** Paste your resume on the profile page.
  "Tailor my resume" on a job asks Claude for rewordings of your existing
  lines for that posting, plus the lines to lead with. The model may only
  reorder or rephrase. Code then checks every suggestion and blocks any that
  quotes a line not in your resume, adds a number the original line doesn't
  state, or names a skill your resume doesn't mention (checked against a watch
  list of common tech skills, the alias table, and the posting's parsed
  skills). Blocked suggestions are shown with the reason. Each surviving
  suggestion waits for Approve or Reject; the tailored resume only changes for
  approved ones, and you copy it yourself. Nothing is sent anywhere.
- **Model-call log.** Every call is a row in `model_calls`: feature, model that
  answered, input and output tokens, latency, and whether it succeeded.
- **Eval.** `go run ./cmd/parse-eval` (in `apps/api`) runs the parser over
  real postings in `apps/api/evals/postings` and scores it against hand-checked
  answers. See the README there. No accuracy numbers are published until that
  run happens on real postings.

## API

All bodies are JSON. Everything except `/health` and `/auth/*` needs
`Authorization: Bearer <token>`. Validation errors are `422` with
`{"error": "validation failed", "fields": {"title": "is required"}}`.

| Method | Path | What it does |
|---|---|---|
| GET | `/health` | API and database status |
| POST | `/auth/signup` | `{email, password, name?}` → `201 {token, user}` |
| POST | `/auth/login` | `{email, password}` → `{token, user}` |
| GET | `/me` | Current user |
| GET | `/jobs?status=applied` | Your jobs, newest first, optional status filter |
| POST | `/jobs` | Add a job; its application starts as `saved` |
| GET | `/jobs/{id}` | One job with its application |
| PUT | `/jobs/{id}` | Replace the job's fields and notes |
| DELETE | `/jobs/{id}` | Delete the job, its application and history |
| PUT | `/jobs/{id}/status` | `{status, note?}`; `409` if unchanged |
| GET | `/jobs/{id}/history` | Status changes, oldest first |
| GET | `/reminders` | Open follow-up reminders (applied 7+ days, no change) |
| POST | `/reminders/{id}/dismiss` | Dismiss a reminder |
| GET | `/profile` | Your profile (empty until first saved) |
| PUT | `/profile` | `{skills, experienceYears, locations, workModes, minPay, payCurrency, payPeriod}` |
| POST | `/jobs/{id}/tailor` | Suggest rewordings of the profile resume for this job; guarded, all `pending`. `422` without a resume |
| GET | `/jobs/{id}/tailoring` | Latest suggestions, blocked ones with reasons, and the resume with approved ones applied (null if none) |
| PUT | `/jobs/{id}/tailoring/suggestions/{n}` | `{status: approved\|rejected\|pending}` |
| POST | `/jobs/{id}/parse` | Run the posting parser on the description; result in the job's `parsed` field. `503` without an API key, `422` if the model declines, `502` if the call fails |

Job responses include `parsed` (parser output or null) and `fit`
(`{score, matchedSkills, missingSkills, matchedPreferred, filters, reasons}`,
null until parsed).

Job fields: `company`, `title`, `description` (required), `location`, `url`
(http/https), `platform`, `salaryMin`, `salaryMax`, `currency` (3 letters), `notes`.
