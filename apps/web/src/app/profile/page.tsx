import { Suspense } from "react";
import { AppHeader } from "@/components/app-header";
import { ProfileForm } from "@/components/profile-form";
import { api } from "@/lib/api";
import type { Profile } from "@/lib/types";

export default function ProfilePage() {
  return (
    <>
      <AppHeader />
      <main className="mx-auto w-full max-w-2xl flex-1 px-4 py-6">
        <h1 className="text-2xl font-semibold text-gray-900">Your profile</h1>
        <p className="mb-6 text-sm text-gray-600">
          Fit scores compare each parsed posting with this. List only skills you actually have: JobPilot never adds
          skills for you.
        </p>
        <Suspense fallback={<p className="text-sm text-gray-500">Loading...</p>}>
          <LoadedForm />
        </Suspense>
      </main>
    </>
  );
}

async function LoadedForm() {
  const profile = await api<Profile>("/profile");
  return <ProfileForm profile={profile} />;
}
