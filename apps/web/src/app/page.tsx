import { Suspense } from "react";

// Placeholder home page. The job list, add form and status board arrive on Day 5.

const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:4000";

async function ApiStatus() {
  let status = "API unreachable";
  try {
    const res = await fetch(`${apiUrl}/health`, { cache: "no-store" });
    const body = (await res.json()) as { status: string; db: string };
    status = `API ${body.status}, database ${body.db}`;
  } catch {}
  return <span className="font-mono">{status}</span>;
}

export default function Home() {
  return (
    <main className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-semibold">JobPilot</h1>
      <p className="mt-2 text-gray-600">Track applications, score job fit, tailor your resume.</p>
      <p className="mt-6 text-sm">
        Status:{" "}
        <Suspense fallback={<span className="font-mono">checking...</span>}>
          <ApiStatus />
        </Suspense>
      </p>
    </main>
  );
}
