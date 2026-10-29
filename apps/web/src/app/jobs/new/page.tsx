import { AppHeader } from "@/components/app-header";
import { JobForm } from "@/components/job-form";
import { createJob } from "@/lib/actions";

export default function NewJobPage() {
  return (
    <>
      <AppHeader />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-6">
        <h1 className="mb-6 text-2xl font-semibold text-gray-900">Add a job</h1>
        <JobForm action={createJob} submitLabel="Save job" />
      </main>
    </>
  );
}
