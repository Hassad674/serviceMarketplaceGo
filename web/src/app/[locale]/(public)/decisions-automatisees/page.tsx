import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalShell } from "@/shared/components/legal/legal-shell"
import { getDpoEmail } from "@/shared/lib/dpo"

// /decisions-automatisees — RGPD art. 22 disclosure (Phase B.5).
//
// Tells users about the three automated decisions running on the
// marketplace (AI moderation, search ranking, Stripe risk scoring),
// the consequences of each decision, and the appeal procedure to
// obtain human review. Server-rendered, indexed-by-default per
// transparency obligation (article 13).
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.automatedDecisions",
  })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
  }
}

export default async function AutomatedDecisionsPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.automatedDecisions",
  })
  const dpoEmail = getDpoEmail()

  return (
    <LegalShell
      titleKey="automatedDecisions.title"
      introKey="automatedDecisions.intro"
      lastUpdatedISO="2026-05-10"
    >
      <SystemSection
        title={t("systems.moderationTitle")}
        description={t("systems.moderationDescription")}
        consequence={t("systems.moderationConsequence")}
      />
      <SystemSection
        title={t("systems.rankingTitle")}
        description={t("systems.rankingDescription")}
        consequence={t("systems.rankingConsequence")}
      />
      <SystemSection
        title={t("systems.paymentTitle")}
        description={t("systems.paymentDescription")}
        consequence={t("systems.paymentConsequence")}
      />

      <div className="space-y-3">
        <h2 className="font-display text-xl text-foreground">
          {t("rightsTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("rightsDescription")}</p>
      </div>

      <div className="space-y-3">
        <h2 className="font-display text-xl text-foreground">
          {t("appealTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t.rich("appealDescription", {
            email: () => (
              <a
                href={`mailto:${dpoEmail}`}
                className="text-accent underline-offset-4 hover:underline"
              >
                {dpoEmail}
              </a>
            ),
          })}
        </p>
      </div>

      <div className="space-y-3">
        <h2 className="font-display text-xl text-foreground">
          {t("appealAlternativeTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t.rich("appealAlternativeDescription", {
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
    </LegalShell>
  )
}

interface SystemSectionProps {
  title: string
  description: string
  consequence: string
}

function SystemSection({ title, description, consequence }: SystemSectionProps) {
  return (
    <article className="space-y-2 rounded-2xl border border-border bg-card p-5">
      <h2 className="font-display text-xl text-foreground">{title}</h2>
      <p className="text-sm text-muted-foreground">{description}</p>
      <p className="text-sm text-foreground">{consequence}</p>
    </article>
  )
}
