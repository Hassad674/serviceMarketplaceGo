import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { LandingSearchBar } from "./landing-search-bar"

// LandingHero — eyebrow pill, editorial Fraunces title with italic
// corail accent, subtitle, and the centerpiece search bar.
//
// Two CTAs surface on mobile only (the desktop composition uses the
// search bar as the primary action) — the mockup of the responsive
// layout pairs "Créer un compte gratuit" + "Trouver un prestataire"
// directly under the subtitle. Tailwind's `sm:hidden` keeps them out
// of the desktop tree without duplicating the markup.

export function LandingHero() {
  return (
    <section
      aria-labelledby="landing-hero-title"
      className="relative isolate mx-auto w-full max-w-6xl px-4 pb-16 pt-8 sm:px-6 sm:pb-24 lg:pt-12"
    >
      <DecorativeGlow />
      <div className="relative">
        <HeroEyebrow />
        <HeroTitle />
        <HeroSubtitle />
        <MobileCtas />
      </div>
      <div className="relative mt-10 hidden sm:mt-14 sm:block">
        <LandingSearchBar />
      </div>
    </section>
  )
}

function HeroEyebrow() {
  const t = useTranslations("landing.hero")
  return (
    <div className="inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1.5 text-[12px] font-medium text-muted-foreground shadow-[var(--shadow-card)]">
      <span
        aria-hidden="true"
        className="h-1.5 w-1.5 rounded-full bg-success"
      />
      <span className="hidden sm:inline">{t("eyebrowDesktop")}</span>
      <span className="sm:hidden">{t("eyebrowMobile")}</span>
    </div>
  )
}

function HeroTitle() {
  const t = useTranslations("landing.hero")
  return (
    <h1
      id="landing-hero-title"
      className="mt-6 max-w-3xl font-serif text-[40px] font-medium leading-[1.04] tracking-[-0.025em] text-foreground sm:mt-8 sm:text-[56px] lg:text-[68px]"
    >
      <span className="block">{t("titleLead")}</span>
      <span className="block italic text-primary">{t("titleAccent")}</span>
    </h1>
  )
}

function HeroSubtitle() {
  const t = useTranslations("landing.hero")
  return (
    <p className="mt-5 max-w-2xl text-[15px] leading-[1.55] text-muted-foreground sm:mt-7 sm:text-[16px]">
      <span className="hidden sm:inline">{t("subtitleDesktop")}</span>
      <span className="sm:hidden">{t("subtitleMobile")}</span>
    </p>
  )
}

function MobileCtas() {
  const t = useTranslations("landing.hero")
  return (
    <div className="mt-6 flex flex-col gap-3 sm:hidden">
      <Link
        href="/register"
        className="inline-flex h-12 items-center justify-center rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-[var(--shadow-message)] transition-colors hover:bg-primary-deep"
      >
        {t("ctaPrimary")}
      </Link>
      <Link
        href="/freelancers"
        className="inline-flex h-12 items-center justify-center rounded-full border border-border bg-card px-6 text-sm font-semibold text-foreground transition-colors hover:border-border-strong"
      >
        {t("ctaSecondary")}
      </Link>
    </div>
  )
}

// DecorativeGlow renders the soft warm radial that hugs the
// hero's right edge in the desktop mockup. Pure decoration, no
// pointer events, hidden from screen readers.
function DecorativeGlow() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-full overflow-hidden"
    >
      <div className="absolute -right-[10%] top-[-10%] h-[440px] w-[640px] rounded-full bg-primary-soft opacity-70 blur-[120px]" />
      <div className="absolute right-[-5%] top-[20%] h-[280px] w-[420px] rounded-full bg-pink-soft opacity-60 blur-[100px]" />
    </div>
  )
}
