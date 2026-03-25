import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { PublicProfile } from "@/features/provider/components/public-profile"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "publicProfile" })

  return {
    title: `${t("referrerProfile")} | Marketplace Service`,
    description: t("referrerProfileDesc"),
  }
}

export default async function ReferrerProfilePage({ params }: Props) {
  const { id } = await params
  return <PublicProfile userId={id} type="referrer" />
}
