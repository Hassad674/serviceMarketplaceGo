import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/cgv — D4 (GDPR Phase C). Public surface for the full CGV.
// Source Markdown canonique : /legal/cgv.md. The short /cgv page
// remains accessible via the legal footer as a synthetic entry point.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.cgv" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    robots: { index: false, follow: false },
  }
}

export default async function LegalCgvPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.docs.cgv" })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const sections: LegalSection[] = [
    {
      id: "model",
      heading: t("modelHeading"),
      blocks: [{ type: "p", content: t("modelBody") }],
    },
    {
      id: "kyc",
      heading: t("kycHeading"),
      blocks: [{ type: "p", content: t("kycBody") }],
    },
    {
      id: "cycle",
      heading: t("cycleHeading"),
      blocks: [{ type: "p", content: t("cycleBody") }],
    },
    {
      id: "payment",
      heading: t("paymentHeading"),
      blocks: [{ type: "p", content: t("paymentBody") }],
    },
    {
      id: "dispute",
      heading: t("disputeHeading"),
      blocks: [{ type: "p", content: t("disputeBody") }],
    },
    {
      id: "referrer",
      heading: t("referrerHeading"),
      blocks: [{ type: "p", content: t("referrerBody") }],
    },
    {
      id: "invoices",
      heading: t("invoicesHeading"),
      blocks: [{ type: "p", content: t("invoicesBody") }],
    },
    {
      id: "dormant",
      heading: t("dormantHeading"),
      blocks: [{ type: "p", content: t("dormantBody") }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/cgv.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
