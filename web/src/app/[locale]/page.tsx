import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { Building2, User, Briefcase, ArrowRight } from "lucide-react"
import { ThemeToggle } from "@/shared/components/theme-toggle"
import { TestDB } from "./test-db"

export default function HomePage() {
  const t = useTranslations("landing")
  const tCommon = useTranslations("common")

  const features = [
    {
      icon: Building2,
      titleKey: "agenciesTitle" as const,
      descKey: "agenciesDesc" as const,
      href: "/agencies" as const,
      linkKey: "browseAgencies" as const,
      iconBg: "bg-blue-100 dark:bg-blue-500/20",
      iconColor: "text-blue-600 dark:text-blue-400",
    },
    {
      icon: User,
      titleKey: "freelancesTitle" as const,
      descKey: "freelancesDesc" as const,
      href: "/freelances" as const,
      linkKey: "browseFreelances" as const,
      iconBg: "bg-rose-100 dark:bg-rose-500/20",
      iconColor: "text-rose-600 dark:text-rose-400",
    },
    {
      icon: Briefcase,
      titleKey: "enterprisesTitle" as const,
      descKey: "enterprisesDesc" as const,
      href: "/projects" as const,
      linkKey: "viewProjects" as const,
      iconBg: "bg-emerald-100 dark:bg-emerald-500/20",
      iconColor: "text-emerald-600 dark:text-emerald-400",
    },
  ]

  return (
    <main className="flex min-h-screen flex-col bg-white dark:bg-gray-950">
      {/* Navbar */}
      <header className="absolute top-0 z-10 w-full">
        <nav className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
          <span className="text-xl font-bold tracking-tight text-white">
            Marketplace
          </span>
          <div className="flex items-center gap-3">
            <ThemeToggle className="border-white/20 bg-white/10 dark:bg-gray-800/50 shadow-none hover:shadow-none hover:bg-white/20" />
            <Link
              href="/login"
              className="rounded-xl px-4 py-2 text-sm font-medium text-white/90 transition-colors hover:text-white"
            >
              {tCommon("signIn")}
            </Link>
            <Link
              href="/register"
              className="rounded-xl bg-white px-5 py-2 text-sm font-semibold text-rose-600 shadow-md transition-all hover:shadow-lg active:scale-[0.98]"
            >
              {tCommon("createAccount")}
            </Link>
          </div>
        </nav>
      </header>

      {/* Hero */}
      <section className="gradient-hero relative flex min-h-[80vh] flex-col items-center justify-center overflow-hidden px-6 pt-16 text-center">
        {/* Decorative floating shapes */}
        <div className="animate-float pointer-events-none absolute left-[10%] top-[20%] h-64 w-64 rounded-full bg-white/10 blur-3xl" />
        <div className="animate-float-delayed pointer-events-none absolute right-[15%] top-[30%] h-48 w-48 rounded-full bg-white/10 blur-3xl" />
        <div className="animate-float-slow pointer-events-none absolute bottom-[20%] left-[30%] h-56 w-56 rounded-full bg-white/5 blur-3xl" />

        <h1 className="relative max-w-4xl text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl lg:text-6xl">
          {t("heroTitle")}
        </h1>
        <p className="relative mt-6 max-w-2xl text-lg leading-relaxed text-white/80">
          {t("heroSubtitle")}
        </p>
        <div className="relative mt-10 flex flex-wrap items-center justify-center gap-4">
          <Link
            href="/register"
            className="rounded-xl bg-white px-8 py-3.5 text-sm font-semibold text-rose-600 shadow-lg transition-all hover:shadow-xl active:scale-[0.98]"
          >
            {t("getStarted")}
          </Link>
          <Link
            href="/projects"
            className="rounded-xl border border-white/20 px-8 py-3.5 text-sm font-semibold text-white backdrop-blur-sm transition-all hover:border-white/40 hover:bg-white/10"
          >
            {t("browseProjects")}
          </Link>
        </div>
      </section>

      {/* Feature cards */}
      <section className="mx-auto grid w-full max-w-7xl gap-8 px-6 py-24 sm:grid-cols-3 dark:bg-gray-950">
        {features.map((feature) => {
          const Icon = feature.icon
          return (
            <div
              key={feature.titleKey}
              className="group rounded-2xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-8 shadow-sm transition-all duration-200 hover:shadow-md"
            >
              <div className={`mb-5 flex h-12 w-12 items-center justify-center rounded-xl ${feature.iconBg}`}>
                <Icon className={`h-6 w-6 ${feature.iconColor}`} />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t(feature.titleKey)}</h3>
              <p className="mt-2 text-sm leading-relaxed text-gray-500 dark:text-gray-400">
                {t(feature.descKey)}
              </p>
              <Link
                href={feature.href}
                className="mt-5 inline-flex items-center gap-1.5 text-sm font-medium text-rose-500 transition-colors hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300"
              >
                {t(feature.linkKey)}
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Link>
            </div>
          )
        })}
      </section>

      {/* Debug: Test DB connectivity */}
      <TestDB />
    </main>
  )
}
