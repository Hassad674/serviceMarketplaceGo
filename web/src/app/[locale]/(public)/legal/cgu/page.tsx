import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/cgu — Conditions Générales d'Utilisation (B2B).
//
// Stripe Restricted Businesses + DSA art. 14 require the CGU to be
// publicly indexable. The page renders the structured i18n content
// described in `legal.docs.cgu.*` plus the legal-max-blindage
// additions: definitions, anti-circumvention, force majeure,
// modification, worker classification (Malt verbatim), DSA, and
// referrer KYC.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.cgu" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    alternates: { canonical: "/legal/cgu" },
  }
}

export default async function LegalCguPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.cgu" })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const sections: LegalSection[] = [
    {
      id: "definitions",
      heading: t("definitionsHeading"),
      blocks: [
        { type: "p", content: t("definitionsIntro") },
        {
          type: "ul",
          items: t("definitionsItems").split("|"),
        },
      ],
    },
    {
      id: "object",
      heading: t("objectHeading"),
      blocks: [{ type: "p", content: t("objectBody") }],
    },
    {
      id: "access",
      heading: t("accessHeading"),
      blocks: [{ type: "p", content: t("accessBody") }],
    },
    {
      id: "behavior",
      heading: t("behaviorHeading"),
      blocks: [
        { type: "p", content: t("behaviorBody") },
        {
          type: "callout",
          variant: "info",
          content: t("behaviorCodeOfConduct"),
        },
      ],
    },
    {
      id: "finance",
      heading: t("financeHeading"),
      blocks: [{ type: "p", content: t("financeBody") }],
    },
    {
      id: "anti-circumvention",
      heading: t("antiCircumventionHeading"),
      blocks: [
        { type: "p", content: t("antiCircumventionBody") },
        {
          type: "ul",
          items: t("antiCircumventionItems").split("|"),
        },
        {
          type: "callout",
          variant: "warning",
          content: t("antiCircumventionSanction"),
        },
      ],
    },
    {
      id: "ip",
      heading: t("ipHeading"),
      blocks: [{ type: "p", content: t("ipBody") }],
    },
    {
      id: "liability",
      heading: t("liabilityHeading"),
      blocks: [{ type: "p", content: t("liabilityBody") }],
    },
    {
      id: "worker-classification",
      heading: t("workerClassificationHeading"),
      blocks: [
        { type: "p", content: t("workerClassificationIntro") },
        {
          type: "ul",
          items: t("workerClassificationItems").split("|"),
        },
        { type: "p", content: t("workerClassificationDuration") },
        { type: "p", content: t("workerClassificationPlurality") },
        { type: "p", content: t("workerClassificationDisclaimer") },
      ],
    },
    {
      id: "referrer",
      heading: t("referrerHeading"),
      blocks: [{ type: "p", content: t("referrerBody") }],
    },
    {
      id: "moderation",
      heading: t("moderationHeading"),
      blocks: [
        { type: "p", content: t("moderationBody") },
        {
          type: "ul",
          items: t("moderationItems").split("|"),
        },
      ],
    },
    {
      id: "suspension",
      heading: t("suspensionHeading"),
      blocks: [{ type: "p", content: t("suspensionBody") }],
    },
    {
      id: "termination",
      heading: t("terminationHeading"),
      blocks: [{ type: "p", content: t("terminationBody") }],
    },
    {
      id: "modification",
      heading: t("modificationHeading"),
      blocks: [{ type: "p", content: t("modificationBody") }],
    },
    {
      id: "force-majeure",
      heading: t("forceMajeureHeading"),
      blocks: [{ type: "p", content: t("forceMajeureBody") }],
    },
    {
      id: "law",
      heading: t("lawHeading"),
      blocks: [{ type: "p", content: t("lawBody") }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/cgu.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
