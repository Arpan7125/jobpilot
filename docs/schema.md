# Database schema

Source of truth: [`apps/api/prisma/schema.prisma`](../apps/api/prisma/schema.prisma).
First migration: `apps/api/prisma/migrations/*_init/migration.sql`.

```
users 1──* jobs 1──1 applications 1──* application_status_history
```

| Table | What a row is | Key columns |
|---|---|---|
| `users` | An account | `email` (unique, lowercased), `password_hash` |
| `jobs` | A posting you pasted | `user_id`, `company`, `title`, `platform`, `description` (raw text), `salary_min/max`, `parsed` (JSONB for the Week 2 parser) |
| `applications` | Your pursuit of one job | `job_id` (unique), `status`, `applied_at`, `status_changed_at`, `notes` |
| `application_status_history` | One status change | `application_id`, `from_status` (null on first row), `to_status`, `changed_at` |

`status` is a Postgres enum: `SAVED, APPLIED, REPLIED, INTERVIEW, OFFER, REJECTED, WITHDRAWN`.

## Design choices

- **Job and application are separate.** The posting is facts about the role; the
  application is your progress. The parser output and the Week 3 skills table hang
  off `jobs`; the funnel hangs off `applications`. One application per job
  (`job_id` is unique), created as `SAVED` when you add the job.
- **Ownership lives on `jobs.user_id`.** Applications and history reach the user
  through the job, so there is no second `user_id` that could disagree.
- **History is append-only.** Every status change writes the new status on
  `applications` and a history row in the same transaction (Day 4). The funnel
  and time-between-stages views (Day 16) read only from history.
- **`status_changed_at` drives reminders.** Day 6's job selects `APPLIED` rows
  older than 7 days, served by the `(status, status_changed_at)` index.
- **UUID keys, `timestamptz` everywhere, cascading deletes** from user down.
