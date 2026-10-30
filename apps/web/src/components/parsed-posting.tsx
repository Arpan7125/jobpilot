import type { ParsedPosting as Parsed } from "@/lib/types";

const LABELS: Record<string, string> = {
  intern: "Intern",
  entry: "Entry level",
  mid: "Mid level",
  senior: "Senior",
  lead: "Lead",
  full_time: "Full time",
  part_time: "Part time",
  internship: "Internship",
  contract: "Contract",
  onsite: "On-site",
  hybrid: "Hybrid",
  remote: "Remote",
  hour: "per hour",
  month: "per month",
  year: "per year",
  total: "total",
};

function range(min: number | null, max: number | null, fmt = (n: number) => String(n)): string | null {
  if (min != null && max != null) return min === max ? fmt(min) : `${fmt(min)} to ${fmt(max)}`;
  if (min != null) return `${fmt(min)}+`;
  if (max != null) return `up to ${fmt(max)}`;
  return null;
}

function Skills({ items }: { items: string[] }) {
  if (items.length === 0) return <span className="text-gray-500">None stated</span>;
  return (
    <ul className="flex flex-wrap gap-1.5">
      {items.map((s) => (
        <li key={s} className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-800">
          {s}
        </li>
      ))}
    </ul>
  );
}

// Shows only what the parser found; anything not stated reads "Not stated".
export function ParsedPosting({ parsed }: { parsed: Parsed }) {
  const experience = range(parsed.experienceYears.min, parsed.experienceYears.max);
  const pay = range(parsed.pay.min, parsed.pay.max, (n) => n.toLocaleString("en-IN"));
  const rows: [string, React.ReactNode][] = [
    ["Role", parsed.role || null],
    ["Level", LABELS[parsed.seniority]],
    ["Type", LABELS[parsed.employmentType]],
    ["Experience", experience && `${experience} years`],
    [
      "Pay",
      pay &&
        [pay, parsed.pay.currency, LABELS[parsed.pay.period]].filter(Boolean).join(" "),
    ],
    ["Location", [parsed.locations.join(", "), LABELS[parsed.workMode]].filter(Boolean).join(" · ") || null],
    ["Degree", parsed.degree],
    ["Required skills", <Skills key="r" items={parsed.requiredSkills} />],
    ["Nice to have", <Skills key="p" items={parsed.preferredSkills} />],
  ];
  return (
    <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-2 text-sm">
      {rows.map(([label, value]) => (
        <div key={label} className="contents">
          <dt className="font-medium text-gray-600">{label}</dt>
          <dd className="text-gray-900">{value ?? <span className="text-gray-500">Not stated</span>}</dd>
        </div>
      ))}
    </dl>
  );
}
