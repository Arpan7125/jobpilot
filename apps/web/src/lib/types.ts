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
  createdAt: string;
  updatedAt: string;
};

export type StatusChange = {
  fromStatus: Status | null;
  toStatus: Status;
  changedAt: string;
  note: string | null;
};

export type User = { id: string; email: string; name: string | null };

// Returned by server actions so forms can show what went wrong. `values`
// echoes the submitted fields, because React resets a form after its action
// runs and the user would otherwise lose what they typed.
export type FormState =
  | { error?: string; fields?: Record<string, string>; values?: Record<string, string> }
  | undefined;
