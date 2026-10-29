# JobPilot

Job application tracker with an AI agent that scores job fit and drafts tailored
resumes for your approval. Built over four weeks; this is Week 1 (full-stack foundation).

## Stack

- `apps/web`: Next.js 16 (App Router, TypeScript, Tailwind)
- `apps/api`: Node.js, Express 5, Prisma 7, TypeScript, Vitest
- PostgreSQL 16

Schema design: [docs/schema.md](docs/schema.md).

## Run locally

Needs Node 22+ and Docker (or any Postgres 16 on `localhost:5432`).

```bash
npm install
npm run db:up                              # Postgres in Docker
cp apps/api/.env.example apps/api/.env
npm run db:deploy -w apps/api              # apply migrations
npm run dev:api                            # http://localhost:4000/health
npm run dev:web                            # http://localhost:3000
npm test -w apps/api
```

Change the schema with `npm run db:migrate -w apps/api -- --name <change>`.
