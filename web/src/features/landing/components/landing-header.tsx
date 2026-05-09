import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"

// LandingHeader — top bar of the public landing page.
//
// Soleil v2 anatomy: Atelier wordmark (Fraunces) with corail dot, a
// thin row of contextual nav links, then Se connecter ghost link
// and Créer un compte filled CTA. The mobile mockup collapses the
// inline nav so the bar shows only logo + Se connecter — Tailwind
// breakpoints (`hidden lg:flex` / `hidden sm:inline-flex`) handle
// the responsive split.
//
// Server Component — no event handlers, no state.

const NAV_LINKS = [
  { href: "#how-it-works", labelKey: "howItWorks" },
  { href: "#referrers", labelKey: "providers" },
  { href: "#agencies", labelKey: "agencies" },
  { href: "#pricing", labelKey: "pricing" },
] as const

export function LandingHeader() {
  return (
    <header className="relative z-20 mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-5 sm:px-6 sm:py-6">
      <BrandMark />
      <Nav />
      <AuthLinks />
    </header>
  )
}

function BrandMark() {
  return (
    <Link
      href="/"
      className="inline-flex items-center font-serif text-2xl font-medium tracking-[-0.02em] text-foreground"
    >
      <span>Atelier</span>
      <span aria-hidden="true" className="ml-[2px] text-primary">
        .
      </span>
    </Link>
  )
}

function Nav() {
  const t = useTranslations("landing.nav")
  return (
    <nav
      aria-label={t("howItWorks")}
      className="hidden items-center gap-7 text-sm text-muted-foreground lg:flex"
    >
      {NAV_LINKS.map((link) => (
        <a
          key={link.labelKey}
          href={link.href}
          className="transition-colors hover:text-foreground"
        >
          {t(link.labelKey)}
        </a>
      ))}
    </nav>
  )
}

function AuthLinks() {
  const tCommon = useTranslations("common")
  return (
    <div className="flex items-center gap-3">
      <Link
        href="/login"
        className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
      >
        {tCommon("signIn")}
      </Link>
      <Link
        href="/register"
        className="hidden rounded-full bg-foreground px-5 py-2.5 text-sm font-semibold text-background transition-colors hover:bg-foreground/90 sm:inline-flex"
      >
        {tCommon("createAccount")}
      </Link>
    </div>
  )
}
