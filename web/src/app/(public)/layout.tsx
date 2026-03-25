import Link from "next/link"

export default function PublicLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <nav className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
          <Link
            href="/"
            className="text-lg font-bold tracking-tight text-gray-900"
          >
            Marketplace Service
          </Link>
          <div className="flex items-center gap-6">
            <Link
              href="/agencies"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Agencies
            </Link>
            <Link
              href="/freelances"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Freelancers
            </Link>
            <Link
              href="/projects"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Projects
            </Link>
            <Link
              href="/login"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Sign In
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800"
            >
              Sign Up
            </Link>
          </div>
        </nav>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-10">{children}</main>
    </div>
  )
}
