import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { getDpoEmail } from "@/shared/lib/dpo"

// LegalShell is the consistent wrapper for the legal pages
// (/privacy, /cookies, /legal, /cgu, /cgv, /sous-processeurs).
//
// Soleil v2: ivoire bg via the parent layout, Fraunces title, content
// max-width 5xl. Uses semantic tokens only — no hex.
//
// The component is server-renderable: every label is read via
// `useTranslations` (server-safe via next-intl 4.x) and the DPO email
// is read once from env at request time.
//
// The "draft notice" banner ("cette page sera complétée…") is OPT-IN
// via `showDraftNotice`. It defaults to false: every legal surface now
// ships final, publicly-indexable content (Stripe Restricted
// Businesses + DSA art. 14 reject a public legal page that advertises
// itself as unfinished). Set it true only for a genuinely pending
// page.
export interface LegalShellProps {
  titleKey: string
  introKey?: string
  lastUpdatedISO: string
  showDraftNotice?: boolean
  children?: React.ReactNode
}

export function LegalShell({
  titleKey,
  introKey,
  lastUpdatedISO,
  showDraftNotice = false,
  children,
}: LegalShellProps) {
  const t = useTranslations("legal")
  const dpoEmail = getDpoEmail()
  const formattedDate = new Date(lastUpdatedISO).toISOString().slice(0, 10)

  return (
    <article className="mx-auto w-full max-w-5xl space-y-6 py-12">
      <header className="space-y-3 border-b border-border pb-6">
        <h1 className="font-display text-3xl text-foreground sm:text-4xl">
          {t(titleKey)}
        </h1>
        {introKey ? (
          <p className="text-muted-foreground">{t(introKey)}</p>
        ) : null}
        <p className="text-xs text-muted-foreground">
          {t("common.lastUpdated", { date: formattedDate })}
        </p>
      </header>

      <section className="space-y-4 text-foreground">
        {showDraftNotice ? (
          <p className="rounded-2xl border border-border bg-card p-4 text-sm text-muted-foreground">
            {t("common.placeholder")}
          </p>
        ) : null}
        {children}
      </section>

      <footer className="space-y-3 border-t border-border pt-6 text-sm text-muted-foreground">
        <p>
          {t("common.contactDpo", { email: dpoEmail })}
        </p>
        <Link href="/" className="text-accent underline-offset-4 hover:underline">
          {t("common.back")}
        </Link>
      </footer>
    </article>
  )
}
