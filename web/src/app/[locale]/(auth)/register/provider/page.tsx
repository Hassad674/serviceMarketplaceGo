import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { ProviderRegisterForm } from "@/features/auth/components/provider-register-form"
import { Portrait } from "@/shared/components/ui/portrait"
import { ThemeToggle } from "@/shared/components/theme-toggle"

// Step 2/2 · Inscription Freelance · Soleil v2 polish.
//
// Mirrors the W-01 Login + W-02 Register step-1 visual identity:
// split 2 cols on desktop (form 1fr + editorial hero 1.2fr), single
// column on mobile (hero hidden). The AtelierMark + theme toggle live
// inside the form column so the layout matches the rest of the auth
// flow exactly.
//
// Copy / structure / fields / validation rules / submit handlers are
// untouched — only the visual chrome changes.
export default function RegisterProviderPage() {
  const t = useTranslations("registerFix")
  const tAuth = useTranslations("auth")
  const tCommon = useTranslations("common")

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
        <div className="mx-auto flex w-full max-w-[480px] flex-1 flex-col justify-center py-12">
          <p className="mb-3 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
            {t("provider.eyebrow")}
          </p>
          <h1 className="font-serif text-[36px] font-normal leading-[1.05] tracking-[-0.025em] text-foreground sm:text-[44px]">
            {t("provider.titlePrefix")}{" "}
            <span className="italic text-primary">
              {t("provider.titleAccent")}
            </span>
          </h1>
          <p className="mb-8 mt-3 text-[15px] leading-relaxed text-muted-foreground">
            {t("provider.subtitle")}
          </p>

          <ProviderRegisterForm />

          {/* Bottom secondary actions */}
          <div className="mt-7 flex flex-col items-center gap-2.5 text-[13px] sm:flex-row sm:justify-between">
            <Link
              href="/register"
              className="inline-flex items-center gap-1.5 font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              <span aria-hidden="true">←</span>
              {t("shared.changeProfile")}
            </Link>
            <p className="text-muted-foreground">
              {t("shared.alreadyAccount")}{" "}
              <Link
                href="/login"
                className="font-semibold text-primary transition-colors hover:text-primary-deep"
              >
                {tCommon("signIn")}
              </Link>
            </p>
          </div>
        </div>

        {/* Footer — micro */}
        <div className="text-center font-mono text-[11px] text-subtle-foreground">
          © Atelier ·{" "}
          <Link href="/terms" className="hover:text-foreground">
            {tAuth("terms")}
          </Link>{" "}
          ·{" "}
          <Link href="/privacy" className="hover:text-foreground">
            {tAuth("privacy")}
          </Link>
        </div>
      </section>

      {/* Right — editorial hero (desktop only) */}
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
            ↳ {tAuth("heroEyebrow")}
          </p>
          <h2 className="font-serif text-[32px] font-normal leading-tight tracking-[-0.02em] text-foreground text-pretty sm:text-[38px]">
            {tAuth("heroTitlePart1")}{" "}
            <span className="italic text-primary">
              {tAuth("heroTitleAccent1")}
            </span>{" "}
            {tAuth("heroTitleConnector")}{" "}
            <span className="italic text-primary">
              {tAuth("heroTitleAccent2")}
            </span>
          </h2>
        </div>

        {/* Single decorative portrait */}
        <div className="relative z-10 my-6 flex flex-1 items-center justify-center">
          <div
            className="rounded-full"
            style={{ boxShadow: "var(--shadow-card-strong)" }}
          >
            <Portrait id={1} size={132} alt="" />
          </div>
        </div>

        {/* Three pillars at bottom — same as login */}
        <div className="relative z-10 grid grid-cols-3 gap-4">
          {(["secure", "verified", "noFee"] as const).map((key) => (
            <div key={key}>
              <div
                className="mb-2.5 flex h-8 w-8 items-center justify-center rounded-[10px] bg-surface text-primary-deep"
                style={{ boxShadow: "var(--shadow-portrait)" }}
                aria-hidden="true"
              >
                <span className="font-serif text-base font-semibold italic">
                  ✦
                </span>
              </div>
              <p className="mb-0.5 text-[13.5px] font-semibold text-foreground">
                {tAuth(`pillar.${key}.label`)}
              </p>
              <p className="text-xs leading-snug text-muted-foreground text-pretty">
                {tAuth(`pillar.${key}.description`)}
              </p>
            </div>
          ))}
        </div>
      </aside>
    </div>
  )
}
