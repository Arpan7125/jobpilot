"use client";

// Formats a timestamp in the viewer's time zone. The server renders in UTC,
// so the first paint can differ from the hydrated text.
export function LocalTime({ iso, dateOnly = false }: { iso: string; dateOnly?: boolean }) {
  const d = new Date(iso);
  return (
    <time dateTime={iso} suppressHydrationWarning>
      {dateOnly ? d.toLocaleDateString("en-IN") : d.toLocaleString("en-IN")}
    </time>
  );
}
