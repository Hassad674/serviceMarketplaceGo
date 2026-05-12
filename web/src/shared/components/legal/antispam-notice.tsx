import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { Info } from "lucide-react"

/**
 * AntispamNotice — discrete reminder of the rules of engagement on
 * the Platform (Code of conduct + anti-spam) for first-time
 * conversations or new-user surfaces.
 *
 * Soleil v2 callout pattern. The notice is i18n-keyed
 * (`legal.antispamNotice.*`) and links to the canonical
 * `/legal/code-de-conduite` page.
 *
 * Designed to be mounted in messaging headers, proposal modals and
 * new-user onboarding flows. Server-renderable.
 */
export function AntispamNotice() {
  const t = useTranslations("legal.antispamNotice")

  return (
    <aside
      role="note"
      aria-label={t("ariaLabel")}
      className="flex items-start gap-3 rounded-2xl border border-border bg-muted/30 p-3 text-xs text-muted-foreground"
    >
      <Info className="size-4 shrink-0 text-accent" aria-hidden />
      <p>
        {t.rich("body", {
          link: (chunks) => (
            <Link
              href="/legal/code-de-conduite"
              className="text-accent underline-offset-4 hover:underline"
            >
              {chunks}
            </Link>
          ),
        })}
      </p>
    </aside>
  )
}
