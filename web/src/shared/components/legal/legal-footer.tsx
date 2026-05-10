import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { getDpoEmail } from "@/shared/lib/dpo"

// LegalFooter — minimalist legal-link footer rendered under public
// pages. Surfaces the 6 legal placeholder routes plus the DPO email so
// every visitor has a one-click path to exercise their RGPD rights.
//
// Server-renderable (no client interaction). Uses Soleil v2 tokens.
export function LegalFooter() {
  const t = useTranslations("legal.footer")
  const dpoEmail = getDpoEmail()
  const year = new Date().getFullYear()

  const links: ReadonlyArray<{ href: string; key: string }> = [
    { href: "/privacy", key: "privacy" },
    { href: "/cookies", key: "cookies" },
    { href: "/legal", key: "legal" },
    { href: "/cgu", key: "cgu" },
    { href: "/cgv", key: "cgv" },
    { href: "/sous-processeurs", key: "subprocessors" },
    { href: "/decisions-automatisees", key: "automatedDecisions" },
  ]

  return (
    <footer
      role="contentinfo"
      aria-label={t("ariaLabel")}
      className="border-t border-border bg-background"
    >
      <div className="mx-auto flex max-w-7xl flex-col gap-3 px-6 py-6 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
        <nav aria-label={t("ariaLabel")} className="flex flex-wrap items-center gap-x-4 gap-y-2">
          {links.map(({ href, key }) => (
            <Link
              key={href}
              href={href}
              className="hover:text-foreground hover:underline"
            >
              {t(key)}
            </Link>
          ))}
          <a
            href={`mailto:${dpoEmail}`}
            className="hover:text-foreground hover:underline"
          >
            {t("dpoContact")}
          </a>
        </nav>
        <p className="text-muted-foreground/80">{t("copyright", { year })}</p>
      </div>
    </footer>
  )
}
