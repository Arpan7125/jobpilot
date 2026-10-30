import Link from "next/link";
import { Suspense } from "react";
import { AppHeader } from "@/components/app-header";
import { FitBadge } from "@/components/fit-panel";
import { LocalTime } from "@/components/local-time";
import { StatusForm } from "@/components/status-form";
import { api } from "@/lib/api";
import { dismissReminder } from "@/lib/actions";
import { STATUSES, STATUS_LABELS, type Job, type Reminder } from "@/lib/types";

export default function BoardPage() {
  return (
    <>
      <AppHeader />
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6">
        <Suspense fallback={<p className="text-sm text-gray-500">Loading your applications...</p>}>
          <Board />
        </Suspense>
      </main>
    </>
  );
}

async function Board() {
  const [jobs, reminders] = await Promise.all([api<Job[]>("/jobs"), api<Reminder[]>("/reminders")]);

  if (jobs.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-gray-300 p-10 text-center">
        <p className="text-gray-700">No applications yet.</p>
        <Link href="/jobs/new" className="mt-2 inline-block text-blue-600 hover:underline">
          Paste your first job posting
        </Link>
      </div>
    );
  }

  return (
    <>
      <FollowUps reminders={reminders} />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-7">
        {STATUSES.map((status) => {
          const column = jobs.filter((j) => j.status === status);
          return (
            <section key={status} className="rounded-lg bg-gray-100 p-3">
              <h2 className="mb-3 flex justify-between text-sm font-semibold text-gray-700">
                {STATUS_LABELS[status]}
                <span className="text-gray-500">{column.length}</span>
              </h2>
              <ul className="space-y-2">
                {column.map((job) => (
                  <li key={job.id} className="rounded-md bg-white p-3 shadow-sm">
                    <Link href={`/jobs/${job.id}`} className="block hover:underline">
                      <p className="text-sm font-medium text-gray-900">{job.title}</p>
                      <p className="text-xs text-gray-600">{job.company}</p>
                    </Link>
                    <FitBadge fit={job.fit} />
                    {job.followUpDue && (
                      <span className="mt-1 inline-block rounded bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-800">
                        Follow up
                      </span>
                    )}
                    <p className="mt-1 text-xs text-gray-500">
                      since <LocalTime iso={job.statusChangedAt} dateOnly />
                    </p>
                    <div className="mt-2">
                      <StatusForm jobId={job.id} status={job.status} compact />
                    </div>
                  </li>
                ))}
              </ul>
            </section>
          );
        })}
      </div>
    </>
  );
}

function FollowUps({ reminders }: { reminders: Reminder[] }) {
  if (reminders.length === 0) return null;
  return (
    <section className="mb-6 rounded-lg border border-amber-300 bg-amber-50 p-4">
      <h2 className="text-sm font-semibold text-amber-900">
        {reminders.length === 1 ? "1 application needs" : `${reminders.length} applications need`} a follow-up
      </h2>
      <ul className="mt-2 space-y-1">
        {reminders.map((r) => (
          <li key={r.id} className="flex items-center justify-between gap-4 text-sm text-amber-900">
            <span>
              <Link href={`/jobs/${r.jobId}`} className="font-medium hover:underline">
                {r.title} at {r.company}
              </Link>
              : {r.reason}
            </span>
            <form action={dismissReminder.bind(null, r.id)}>
              <button type="submit" className="shrink-0 text-xs text-amber-800 hover:underline">
                Dismiss
              </button>
            </form>
          </li>
        ))}
      </ul>
    </section>
  );
}
