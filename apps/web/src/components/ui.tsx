import type { ComponentProps } from "react";

export const inputClass =
  "w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-blue-600 focus:outline-none focus:ring-1 focus:ring-blue-600";

export function Field({
  label,
  name,
  error,
  ...props
}: { label: string; name: string; error?: string } & ComponentProps<"input">) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block font-medium text-gray-700">{label}</span>
      <input name={name} className={inputClass} aria-invalid={!!error} {...props} />
      {error && <span className="mt-1 block text-xs text-red-600">{error}</span>}
    </label>
  );
}

export function Button({ className = "", ...props }: ComponentProps<"button">) {
  return (
    <button
      className={`rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 ${className}`}
      {...props}
    />
  );
}

export function FormError({ message }: { message?: string }) {
  if (!message || message === "validation failed") return null;
  return <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{message}</p>;
}
