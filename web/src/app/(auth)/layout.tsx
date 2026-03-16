import Link from "next/link"

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen flex-col bg-gray-50">
      <header className="flex h-16 items-center px-6">
        <Link
          href="/"
          className="text-lg font-bold tracking-tight text-gray-900"
        >
          Marketplace Service
        </Link>
      </header>
      <main className="flex flex-1 items-center justify-center px-6 py-12">
        {children}
      </main>
    </div>
  )
}
