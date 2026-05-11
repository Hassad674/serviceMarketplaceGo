import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/aipd — D4 (GDPR Phase C). Public surface for the AIPD
// (RGPD art. 35). Source Markdown canonique : /legal/aipd.md.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.aipd" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    robots: { index: false, follow: false },
  }
}

export default async function LegalAipdPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.aipd" })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const sections: LegalSection[] = [
    {
      id: "scope",
      heading: t("scopeHeading"),
      blocks: [{ type: "p", content: t("scopeBody") }],
    },
    {
      id: "method",
      heading: t("methodHeading"),
      blocks: [{ type: "p", content: t("methodBody") }],
    },
    {
      id: "aipd1",
      heading: t("aipd1Heading"),
      blocks: [{ type: "p", content: t("aipd1Body") }],
    },
    {
      id: "aipd2",
      heading: t("aipd2Heading"),
      blocks: [{ type: "p", content: t("aipd2Body") }],
    },
    {
      id: "aipd3",
      heading: t("aipd3Heading"),
      blocks: [{ type: "p", content: t("aipd3Body") }],
    },
    {
      id: "residual",
      heading: t("residualHeading"),
      blocks: [{ type: "p", content: t("residualBody") }],
    },
    {
      id: "dpo",
      heading: t("dpoHeading"),
      blocks: [{ type: "p", content: t("dpoBody") }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/aipd.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
