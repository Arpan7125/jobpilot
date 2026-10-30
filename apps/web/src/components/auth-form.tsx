"use client";

import Link from "next/link";
import { useActionState } from "react";
import { login, signup } from "@/lib/actions";
import { Button, Field, FormError } from "./ui";

export function AuthForm({ mode }: { mode: "login" | "signup" }) {
  const [state, action, pending] = useActionState(mode === "login" ? login : signup, undefined);
  const f = state?.fields ?? {};
  const v = state?.values ?? {};

  return (
    <form action={action} className="space-y-4">
      <FormError message={state?.error} />
      {mode === "signup" && <Field label="Name (optional)" name="name" autoComplete="name" defaultValue={v.name} error={f.name} />}
      <Field label="Email" name="email" type="email" required autoComplete="email" defaultValue={v.email} error={f.email} />
      <Field
        label="Password"
        name="password"
        type="password"
        required
        minLength={mode === "signup" ? 8 : undefined}
        autoComplete={mode === "login" ? "current-password" : "new-password"}
        error={f.password}
      />
      <Button type="submit" disabled={pending} className="w-full">
        {mode === "login" ? "Log in" : "Create account"}
      </Button>
      <p className="text-center text-sm text-gray-600">
        {mode === "login" ? (
          <>
            No account? <Link href="/signup" className="text-blue-600 hover:underline">Sign up</Link>
          </>
        ) : (
          <>
            Have an account? <Link href="/login" className="text-blue-600 hover:underline">Log in</Link>
          </>
        )}
      </p>
    </form>
  );
}
