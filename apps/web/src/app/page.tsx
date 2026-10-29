import Link from "next/link";
import { Suspense } from "react";
import { AppHeader } from "@/components/app-header";
import { LocalTime } from "@/components/local-time";
import { StatusForm } from "@/components/status-form";
import { api } from "@/lib/api";
import { STATUSES, STATUS_LABELS, type Job } from "@/lib/types";

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
  const jobs = await api<Job[]>("/jobs");

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
  );
}
