"use client";

import { useActionState } from "react";
import type { FormState, Job } from "@/lib/types";
import { Button, Field, FormError, inputClass } from "./ui";

type Action = (state: FormState, form: FormData) => Promise<FormState>;

export function JobForm({ action, job, submitLabel }: { action: Action; job?: Job; submitLabel: string }) {
  const [state, formAction, pending] = useActionState(action, undefined);
  const f = state?.fields ?? {};
  // After a failed submit show what was typed; otherwise the saved job.
  const v = (key: keyof Job): string => state?.values?.[key] ?? String(job?.[key] ?? "");

  return (
    <form action={formAction} className="space-y-4">
      <FormError message={state?.error} />
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Company" name="company" required defaultValue={v("company")} error={f.company} />
        <Field label="Role title" name="title" required defaultValue={v("title")} error={f.title} />
        <Field label="Location" name="location" defaultValue={v("location")} error={f.location} />
        <Field
          label="Found on"
          name="platform"
          placeholder="LinkedIn, Internshala, referral..."
          defaultValue={v("platform")}
          error={f.platform}
        />
      </div>
      <Field label="Posting URL" name="url" type="url" defaultValue={v("url")} error={f.url} />
      <div className="grid gap-4 sm:grid-cols-3">
        <Field label="Pay from" name="salaryMin" type="number" min={0} defaultValue={v("salaryMin")} error={f.salaryMin} />
        <Field label="Pay to" name="salaryMax" type="number" min={0} defaultValue={v("salaryMax")} error={f.salaryMax} />
        <Field label="Currency" name="currency" maxLength={3} placeholder="INR" defaultValue={v("currency")} error={f.currency} />
      </div>
      <label className="block text-sm">
        <span className="mb-1 block font-medium text-gray-700">Job posting</span>
        <textarea
          name="description"
          required
          rows={10}
          placeholder="Paste the full posting here"
          defaultValue={v("description")}
          className={inputClass}
        />
        {f.description && <span className="mt-1 block text-xs text-red-600">{f.description}</span>}
      </label>
      <label className="block text-sm">
        <span className="mb-1 block font-medium text-gray-700">Notes</span>
        <textarea name="notes" rows={3} defaultValue={v("notes")} className={inputClass} />
        {f.notes && <span className="mt-1 block text-xs text-red-600">{f.notes}</span>}
      </label>
      <Button type="submit" disabled={pending}>
        {submitLabel}
      </Button>
    </form>
  );
}
