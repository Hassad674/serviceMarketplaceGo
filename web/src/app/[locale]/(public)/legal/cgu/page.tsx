import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/cgu — D4 (GDPR Phase C). Public surface for the full CGU.
// Source Markdown canonique : /legal/cgu.md. The short /cgu page
// remains accessible via the legal footer as a synthetic entry point.
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
    robots: { index: false, follow: false },
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
      blocks: [{ type: "p", content: t("behaviorBody") }],
    },
    {
      id: "finance",
      heading: t("financeHeading"),
      blocks: [{ type: "p", content: t("financeBody") }],
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
      id: "termination",
      heading: t("terminationHeading"),
      blocks: [{ type: "p", content: t("terminationBody") }],
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
