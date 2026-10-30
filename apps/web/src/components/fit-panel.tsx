import Link from "next/link";
import type { Fit } from "@/lib/types";

export function scoreClass(score: number) {
  if (score >= 75) return "bg-green-100 text-green-800";
  if (score >= 50) return "bg-amber-100 text-amber-800";
  return "bg-gray-200 text-gray-700";
}

// Small badge for board cards: the score, or a flag when a hard filter fails.
export function FitBadge({ fit }: { fit: Fit | null }) {
  if (!fit) return null;
  return (
    <span className="mt-1 mr-1 inline-flex gap-1">
      {fit.score != null && (
        <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${scoreClass(fit.score)}`}>Fit {fit.score}</span>
      )}
      {fit.filters.length > 0 && (
        <span className="rounded bg-red-100 px-1.5 py-0.5 text-xs font-medium text-red-800" title={fit.filters.join(" ")}>
          Filtered
        </span>
      )}
    </span>
  );
}

export function FitPanel({ fit }: { fit: Fit }) {
  return (
    <div className="space-y-3 text-sm">
      <div className="flex items-center gap-3">
        {fit.score != null ? (
          <span className={`rounded-md px-2.5 py-1 text-lg font-semibold ${scoreClass(fit.score)}`}>{fit.score}</span>
        ) : (
          <span className="text-gray-600">
            No score yet. <Link href="/profile" className="text-blue-600 hover:underline">Fill in your profile</Link> to get one.
          </span>
        )}
        {fit.score != null && <span className="text-gray-600">out of 100</span>}
      </div>
      {fit.filters.length > 0 && (
        <ul className="space-y-1 rounded-md bg-red-50 px-3 py-2 text-red-800">
          {fit.filters.map((f) => (
            <li key={f}>{f}</li>
          ))}
        </ul>
      )}
      {fit.missingSkills.length > 0 && (
        <p className="text-gray-700">
          <span className="font-medium">Missing:</span> {fit.missingSkills.join(", ")}
        </p>
      )}
      <ul className="list-disc space-y-0.5 pl-5 text-gray-700">
        {fit.reasons.map((r) => (
          <li key={r}>{r}</li>
        ))}
      </ul>
    </div>
  );
}
