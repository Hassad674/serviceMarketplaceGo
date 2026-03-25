import { Link } from "@i18n/navigation"
import { ThemeToggle } from "@/shared/components/theme-toggle"

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-b from-gray-50 to-white dark:from-gray-950 dark:to-gray-900">
      <header className="flex h-16 items-center justify-between px-6">
        <Link
          href="/"
          className="text-xl font-bold tracking-tight text-gray-900 dark:text-white"
        >
          Marketplace
        </Link>
        <ThemeToggle />
      </header>
      <main className="flex flex-1 items-center justify-center px-6 py-12">
        {children}
      </main>
    </div>
  )
}
