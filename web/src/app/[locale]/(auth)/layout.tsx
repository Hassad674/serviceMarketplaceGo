import { Link } from "@i18n/navigation"

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-b from-gray-50 to-white">
      <header className="flex h-16 items-center px-6">
        <Link
          href="/"
          className="text-xl font-bold tracking-tight text-gray-900"
        >
          Marketplace
        </Link>
      </header>
      <main className="flex flex-1 items-center justify-center px-6 py-12">
        {children}
      </main>
    </div>
  )
}
