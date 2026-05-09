import type { ReactNode } from "react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { Portrait } from "@/shared/components/ui/portrait"
import { ThemeToggle } from "@/shared/components/theme-toggle"

// AuthPageShell — shared split layout for all Soleil v2 auth screens
// (login, forgot-password, reset-password, …).
//
// Layout: grid 2 cols on desktop (form 1fr + editorial hero 1.2fr).
// Stacks to single column on mobile (hero hidden — form takes full
// width). The right column ("editorial hero") is identical across
// all auth pages by design — it carries the marketplace value
// proposition and the three trust pillars.
//
// Source: design/assets/sources/phase1/soleil-lotE.jsx (W-01 layout
// reused for W-03/W-04). Strings stay i18n-keyed under the existing
// `auth.heroEyebrow`, `auth.heroTitle*`, and `auth.pillar.*` keys —
// already validated by the login redesign.
//
// Usage:
//   <AuthPageShell
//     eyebrow={t("loginEyebrow")}
//     titlePrefix={t("loginTitlePrefix")}
//     titleAccent={t("loginTitleAccent")}
//     subtitle={t("loginSubtitleSoleil")}
//   >
//     <LoginForm />
//   </AuthPageShell>

export interface AuthPageShellProps {
  /** Mono uppercase label above the H1 (e.g. "Bon retour"). */
  eyebrow: string
  /** First half of the H1 (encre). */
  titlePrefix: string
  /** Italic corail accent that closes the H1. */
  titleAccent: string
  /** Supporting paragraph below the H1. */
  subtitle: string
  /** Form (or success state) rendered under the H1. */
  children: ReactNode
}

export function AuthPageShell({
  eyebrow,
  titlePrefix,
  titleAccent,
  subtitle,
  children,
}: AuthPageShellProps) {
  const t = useTranslations("auth")

  return (
    <div className="grid min-h-screen grid-cols-1 lg:grid-cols-[1fr_1.2fr]">
      {/* Left — form column */}
      <section className="relative flex flex-col px-6 py-10 sm:px-10 md:px-16">
        {/* Brand + theme toggle */}
        <div className="flex items-center justify-between">
          <Link
            href="/"
            className="flex items-center gap-2.5 transition-opacity hover:opacity-80"
          >
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary font-serif text-base font-semibold italic text-primary-foreground">
              a
            </span>
            <span className="font-serif text-2xl font-medium tracking-tight text-foreground">
              Atelier
            </span>
          </Link>
          <ThemeToggle />
        </div>

        {/* Centered form */}
        <div className="mx-auto flex w-full max-w-[400px] flex-1 flex-col justify-center py-12">
          <p className="mb-3 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
            {eyebrow}
          </p>
          <h1 className="font-serif text-[36px] font-normal leading-[1.05] tracking-[-0.025em] text-foreground sm:text-[44px]">
            {titlePrefix}{" "}
            <span className="italic text-primary">{titleAccent}</span>
          </h1>
          <p className="mb-8 mt-3 text-sm leading-relaxed text-muted-foreground sm:text-[14.5px]">
            {subtitle}
          </p>

          {children}
        </div>

        {/* Footer — micro */}
        <div className="text-center font-mono text-[11px] text-subtle-foreground">
          © Atelier ·{" "}
          <Link href="/terms" className="hover:text-foreground">
            {t("terms")}
          </Link>{" "}
          ·{" "}
          <Link href="/privacy" className="hover:text-foreground">
            {t("privacy")}
          </Link>
        </div>
      </section>

      {/* Right — editorial hero (desktop only) */}
      <AuthEditorialHero />
    </div>
  )
}

// AuthEditorialHero — the right column carrying the marketplace
// value-prop. Identical across all auth pages by design (login,
// forgot-password, reset-password, …). Kept private to this file so
// callers compose the entire shell rather than the hero alone.
function AuthEditorialHero() {
  const t = useTranslations("auth")

  return (
    <aside className="relative hidden flex-col justify-between overflow-hidden p-14 lg:flex gradient-warm">
      {/* Decorative radial blobs */}
      <div
        className="pointer-events-none absolute -right-20 -top-20 h-80 w-80 rounded-full"
        aria-hidden="true"
        style={{
          background:
            "radial-gradient(circle, rgba(232,93,74,0.25), transparent 65%)",
        }}
      />
      <div
        className="pointer-events-none absolute -left-24 bottom-20 h-64 w-64 rounded-full"
        aria-hidden="true"
        style={{
          background:
            "radial-gradient(circle, rgba(240,138,168,0.35), transparent 65%)",
        }}
      />

      {/* Editorial manifesto */}
      <div className="relative z-10 max-w-[460px]">
        <p className="mb-3.5 font-mono text-[11px] font-bold uppercase tracking-[0.14em] text-primary-deep">
          ↳ {t("heroEyebrow")}
        </p>
        <h2 className="font-serif text-[32px] font-normal leading-tight tracking-[-0.02em] text-foreground text-pretty sm:text-[38px]">
          {t("heroTitlePart1")}{" "}
          <span className="italic text-primary">{t("heroTitleAccent1")}</span>{" "}
          {t("heroTitleConnector")}{" "}
          <span className="italic text-primary">{t("heroTitleAccent2")}</span>
        </h2>
      </div>

      {/* Trio of floating portraits */}
      <div className="relative z-10 my-6 flex flex-1 items-center justify-center">
        <div className="flex items-center">
          <div
            className="-mr-4 rounded-full"
            style={{
              transform: "rotate(-6deg)",
              boxShadow: "var(--shadow-card-strong)",
            }}
          >
            <Portrait id={1} size={88} alt="" />
          </div>
          <div
            className="z-10 rounded-full"
            style={{
              transform: "translateY(-12px)",
              boxShadow: "var(--shadow-card-strong)",
            }}
          >
            <Portrait id={4} size={108} alt="" />
          </div>
          <div
            className="-ml-4 rounded-full"
            style={{
              transform: "rotate(6deg)",
              boxShadow: "var(--shadow-card-strong)",
            }}
          >
            <Portrait id={3} size={88} alt="" />
          </div>
        </div>
      </div>

      {/* Three pillars at bottom */}
      <AuthTrustPillars />
    </aside>
  )
}

const PILLAR_KEYS = ["secure", "verified", "noFee"] as const

function AuthTrustPillars() {
  const t = useTranslations("auth")

  return (
    <div className="relative z-10 grid grid-cols-3 gap-4">
      {PILLAR_KEYS.map((key) => (
        <div key={key}>
          <div
            className="mb-2.5 flex h-8 w-8 items-center justify-center rounded-[10px] bg-surface text-primary-deep"
            style={{ boxShadow: "var(--shadow-portrait)" }}
          >
            <PillarIcon name={key} />
          </div>
          <p className="mb-0.5 text-[13.5px] font-semibold text-foreground">
            {t(`pillar.${key}.label`)}
          </p>
          <p className="text-xs leading-snug text-muted-foreground text-pretty">
            {t(`pillar.${key}.description`)}
          </p>
        </div>
      ))}
    </div>
  )
}

const PILLAR_PATHS: Record<(typeof PILLAR_KEYS)[number], string> = {
  secure: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z",
  verified:
    "M12 3l1.7 5.4 5.6.5-4.3 3.7 1.3 5.5L12 15.3 7.7 18.1 9 12.6 4.7 8.9l5.6-.5z",
  noFee:
    "M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78L12 21.23l8.84-8.84a5.5 5.5 0 0 0 0-7.78z",
}

function PillarIcon({ name }: { name: (typeof PILLAR_KEYS)[number] }) {
  return (
    <svg
      width="15"
      height="15"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={PILLAR_PATHS[name]} />
    </svg>
  )
}
