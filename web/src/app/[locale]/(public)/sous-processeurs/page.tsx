import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /sous-processeurs — public list of all 21 sub-processors enumerated
// in gdpr-audit.md Section 2. Required to be linkable from the privacy
// policy and footer (RGPD art. 28 transparency + DSA art. 14).
//
// legal-max-blindage update — per-vendor transfer mechanism (DPF vs.
// SCC) surfaced explicitly. Schrems II note clarifies that the DPF is
// always supplementary; SCC 2021/914 are the primary basis.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.subprocessors" })
  // RGPD art. 28 transparency + DSA art. 14: the sub-processors
  // list must be publicly indexable so Stripe + auditors + visitors
  // can verify the third parties handling personal data.
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
    alternates: { canonical: "/sous-processeurs" },
  }
}

// Vendor list mirrors gdpr-audit.md Section 2 (21 vendors). The
// `nonEU` flag drives the column "transfer mechanism" — vendors are
// either EU-only (no transfer) or non-EU (mechanism in
// `rows.<key>.transferMech`).
const VENDORS: ReadonlyArray<{ key: string; nonEU: boolean }> = [
  { key: "vercel", nonEU: true },
  { key: "railway", nonEU: true },
  { key: "neon", nonEU: false },
  { key: "r2", nonEU: true },
  { key: "resend", nonEU: true },
  { key: "stripe", nonEU: true },
  { key: "livekit", nonEU: true },
  { key: "openai", nonEU: true },
  { key: "anthropic", nonEU: true },
  { key: "rekognition", nonEU: false },
  { key: "s3transit", nonEU: false },
  { key: "snssqs", nonEU: false },
  { key: "fcm", nonEU: true },
  { key: "typesense", nonEU: false },
  { key: "posthog", nonEU: false },
  { key: "ga4", nonEU: true },
  { key: "vies", nonEU: false },
  { key: "nominatim", nonEU: false },
  { key: "ban", nonEU: false },
  { key: "photon", nonEU: false },
  { key: "cloudflare", nonEU: true },
]

export default async function SubprocessorsPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.subprocessors" })

  return (
    <LegalShell
      titleKey="subprocessors.title"
      introKey="subprocessors.intro"
      lastUpdatedISO="2026-05-12"
    >
      <div className="overflow-x-auto rounded-2xl border border-border bg-card">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-border bg-muted/30 text-xs uppercase text-muted-foreground">
            <tr>
              <th scope="col" className="px-4 py-3">{t("tableHeader.vendor")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.country")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.purpose")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.transfer")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border text-foreground">
            {VENDORS.map(({ key, nonEU }) => (
              <tr key={key}>
                <td className="px-4 py-3 font-medium">{t(`rows.${key}.vendor`)}</td>
                <td className="px-4 py-3 text-muted-foreground">{t(`rows.${key}.country`)}</td>
                <td className="px-4 py-3 text-muted-foreground">{t(`rows.${key}.purpose`)}</td>
                <td className="px-4 py-3 text-muted-foreground">
                  {nonEU ? t(`rows.${key}.transferMech`) : t("transferEU")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <section className="space-y-3 border-t border-border pt-6">
        <h2 className="font-display text-xl text-foreground">
          {t("dpfNote").split("—")[0].trim()}
        </h2>
        <p className="text-sm text-muted-foreground">{t("dpfNote")}</p>
        <p className="text-sm text-muted-foreground">{t("auditNote")}</p>
      </section>
    </LegalShell>
  )
}
