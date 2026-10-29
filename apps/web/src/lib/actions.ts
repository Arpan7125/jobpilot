"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { api, ApiError, clearToken, setToken } from "./api";
import type { FormState, Job, User } from "./types";

function failure(err: unknown, form: FormData): FormState {
  if (err instanceof ApiError) {
    const values: Record<string, string> = {};
    form.forEach((v, k) => {
      if (k !== "password" && typeof v === "string" && !k.startsWith("$")) values[k] = v;
    });
    return { error: err.message, fields: err.fields, values };
  }
  throw err;
}

function text(form: FormData, key: string): string {
  return String(form.get(key) ?? "").trim();
}

// Empty number inputs become null; anything else goes to the API to validate.
function number(form: FormData, key: string): number | null | string {
  const v = text(form, key);
  if (v === "") return null;
  const n = Number(v);
  return Number.isInteger(n) ? n : v;
}

async function authenticate(path: "/auth/login" | "/auth/signup", form: FormData): Promise<FormState> {
  try {
    const { token } = await api<{ token: string; user: User }>(path, {
      method: "POST",
      body: JSON.stringify({
        email: text(form, "email"),
        password: String(form.get("password") ?? ""),
        ...(path === "/auth/signup" && text(form, "name") ? { name: text(form, "name") } : {}),
      }),
    });
    await setToken(token);
  } catch (err) {
    return failure(err, form);
  }
  redirect("/");
}

export async function login(_: FormState, form: FormData) {
  return authenticate("/auth/login", form);
}

export async function signup(_: FormState, form: FormData) {
  return authenticate("/auth/signup", form);
}

export async function logout() {
  await clearToken();
  redirect("/login");
}

function jobBody(form: FormData) {
  return JSON.stringify({
    company: text(form, "company"),
    title: text(form, "title"),
    description: text(form, "description"),
    location: text(form, "location") || null,
    url: text(form, "url") || null,
    platform: text(form, "platform") || null,
    salaryMin: number(form, "salaryMin"),
    salaryMax: number(form, "salaryMax"),
    currency: text(form, "currency") || null,
    notes: text(form, "notes") || null,
  });
}

export async function createJob(_: FormState, form: FormData): Promise<FormState> {
  try {
    await api<Job>("/jobs", { method: "POST", body: jobBody(form) });
  } catch (err) {
    return failure(err, form);
  }
  revalidatePath("/");
  redirect("/");
}

export async function updateJob(id: string, _: FormState, form: FormData): Promise<FormState> {
  try {
    await api<Job>(`/jobs/${id}`, { method: "PUT", body: jobBody(form) });
  } catch (err) {
    return failure(err, form);
  }
  revalidatePath("/");
  redirect(`/jobs/${id}`);
}

export async function setStatus(id: string, _: FormState, form: FormData): Promise<FormState> {
  try {
    await api<Job>(`/jobs/${id}/status`, {
      method: "PUT",
      body: JSON.stringify({ status: text(form, "status"), note: text(form, "note") || null }),
    });
  } catch (err) {
    return failure(err, form);
  }
  revalidatePath("/");
  revalidatePath(`/jobs/${id}`);
  return undefined;
}

export async function deleteJob(id: string) {
  await api(`/jobs/${id}`, { method: "DELETE" });
  revalidatePath("/");
  redirect("/");
}
