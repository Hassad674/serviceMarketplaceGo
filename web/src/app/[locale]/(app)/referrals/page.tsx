import { ReferralDashboard } from "@/features/referral/components/referral-dashboard"

// /referrals — the apporteur dashboard for business referrals.
//
// Distinct from /referral (singular) which remains the public referrer
// profile editor. The naming was chosen to preserve backward compat with
// existing links and i18n keys.
export default function ReferralsPage() {
  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <ReferralDashboard />
    </main>
  )
}
