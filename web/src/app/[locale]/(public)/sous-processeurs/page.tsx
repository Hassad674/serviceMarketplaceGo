import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /sous-processeurs — public list of all 21 sub-processors enumerated
// in gdpr-audit.md Section 2. Required to be linkable from the privacy
// policy and footer (RGPD art. 28 transparency).
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.subprocessors" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
    robots: { index: false, follow: false },
  }
}

// Vendor list mirrors gdpr-audit.md Section 2 (21 vendors). The
// `transferKey` flag is informational; the cookies page details the
// legal mechanism (DPF / SCC) per vendor.
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
      lastUpdatedISO="2026-05-10"
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
                  {nonEU ? t("transferYes") : t("transferNo")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </LegalShell>
  )
}
