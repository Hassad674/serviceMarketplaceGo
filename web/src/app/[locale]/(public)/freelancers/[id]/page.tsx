import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { PublicProfile } from "@/features/provider/components/public-profile"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"
import { ReviewList } from "@/features/review/components/review-list"

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
  return (
    <div className="space-y-6">
      <PublicProfile userId={id} type="freelancer" />
      <ReviewList userId={id} />
      <div className="flex justify-center">
        <SendMessageButton targetUserId={id} />
      </div>
    </div>
  )
}
