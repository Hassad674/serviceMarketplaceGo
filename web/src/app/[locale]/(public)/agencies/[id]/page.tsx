import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { PublicProfile } from "@/features/provider/components/public-profile"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "publicProfile" })

  return {
    title: `${t("agencyProfile")} | Marketplace Service`,
    description: t("agencyProfileDesc"),
  }
}

export default async function AgencyProfilePage({ params }: Props) {
  const { id } = await params
  return (
    <div className="space-y-6">
      <PublicProfile orgId={id} type="agency" />
      <div className="flex justify-center">
        <SendMessageButton targetOrgId={id} />
      </div>
    </div>
  )
}
