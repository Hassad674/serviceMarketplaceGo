import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/registre — D4 (GDPR Phase C). Public surface for the
// registre des activités de traitement (RGPD art. 30). Source
// Markdown canonique : /legal/registre.md.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.registre" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    robots: { index: false, follow: false },
  }
}

export default async function LegalRegistrePage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.registre" })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const treatments = t("treatmentsItems").split("|")
  const references = t("referenceItems").split("|")

  const sections: LegalSection[] = [
    {
      id: "scope",
      heading: t("introHeading"),
      blocks: [{ type: "p", content: t("introBody") }],
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
      id: "subprocessors",
      heading: t("subprocessorsHeading"),
      blocks: [{ type: "p", content: t("subprocessorsBody") }],
    },
    {
      id: "tom",
      heading: t("tomHeading"),
      blocks: [{ type: "p", content: t("tomBody") }],
    },
    {
      id: "references",
      heading: t("referenceHeading"),
      blocks: [{ type: "ul", items: references }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/registre.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
