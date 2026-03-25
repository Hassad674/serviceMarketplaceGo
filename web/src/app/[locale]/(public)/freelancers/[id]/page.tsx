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
    title: `${t("freelancerProfile")} | Marketplace Service`,
    description: t("freelancerProfileDesc"),
  }
}

export default async function FreelancerProfilePage({ params }: Props) {
  const { id } = await params
  return <PublicProfile userId={id} type="freelancer" />
}
