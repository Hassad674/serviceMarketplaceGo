import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /cookies — placeholder cookies and tracker disclosure.
// Lists every cookie / localStorage key set by the service. Source of
// truth aligned with the Tarteaucitron service config (Phase A.2).
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.cookies" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
    robots: { index: false, follow: false },
  }
}

const ROW_KEYS = ["session", "consent", "stripe", "posthog", "ga4", "locale"] as const

export default async function CookiesPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.cookies" })

  return (
    <LegalShell titleKey="cookies.title" introKey="cookies.intro" lastUpdatedISO="2026-05-10">
      <div className="overflow-x-auto rounded-2xl border border-border bg-card">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-border bg-muted/30 text-xs uppercase text-muted-foreground">
            <tr>
              <th scope="col" className="px-4 py-3">{t("tableHeader.name")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.provider")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.type")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.purpose")}</th>
              <th scope="col" className="px-4 py-3">{t("tableHeader.duration")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border text-foreground">
            {ROW_KEYS.map((key) => (
              <tr key={key}>
                <td className="px-4 py-3 font-mono text-xs">{t(`rows.${key}.name`)}</td>
                <td className="px-4 py-3">{t(`rows.${key}.provider`)}</td>
                <td className="px-4 py-3 text-muted-foreground">{t(`rows.${key}.type`)}</td>
                <td className="px-4 py-3 text-muted-foreground">{t(`rows.${key}.purpose`)}</td>
                <td className="px-4 py-3 text-muted-foreground">{t(`rows.${key}.duration`)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </LegalShell>
  )
}
