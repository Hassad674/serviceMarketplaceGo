import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/dpa-template — D4 (GDPR Phase C). Public surface for the
// data-processing-agreement template (RGPD art. 28). Source Markdown
// canonique : /legal/dpa-template.md.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.dpaTemplate",
  })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    robots: { index: false, follow: false },
  }
}

export default async function LegalDpaTemplatePage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.dpaTemplate",
  })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const articles = t("articlesItems").split("|")

  const sections: LegalSection[] = [
    {
      id: "scope",
      heading: t("scopeHeading"),
      blocks: [{ type: "p", content: t("scopeBody") }],
    },
    {
      id: "articles",
      heading: t("articlesHeading"),
      blocks: [{ type: "ol", items: articles }],
    },
    {
      id: "transfer",
      heading: t("transferHeading"),
      blocks: [{ type: "p", content: t("transferBody") }],
    },
    {
      id: "appendices",
      heading: t("appendicesHeading"),
      blocks: [
        { type: "p", content: t("appendicesBody") },
        { type: "p", content: t("signatureBody") },
      ],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/dpa-template.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
