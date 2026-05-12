import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/code-de-conduite — Code of conduct and moderation policy.
//
// Indexable: DSA art. 14 requires conditions and policies to be in a
// clear, easily-accessible format. This page is referenced from the
// CGU (article 4) and from the suspension/ban procedure described in
// CGU art. 12 and 16.
//
// Content inspired by Malt Charter of Honour + Upwork Trust & Safety +
// Contra Community Guidelines, adapted to French law and DSA. The
// canonical body lives in i18n keys `legal.docs.codeOfConduct.*` to
// keep the content reviewable + bilingual without editing the page.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.codeOfConduct",
  })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    alternates: { canonical: "/legal/code-de-conduite" },
  }
}

export default async function LegalCodeOfConductPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.codeOfConduct",
  })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const sections: LegalSection[] = [
    {
      id: "preamble",
      heading: t("preambleHeading"),
      blocks: [{ type: "p", content: t("preambleBody") }],
    },
    {
      id: "prohibited",
      heading: t("prohibitedHeading"),
      blocks: [
        { type: "p", content: t("prohibitedIntro") },
        { type: "ul", items: t("prohibitedItems").split("|") },
      ],
    },
    {
      id: "report",
      heading: t("reportHeading"),
      blocks: [{ type: "p", content: t("reportBody") }],
    },
    {
      id: "sanctions",
      heading: t("sanctionsHeading"),
      blocks: [
        { type: "p", content: t("sanctionsIntro") },
        { type: "ul", items: t("sanctionsItems").split("|") },
      ],
    },
    {
      id: "appeal",
      heading: t("appealHeading"),
      blocks: [{ type: "p", content: t("appealBody") }],
    },
    {
      id: "dsa",
      heading: t("dsaHeading"),
      blocks: [{ type: "p", content: t("dsaBody") }],
    },
    {
      id: "transparency",
      heading: t("transparencyHeading"),
      blocks: [{ type: "p", content: t("transparencyBody") }],
    },
    {
      id: "review",
      heading: t("reviewHeading"),
      blocks: [{ type: "p", content: t("reviewBody") }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/code-de-conduite.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
