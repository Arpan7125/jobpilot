import "server-only";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

// The browser never sees the JWT: it lives in an httpOnly cookie and only
// this server-side code sends it on to the Go API.

const API_URL = process.env.API_URL ?? "http://localhost:4000";
export const TOKEN_COOKIE = "jp_token";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public fields?: Record<string, string>,
  ) {
    super(message);
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  });
  if (res.status === 401 && !path.startsWith("/auth/")) {
    redirect("/login");
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, body.error ?? `API returned ${res.status}`, body.fields);
  }
  return body as T;
}

export async function setToken(token: string) {
  (await cookies()).set(TOKEN_COOKIE, token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24, // matches the API's 24h token lifetime
  });
}

export async function clearToken() {
  (await cookies()).delete(TOKEN_COOKIE);
}
