"use client";

import Link from "next/link";
import { useActionState, useState, useTransition } from "react";
import { decideSuggestion, tailorJob } from "@/lib/actions";
import type { Tailoring } from "@/lib/types";
import { Button } from "./ui";

export function TailorButton({ jobId, again }: { jobId: string; again: boolean }) {
  const [state, action, pending] = useActionState(tailorJob.bind(null, jobId), undefined);
  return (
    <form action={action} className="flex flex-wrap items-center gap-3">
      <Button type="submit" disabled={pending}>
        {pending ? "Tailoring..." : again ? "Suggest again" : "Tailor my resume"}
      </Button>
      {state?.error && (
        <p className="text-sm text-red-600">
          {state.error}
          {state.error.includes("profile") && (
            <>
              {" "}
              <Link href="/profile" className="text-blue-600 hover:underline">
                Open profile
              </Link>
            </>
          )}
        </p>
      )}
    </form>
  );
}

const STATUS_STYLE = {
  pending: "border-gray-200",
  approved: "border-green-300 bg-green-50",
  rejected: "border-gray-200 opacity-60",
};

// The approval screen: nothing changes in the tailored resume until the user approves it.
export function TailoringReview({ jobId, tailoring }: { jobId: string; tailoring: Tailoring }) {
  const [busy, start] = useTransition();
  const [copied, setCopied] = useState(false);
  const decide = (n: number, status: "approved" | "rejected" | "pending") =>
    start(() => decideSuggestion(jobId, n, status));
  const approved = tailoring.suggestions.filter((s) => s.status === "approved").length;

  return (
    <div className="space-y-4 text-sm">
      {tailoring.stale && (
        <p className="rounded-md bg-amber-50 px-3 py-2 text-amber-900">
          Your resume changed after these suggestions were made. Suggest again to use the new version.
        </p>
      )}
      {tailoring.suggestions.length === 0 && (
        <p className="text-gray-600">No rewordings worth making for this job. Your resume already fits it as written.</p>
      )}
      <ol className="space-y-3">
        {tailoring.suggestions.map((s, i) => (
          <li key={i} className={`rounded-md border p-3 ${STATUS_STYLE[s.status]}`}>
            <p className="text-gray-500 line-through decoration-gray-400">{s.original}</p>
            <p className="mt-1 text-gray-900">{s.rewritten}</p>
            <p className="mt-1 text-xs text-gray-500">{s.reason}</p>
            <div className="mt-2 flex gap-3 text-xs">
              {s.status === "pending" ? (
                <>
                  <button disabled={busy} onClick={() => decide(i, "approved")} className="font-medium text-green-700 hover:underline">
                    Approve
                  </button>
                  <button disabled={busy} onClick={() => decide(i, "rejected")} className="text-gray-600 hover:underline">
                    Reject
                  </button>
                </>
              ) : (
                <>
                  <span className="font-medium text-gray-700">{s.status === "approved" ? "Approved" : "Rejected"}</span>
                  <button disabled={busy} onClick={() => decide(i, "pending")} className="text-blue-600 hover:underline">
                    Undo
                  </button>
                </>
              )}
            </div>
          </li>
        ))}
      </ol>

      {tailoring.relevant.length > 0 && (
        <div>
          <h3 className="font-medium text-gray-900">Lead with these lines</h3>
          <ul className="mt-1 list-disc pl-5 text-gray-700">
            {tailoring.relevant.map((l) => (
              <li key={l}>{l}</li>
            ))}
          </ul>
        </div>
      )}

      {tailoring.dropped.length > 0 && (
        <details className="text-gray-600">
          <summary className="cursor-pointer">
            {tailoring.dropped.length} suggestion{tailoring.dropped.length > 1 ? "s" : ""} blocked for adding something you
            didn&apos;t write
          </summary>
          <ul className="mt-2 space-y-2">
            {tailoring.dropped.map((d, i) => (
              <li key={i}>
                <span className="text-gray-800">{d.rewritten}</span>
                <span className="block text-xs text-red-700">{d.why}</span>
              </li>
            ))}
          </ul>
        </details>
      )}

      <div>
        <div className="mb-1 flex items-center justify-between">
          <h3 className="font-medium text-gray-900">
            Tailored resume <span className="font-normal text-gray-500">({approved} change{approved === 1 ? "" : "s"} applied)</span>
          </h3>
          <button
            onClick={() => navigator.clipboard.writeText(tailoring.tailored).then(() => setCopied(true))}
            className="text-xs text-blue-600 hover:underline"
          >
            {copied ? "Copied" : "Copy"}
          </button>
        </div>
        <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md bg-gray-50 p-3 font-mono text-xs text-gray-800">
          {tailoring.tailored}
        </pre>
        <p className="mt-1 text-xs text-gray-500">JobPilot never sends this anywhere. Copy it when you&apos;re happy with it.</p>
      </div>
    </div>
  );
}
