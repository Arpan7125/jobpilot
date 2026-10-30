"use client";

import { useActionState } from "react";
import { saveProfile } from "@/lib/actions";
import { PAY_PERIODS, WORK_MODES, type Profile } from "@/lib/types";
import { Button, Field, FormError, inputClass } from "./ui";

const MODE_LABELS = { onsite: "On-site", hybrid: "Hybrid", remote: "Remote" };

export function ProfileForm({ profile }: { profile: Profile }) {
  const [state, action, pending] = useActionState(saveProfile, undefined);
  const f = state?.fields ?? {};
  const v = state?.values;

  return (
    <form action={action} className="space-y-4">
      <FormError message={state?.error} />
      {state?.message && <p className="rounded-md bg-green-50 px-3 py-2 text-sm text-green-800">{state.message}</p>}
      <label className="block text-sm">
        <span className="mb-1 block font-medium text-gray-700">Skills</span>
        <textarea
          name="skills"
          rows={3}
          placeholder="Go, PostgreSQL, Docker, React"
          defaultValue={v?.skills ?? profile.skills.join(", ")}
          className={inputClass}
        />
        <span className="mt-1 block text-xs text-gray-500">Separate with commas. Common spellings match, e.g. Golang and Go.</span>
        {f.skills && <span className="mt-1 block text-xs text-red-600">{f.skills}</span>}
      </label>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          label="Years of experience"
          name="experienceYears"
          type="number"
          min={0}
          defaultValue={v?.experienceYears ?? profile.experienceYears ?? ""}
          error={f.experienceYears}
        />
        <Field
          label="Locations"
          name="locations"
          placeholder="Bengaluru, Hyderabad"
          defaultValue={v?.locations ?? profile.locations.join(", ")}
          error={f.locations}
        />
      </div>
      <fieldset className="text-sm">
        <legend className="mb-1 font-medium text-gray-700">Work modes you accept</legend>
        <div className="flex gap-4">
          {WORK_MODES.map((m) => (
            <label key={m} className="flex items-center gap-1.5 text-gray-800">
              <input type="checkbox" name="workModes" value={m} defaultChecked={profile.workModes.includes(m)} />
              {MODE_LABELS[m]}
            </label>
          ))}
        </div>
        <span className="mt-1 block text-xs text-gray-500">Leave all unticked to accept any. Jobs in other modes are flagged.</span>
        {f.workModes && <span className="mt-1 block text-xs text-red-600">{f.workModes}</span>}
      </fieldset>
      <div className="grid gap-4 sm:grid-cols-3">
        <Field
          label="Minimum pay"
          name="minPay"
          type="number"
          min={0}
          defaultValue={v?.minPay ?? profile.minPay ?? ""}
          error={f.minPay}
        />
        <Field
          label="Currency"
          name="payCurrency"
          maxLength={3}
          placeholder="INR"
          defaultValue={v?.payCurrency ?? profile.payCurrency ?? ""}
          error={f.payCurrency}
        />
        <label className="block text-sm">
          <span className="mb-1 block font-medium text-gray-700">Per</span>
          <select name="payPeriod" defaultValue={v?.payPeriod ?? profile.payPeriod ?? ""} className={inputClass}>
            <option value="">Not set</option>
            {PAY_PERIODS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
          {f.payPeriod && <span className="mt-1 block text-xs text-red-600">{f.payPeriod}</span>}
        </label>
      </div>
      <label className="block text-sm">
        <span className="mb-1 block font-medium text-gray-700">Resume</span>
        <textarea
          name="resume"
          rows={14}
          placeholder="Paste your resume as plain text, one bullet per line"
          defaultValue={v?.resume ?? profile.resume}
          className={`${inputClass} font-mono`}
        />
        <span className="mt-1 block text-xs text-gray-500">
          Used for tailoring. Suggestions only reword lines that are already here.
        </span>
        {f.resume && <span className="mt-1 block text-xs text-red-600">{f.resume}</span>}
      </label>
      <Button type="submit" disabled={pending}>
        Save profile
      </Button>
    </form>
  );
}
