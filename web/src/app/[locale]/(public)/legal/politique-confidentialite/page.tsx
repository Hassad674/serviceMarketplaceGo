import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/politique-confidentialite — single canonical privacy policy.
//
// The short /privacy page was merged into this URL (May 2026 — CNIL
// requires a single, exhaustive policy). This page covers RGPD art.
// 5 (principles), 6 (legality), 7 (consent), 9 (special categories),
// 12 (transparency), 13-14 (information), 15-22 (rights), 22
// (automated decisions), 28 (sub-processors), 33-34 (breach), 44-49
// (transfers), and the 11 processing activities documented in the
// art. 30 register.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.politiqueConfidentialite",
  })
  // CNIL — the privacy policy is the canonical single source of
  // truth (after the May 2026 merge of the short /privacy into this
  // URL) and MUST be crawlable. RGPD art. 12 requires accessibility
  // in a clear, intelligible, easily-accessible format.
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    alternates: { canonical: "/legal/politique-confidentialite" },
  }
}

export default async function LegalPolitiquePage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.politiqueConfidentialite",
  })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const summary = t("summaryItems").split("|")
  const rights = t("rightsItems").split("|")
  const treatments = t("treatmentsItems").split("|")

  const sections: LegalSection[] = [
    {
      id: "summary",
      heading: t("summaryHeading"),
      blocks: [{ type: "ul", items: summary }],
    },
    {
      id: "rights",
      heading: t("rightsHeading"),
      blocks: [{ type: "ul", items: rights }],
    },
    {
      id: "treatments",
      heading: t("treatmentsHeading"),
      blocks: [
        { type: "p", content: t("treatmentsIntro") },
        { type: "ul", items: treatments },
      ],
    },
    {
      id: "retention",
      heading: t("retentionHeading"),
      blocks: [{ type: "p", content: t("retentionBody") }],
    },
    {
      id: "profiling",
      heading: t("profilingHeading"),
      blocks: [{ type: "p", content: t("profilingBody") }],
    },
    {
      id: "transfer",
      heading: t("transferHeading"),
      blocks: [{ type: "p", content: t("transferBody") }],
    },
    {
      id: "subprocessors",
      heading: t("subprocessorsHeading"),
      blocks: [{ type: "p", content: t("subprocessorsBody") }],
    },
    {
      id: "special-categories",
      heading: t("specialCategoriesHeading"),
      blocks: [{ type: "p", content: t("specialCategoriesBody") }],
    },
    {
      id: "minors",
      heading: t("minorsHeading"),
      blocks: [{ type: "p", content: t("minorsBody") }],
    },
    {
      id: "breach",
      heading: t("breachHeading"),
      blocks: [{ type: "p", content: t("breachBody") }],
    },
    {
      id: "automated-decisions",
      heading: t("automatedDecisionsHeading"),
      blocks: [{ type: "p", content: t("automatedDecisionsBody") }],
    },
    {
      id: "contact",
      heading: t("contactHeading"),
      blocks: [{ type: "p", content: t("contactBody") }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/politique-confidentialite.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
