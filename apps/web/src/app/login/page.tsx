import { AuthForm } from "@/components/auth-form";

export default function LoginPage() {
  return (
    <main className="mx-auto w-full max-w-sm flex-1 px-4 py-16">
      <h1 className="mb-6 text-2xl font-semibold text-gray-900">Log in to JobPilot</h1>
      <AuthForm mode="login" />
    </main>
  );
}
