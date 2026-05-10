import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /legal — placeholder mentions légales (LCEN art. 6 III).
// Editor identity, RCS, capital, hosting info. Awaits the legal entity
// registration to be filled with real values (Phase C.3).
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.mentions" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
    robots: { index: false, follow: false },
  }
}

export default async function MentionsLegalesPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.mentions" })

  return (
    <LegalShell titleKey="mentions.title" introKey="mentions.intro" lastUpdatedISO="2026-05-10">
      <div className="space-y-2">
        <h2 className="font-display text-xl text-foreground">{t("editorTitle")}</h2>
        <p className="text-sm text-muted-foreground">{t("editorPlaceholder")}</p>
      </div>
      <div className="space-y-2">
        <h2 className="font-display text-xl text-foreground">{t("hostingTitle")}</h2>
        <p className="text-sm text-muted-foreground">{t("hostingDescription")}</p>
      </div>
    </LegalShell>
  )
}
