# Database schema

Source of truth: [`apps/api/internal/db/migrations`](../apps/api/internal/db/migrations).
The API applies pending migrations on startup (goose, embedded in the binary).

```
users 1──* jobs 1──1 applications 1──* application_status_history
                                 └────* reminders
```

| Table | What a row is | Key columns |
|---|---|---|
| `users` | An account | `email` (unique, lowercased), `password_hash` (bcrypt) |
| `jobs` | A posting you pasted | `user_id`, `company`, `title`, `platform`, `description` (raw text), `salary_min/max`, `currency`, `parsed` (JSONB for the Week 2 parser) |
| `applications` | Your pursuit of one job | `job_id` (unique), `status`, `applied_at`, `status_changed_at`, `notes` |
| `application_status_history` | One status change | `application_id`, `from_status` (null on first row), `to_status`, `changed_at`, `note` |
| `reminders` | A follow-up nudge | `application_id`, `reason`, `created_at`, `dismissed_at` (null while open) |

`status` is a Postgres enum: `saved, applied, replied, interview, offer, rejected, withdrawn`.

## Design choices

- **Job and application are separate.** The posting is facts about the role; the
  application is your progress. The parser output and the Week 3 skills table hang
  off `jobs`; the funnel hangs off `applications`. One application per job
  (`job_id` is unique), created as `saved` in the same transaction as the job.
- **Ownership lives on `jobs.user_id`.** Applications and history reach the user
  through the job, so there is no second `user_id` that could disagree. Every
  query filters on it; another user's job is a 404, never a 403.
- **History is append-only.** A status change updates `applications` and inserts
  a history row in one transaction, with the application row locked
  (`FOR UPDATE`). The funnel and time-between-stages views (Day 16) read history.
- **`applied_at` is set once,** the first time an application reaches `applied`.
- **`status_changed_at` drives reminders.** The follow-up job (in the API process,
  at startup and every 24 hours) adds a reminder to each `applied` row older
  than 7 days, served by the `(status, status_changed_at)` index. A partial
  unique index allows one open reminder per application, a dismissed reminder
  does not return until the status changes again, and any status change closes
  the open one. The job is idempotent, so restarts or extra runs are harmless.
- **Database checks back up the API:** non-negative salaries, `salary_min <= salary_max`.
