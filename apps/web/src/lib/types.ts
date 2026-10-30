export const STATUSES = ["saved", "applied", "replied", "interview", "offer", "rejected", "withdrawn"] as const;
export type Status = (typeof STATUSES)[number];

export const STATUS_LABELS: Record<Status, string> = {
  saved: "Saved",
  applied: "Applied",
  replied: "Replied",
  interview: "Interview",
  offer: "Offer",
  rejected: "Rejected",
  withdrawn: "Withdrawn",
};

export type Job = {
  id: string;
  company: string;
  title: string;
  location: string | null;
  url: string | null;
  platform: string | null;
  description: string;
  salaryMin: number | null;
  salaryMax: number | null;
  currency: string | null;
  status: Status;
  appliedAt: string | null;
  statusChangedAt: string;
  notes: string | null;
  followUpDue: boolean;
  parsed: ParsedPosting | null;
  parsedAt: string | null;
  fit: Fit | null;
  createdAt: string;
  updatedAt: string;
};

// What the posting parser extracted. Fields the posting doesn't state are
// null, empty or "unknown".
export type ParsedPosting = {
  role: string;
  company: string | null;
  seniority: "intern" | "entry" | "mid" | "senior" | "lead" | "unknown";
  employmentType: "full_time" | "part_time" | "internship" | "contract" | "unknown";
  requiredSkills: string[];
  preferredSkills: string[];
  experienceYears: { min: number | null; max: number | null };
  pay: { min: number | null; max: number | null; currency: string | null; period: "hour" | "month" | "year" | "total" | "unknown" };
  locations: string[];
  workMode: "onsite" | "hybrid" | "remote" | "unknown";
  degree: string | null;
};

// How a parsed job matches the profile. score is null when there's not
// enough on either side to compare.
export type Fit = {
  score: number | null;
  matchedSkills: string[];
  missingSkills: string[];
  matchedPreferred: string[];
  filters: string[];
  reasons: string[];
};

export const WORK_MODES = ["onsite", "hybrid", "remote"] as const;
export const PAY_PERIODS = ["month", "year", "hour", "total"] as const;

export type Profile = {
  skills: string[];
  experienceYears: number | null;
  locations: string[];
  workModes: (typeof WORK_MODES)[number][];
  minPay: number | null;
  payCurrency: string | null;
  payPeriod: (typeof PAY_PERIODS)[number] | null;
  resume: string;
  updatedAt: string | null;
};

export type Suggestion = {
  original: string;
  rewritten: string;
  reason: string;
  status: "pending" | "approved" | "rejected";
};

// Resume suggestions for one job, waiting for the user's decisions.
export type Tailoring = {
  id: string;
  createdAt: string;
  suggestions: Suggestion[];
  dropped: { original: string; rewritten: string; why: string }[];
  relevant: string[];
  tailored: string;
  stale: boolean;
};

export type StatusChange = {
  fromStatus: Status | null;
  toStatus: Status;
  changedAt: string;
  note: string | null;
};

export type Reminder = {
  id: string;
  jobId: string;
  company: string;
  title: string;
  reason: string;
  createdAt: string;
};

export type User = { id: string; email: string; name: string | null };

// Returned by server actions so forms can show what went wrong. `values`
// echoes the submitted fields, because React resets a form after its action
// runs and the user would otherwise lose what they typed.
export type FormState =
  | { error?: string; fields?: Record<string, string>; values?: Record<string, string>; message?: string }
  | undefined;
