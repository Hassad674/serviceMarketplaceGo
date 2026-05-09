import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"

// LandingCta — the closing CTA card. Soft pink-soft background card
// with two decorative blobs (corail + pink) behind a centered
// editorial title and three role-aware action pills.
//
// Desktop CTA buttons: "Je suis entreprise" (encre) · "Je suis
// prestataire" (corail accent) · "Je suis apporteur" (outline).
// Mobile CTA stack: "Créer un compte" + "Se connecter" (matches the
// responsive mockup which uses fewer, simpler actions on small
// screens).
//
// Server Component.

export function LandingCta() {
  return (
    <section
      aria-labelledby="landing-cta-title"
      className="mx-auto w-full max-w-6xl px-4 pb-16 sm:px-6 sm:pb-24"
    >
      <div className="relative overflow-hidden rounded-[28px] bg-pink-soft px-6 py-16 text-center shadow-[var(--shadow-card)] sm:px-12 sm:py-20">
        <CtaBlobs />
        <CtaTitles />
        <CtaSubtitles />
        <DesktopCtas />
        <MobileCtas />
      </div>
    </section>
  )
}

function CtaBlobs() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 -z-10"
    >
      <div className="absolute left-[10%] top-[20%] h-48 w-48 rounded-full bg-primary-soft blur-3xl" />
      <div className="absolute right-[15%] top-[10%] h-56 w-56 rounded-full bg-pink-soft blur-3xl" />
      <div className="absolute bottom-[15%] left-[40%] h-40 w-40 rounded-full bg-primary-soft blur-3xl" />
    </div>
  )
}

function CtaTitles() {
  const t = useTranslations("landing.ctaSection")
  return (
    <>
      <h2
        id="landing-cta-title"
        className="mx-auto hidden max-w-3xl font-serif text-[36px] font-medium leading-[1.08] tracking-[-0.02em] text-foreground sm:block sm:text-[52px]"
      >
        <span className="block sm:inline">{t("titleLead")}</span>{" "}
        <span className="italic text-primary">{t("titleAccent")}</span>{" "}
        <span>{t("titleTrail")}</span>
      </h2>
      <h2 className="mx-auto max-w-md font-serif text-[32px] font-medium leading-[1.08] tracking-[-0.01em] text-foreground sm:hidden">
        <span>{t("mobileTitleLead")}</span>{" "}
        <span className="italic text-primary">{t("mobileTitleAccent")}</span>{" "}
        <span>{t("mobileTitleTrail")}</span>
      </h2>
    </>
  )
}

function CtaSubtitles() {
  const t = useTranslations("landing.ctaSection")
  return (
    <>
      <p className="mx-auto mt-5 hidden max-w-xl text-[15px] leading-[1.55] text-muted-foreground sm:block">
        {t("subtitle")}
      </p>
      <p className="mx-auto mt-4 max-w-md text-[14px] leading-[1.55] text-muted-foreground sm:hidden">
        {t("mobileSubtitle")}
      </p>
    </>
  )
}

function DesktopCtas() {
  const t = useTranslations("landing.ctaSection")
  return (
    <div className="mt-10 hidden flex-wrap items-center justify-center gap-3 sm:flex">
      <Link
        href="/register?role=enterprise"
        className="inline-flex h-12 items-center justify-center rounded-full bg-foreground px-6 text-sm font-semibold text-background transition-colors hover:bg-foreground/90"
      >
        {t("ctaEnterprise")}
      </Link>
      <Link
        href="/register?role=provider"
        className="inline-flex h-12 items-center justify-center rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-[var(--shadow-message)] transition-colors hover:bg-primary-deep"
      >
        {t("ctaProvider")}
      </Link>
      <Link
        href="/register?role=referrer"
        className="inline-flex h-12 items-center justify-center rounded-full border border-border bg-card px-6 text-sm font-semibold text-foreground transition-colors hover:border-border-strong"
      >
        {t("ctaReferrer")}
      </Link>
    </div>
  )
}

function MobileCtas() {
  const t = useTranslations("landing.ctaSection")
  return (
    <div className="mx-auto mt-8 flex max-w-sm flex-col gap-3 sm:hidden">
      <Link
        href="/register"
        className="inline-flex h-12 items-center justify-center rounded-full bg-foreground px-6 text-sm font-semibold text-background transition-colors hover:bg-foreground/90"
      >
        {t("mobileCtaPrimary")}
      </Link>
      <Link
        href="/login"
        className="inline-flex h-12 items-center justify-center rounded-full border border-border bg-card px-6 text-sm font-semibold text-foreground transition-colors hover:border-border-strong"
      >
        {t("mobileCtaSecondary")}
      </Link>
    </div>
  )
}
