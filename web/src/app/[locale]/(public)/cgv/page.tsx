import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /cgv — placeholder Conditions Générales de Vente. Phase C.2 will
// replace with the full legally-reviewed document.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal.cgv" })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("intro"),
    robots: { index: false, follow: false },
  }
}

export default function CgvPage() {
  return <LegalShell titleKey="cgv.title" introKey="cgv.intro" lastUpdatedISO="2026-05-10" />
}
