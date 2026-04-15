import { ReferralDetailView } from "@/features/referral/components/referral-detail-view"

interface PageProps {
  params: Promise<{ id: string; locale: string }>
}

// /referrals/[id] — single referral detail. The view itself dispatches
// based on the viewer role (referrer / provider / client) so all three
// parties land on the same URL but see the right anonymised cards and
// available actions.
export default async function ReferralDetailPage({ params }: PageProps) {
  const { id } = await params
  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <ReferralDetailView referralId={id} />
    </main>
  )
}
