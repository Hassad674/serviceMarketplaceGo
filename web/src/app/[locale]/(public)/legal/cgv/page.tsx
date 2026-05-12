import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/cgv — Conditions Générales de Vente (B2B).
//
// Stripe Restricted Businesses + DSA art. 14 + Code de la consommation
// L.111-1 require the CGV to be publicly indexable AND to disclose
// final prices in a clear, precise, unambiguous manner. The legal-
// max-blindage round fixes the rates (5 % Client + 10 % Provider + 5 %
// Referrer), documents the invoicing mandate (art. 289 I-2 CGI), DAC7,
// the Stripe Restricted Businesses alignment, and the platform refund
// policy.
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
    alternates: { canonical: "/legal/cgv" },
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
      blocks: [
        { type: "p", content: t("modelBody") },
        { type: "ul", items: t("modelItems").split("|") },
        { type: "p", content: t("modelStripeFees") },
        { type: "p", content: t("modelPremium") },
      ],
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
      blocks: [
        { type: "p", content: t("invoicesIntroBody") },
        { type: "ul", items: t("invoicesItems").split("|") },
        { type: "p", content: t("invoicesMandate") },
        { type: "p", content: t("invoicesVat") },
        { type: "p", content: t("invoicesRetention") },
      ],
    },
    {
      id: "dac7",
      heading: t("dac7Heading"),
      blocks: [{ type: "p", content: t("dac7Body") }],
    },
    {
      id: "refunds",
      heading: t("refundsHeading"),
      blocks: [{ type: "p", content: t("refundsBody") }],
    },
    {
      id: "restricted",
      heading: t("restrictedHeading"),
      blocks: [
        { type: "p", content: t("restrictedIntro") },
        { type: "ul", items: t("restrictedItems").split("|") },
      ],
    },
    {
      id: "dormant",
      heading: t("dormantHeading"),
      blocks: [{ type: "p", content: t("dormantBody") }],
    },
    {
      id: "liability",
      heading: t("liabilityHeading"),
      blocks: [{ type: "p", content: t("liabilityBody") }],
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
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/cgv.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
