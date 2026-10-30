import Link from "next/link";
import { logout } from "@/lib/actions";

export function AppHeader() {
  return (
    <header className="border-b border-gray-200 bg-white">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
        <Link href="/" className="text-lg font-semibold text-gray-900">
          JobPilot
        </Link>
        <nav className="flex items-center gap-4 text-sm">
          <Link href="/jobs/new" className="rounded-md bg-blue-600 px-3 py-1.5 font-medium text-white hover:bg-blue-700">
            Add job
          </Link>
          <form action={logout}>
            <button type="submit" className="text-gray-600 hover:text-gray-900">
              Log out
            </button>
          </form>
        </nav>
      </div>
    </header>
  );
}
