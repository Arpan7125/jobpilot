"use client";

import { useActionState } from "react";
import { parseJob } from "@/lib/actions";
import { Button } from "./ui";

export function ParseForm({ jobId, parsed }: { jobId: string; parsed: boolean }) {
  const [state, action, pending] = useActionState(parseJob.bind(null, jobId), undefined);
  return (
    <form action={action} className="flex items-center gap-3">
      <Button type="submit" disabled={pending}>
        {pending ? "Parsing..." : parsed ? "Parse again" : "Parse posting"}
      </Button>
      {state?.error && <p className="text-sm text-red-600">{state.error}</p>}
    </form>
  );
}
