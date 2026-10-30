import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { AppHeader } from "@/components/app-header";
import { FitPanel } from "@/components/fit-panel";
import { JobForm } from "@/components/job-form";
import { LocalTime } from "@/components/local-time";
import { ParseForm } from "@/components/parse-form";
import { ParsedPosting } from "@/components/parsed-posting";
import { StatusForm } from "@/components/status-form";
import { TailorButton, TailoringReview } from "@/components/tailoring";
import { deleteJob, updateJob } from "@/lib/actions";
import { api, ApiError } from "@/lib/api";
import { STATUS_LABELS, type Job, type StatusChange, type Tailoring } from "@/lib/types";

export default function JobPage({ params, searchParams }: PageProps<"/jobs/[id]">) {
  return (
    <>
      <AppHeader />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-6">
        <Suspense fallback={<p className="text-sm text-gray-500">Loading...</p>}>
          <JobDetail params={params} searchParams={searchParams} />
        </Suspense>
      </main>
    </>
  );
}

async function JobDetail({ params, searchParams }: Pick<PageProps<"/jobs/[id]">, "params" | "searchParams">) {
  const { id } = await params;
  const editing = (await searchParams).edit === "1";

  let job: Job, history: StatusChange[], tailoring: Tailoring | null;
  try {
    [job, history, tailoring] = await Promise.all([
      api<Job>(`/jobs/${id}`),
      api<StatusChange[]>(`/jobs/${id}/history`),
      api<Tailoring | null>(`/jobs/${id}/tailoring`),
    ]);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) notFound();
    throw err;
  }

  if (editing) {
    return (
      <>
        <h1 className="mb-6 text-2xl font-semibold text-gray-900">Edit {job.title}</h1>
        <JobForm action={updateJob.bind(null, job.id)} job={job} submitLabel="Save changes" />
      </>
    );
  }

  const pay =
    job.salaryMin != null || job.salaryMax != null
      ? [job.salaryMin, job.salaryMax].filter((n) => n != null).map((n) => n!.toLocaleString("en-IN")).join(" to ") +
        (job.currency ? ` ${job.currency}` : "")
      : null;

  return (
    <article className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">{job.title}</h1>
          <p className="text-gray-700">
            {job.company}
            {job.location && ` · ${job.location}`}
            {job.platform && ` · via ${job.platform}`}
          </p>
          {pay && <p className="text-sm text-gray-600">{pay}</p>}
          {job.url && (
            <a href={job.url} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline">
              Open posting
            </a>
          )}
        </div>
        <div className="flex shrink-0 gap-3 text-sm">
          <Link href={`/jobs/${job.id}?edit=1`} className="text-blue-600 hover:underline">
            Edit
          </Link>
          <form action={deleteJob.bind(null, job.id)}>
            <button type="submit" className="text-red-600 hover:underline">
              Delete
            </button>
          </form>
        </div>
      </div>

      <section className="rounded-lg border border-gray-200 p-4">
        {/* keyed so the select resets to the new status after a change */}
        <StatusForm key={job.status} jobId={job.id} status={job.status} />
      </section>

      <section className="rounded-lg border border-gray-200 p-4">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="font-semibold text-gray-900">Parsed by AI</h2>
            <p className="text-xs text-gray-500">
              {job.parsedAt ? (
                <>
                  Parsed <LocalTime iso={job.parsedAt} />. Check it against the posting below.
                </>
              ) : (
                "Pull the role, skills, experience and pay out of the posting text."
              )}
            </p>
          </div>
          <ParseForm key={job.parsedAt ?? "none"} jobId={job.id} parsed={job.parsed != null} />
        </div>
        {job.parsed && <ParsedPosting parsed={job.parsed} />}
      </section>

      {job.fit && (
        <section className="rounded-lg border border-gray-200 p-4">
          <h2 className="mb-3 font-semibold text-gray-900">Fit with your profile</h2>
          <FitPanel fit={job.fit} />
        </section>
      )}

      <section className="rounded-lg border border-gray-200 p-4">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="font-semibold text-gray-900">Tailor your resume</h2>
            <p className="text-xs text-gray-500">
              Rewordings of your own resume lines for this job. Nothing is applied until you approve it.
            </p>
          </div>
          <TailorButton key={tailoring?.id ?? "none"} jobId={job.id} again={tailoring != null} />
        </div>
        {tailoring && <TailoringReview jobId={job.id} tailoring={tailoring} />}
      </section>

      <section>
        <h2 className="mb-2 font-semibold text-gray-900">History</h2>
        <ol className="space-y-1 text-sm text-gray-700">
          {history.map((h, i) => (
            <li key={i}>
              <span className="text-gray-500">
                <LocalTime iso={h.changedAt} />
              </span>{" "}
              {h.fromStatus ? `${STATUS_LABELS[h.fromStatus]} → ${STATUS_LABELS[h.toStatus]}` : `Added as ${STATUS_LABELS[h.toStatus]}`}
              {h.note && <span className="text-gray-500"> · {h.note}</span>}
            </li>
          ))}
        </ol>
      </section>

      {job.notes && (
        <section>
          <h2 className="mb-2 font-semibold text-gray-900">Notes</h2>
          <p className="whitespace-pre-wrap text-sm text-gray-700">{job.notes}</p>
        </section>
      )}

      <section>
        <h2 className="mb-2 font-semibold text-gray-900">Posting</h2>
        <p className="whitespace-pre-wrap text-sm text-gray-700">{job.description}</p>
      </section>
    </article>
  );
}
