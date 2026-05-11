import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { Link } from "@i18n/navigation"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /privacy — placeholder Politique de confidentialité.
// Phase A.4: minimal page with i18n + DPO email + sub-processor link.
// Phase C.1 will replace this with the full policy MDX.

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.privacy" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
    robots: { index: false, follow: false },
  }
}

export default async function PrivacyPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.privacy" })

  return (
    <LegalShell
      titleKey="privacy.title"
      introKey="privacy.intro"
      lastUpdatedISO="2026-05-10"
    >
      <div className="space-y-3">
        <h2 className="font-display text-xl text-foreground">{t("rightsTitle")}</h2>
        <p className="text-sm text-muted-foreground">
          {t.rich("rightsDescription", {
            cnilUrl: () => (
              <a
                href="https://www.cnil.fr/fr/plaintes"
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent underline-offset-4 hover:underline"
              >
                cnil.fr/fr/plaintes
              </a>
            ),
          })}
        </p>
      </div>

      <div className="space-y-3">
        <h2 className="font-display text-xl text-foreground">
          {t("subprocessorsTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t.rich("subprocessorsDescription", {
            link: () => (
              <Link
                href="/sous-processeurs"
                className="text-accent underline-offset-4 hover:underline"
              >
                /sous-processeurs
              </Link>
            ),
          })}
        </p>
      </div>

      <div className="space-y-3">
        <h2 className="font-display text-xl text-foreground">
          {t("automatedDecisionsTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t.rich("automatedDecisionsDescription", {
            link: () => (
              <Link
                href="/decisions-automatisees"
                className="text-accent underline-offset-4 hover:underline"
              >
                /decisions-automatisees
              </Link>
            ),
          })}
        </p>
      </div>
    </LegalShell>
  )
}
