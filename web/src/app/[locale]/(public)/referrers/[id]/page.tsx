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
    title: `${t("referrerProfile")} | Marketplace Service`,
    description: t("referrerProfileDesc"),
  }
}

export default async function ReferrerProfilePage({ params }: Props) {
  const { id } = await params
  return (
    <div className="space-y-6">
      <PublicProfile userId={id} type="referrer" />
      <div className="flex justify-center">
        <SendMessageButton targetUserId={id} />
      </div>
    </div>
  )
}
