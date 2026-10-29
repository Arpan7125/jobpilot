"use client";

import { useActionState } from "react";
import { setStatus } from "@/lib/actions";
import { STATUSES, STATUS_LABELS, type Status } from "@/lib/types";
import { Button, inputClass } from "./ui";

// Full form on the job page; `compact` is the one-click select on board cards.
export function StatusForm({ jobId, status, compact = false }: { jobId: string; status: Status; compact?: boolean }) {
  const [state, action, pending] = useActionState(setStatus.bind(null, jobId), undefined);

  if (compact) {
    return (
      <form action={action}>
        <select
          name="status"
          defaultValue={status}
          disabled={pending}
          aria-label="Change status"
          onChange={(e) => e.currentTarget.form?.requestSubmit()}
          className="w-full rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-900"
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {STATUS_LABELS[s]}
            </option>
          ))}
        </select>
      </form>
    );
  }

  return (
    <form action={action} className="flex flex-wrap items-end gap-2">
      <label className="text-sm">
        <span className="mb-1 block font-medium text-gray-700">Status</span>
        <select name="status" defaultValue={status} className={inputClass}>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {STATUS_LABELS[s]}
            </option>
          ))}
        </select>
      </label>
      <label className="min-w-48 flex-1 text-sm">
        <span className="mb-1 block font-medium text-gray-700">Note (optional)</span>
        <input name="note" placeholder="e.g. Recruiter replied on email" className={inputClass} />
      </label>
      <Button type="submit" disabled={pending}>
        Update
      </Button>
      {state?.error && <p className="w-full text-sm text-red-600">{state.error}</p>}
    </form>
  );
}
