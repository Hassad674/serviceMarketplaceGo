"use client"

import { CurrentMonthAggregate as InvoicingCurrentMonth } from "@/shared/components/billing-profile/current-month-aggregate"
import { useWallet } from "../hooks/use-wallet"
import { WalletPayoutSection } from "./wallet-payout-section"
import { WalletTransactionsList } from "./wallet-transactions-list"
import { WalletCommissionList } from "./wallet-commission-list"

// Orchestrator for the /wallet page. The actual rendering is owned
// by four sub-components — this file only resolves the wallet
// snapshot, picks the loading / error states, and composes them.
//
// Soleil v2 layout: max-w-[1100px] inner column matching the source
// SoleilWallet maquette. The DashboardShell + Sidebar/Header chrome
// is supplied by the (app) route group — we only render the content.

export function WalletPage() {
  const { data: wallet, isLoading, isError } = useWallet()

  if (isLoading) return <WalletSkeleton />
  if (isError || !wallet) {
    return (
      <div className="mx-auto max-w-[1100px] px-4 py-8 sm:px-6">
        <p className="text-center text-[14px] text-muted-foreground">
          Impossible de charger le wallet
        </p>
      </div>
    )
  }

  const records = wallet.records ?? []
  const commissionRecords = wallet.commission_records ?? []
  const totalEarned =
    wallet.transferred_amount + wallet.commissions.paid_cents

  return (
    <div className="mx-auto w-full max-w-[1100px] space-y-6 px-4 py-8 sm:px-6">
      {/* Hero — total earned + Stripe status + withdraw CTA + modals */}
      <WalletPayoutSection
        totalEarned={totalEarned}
        available={wallet.available_amount}
        stripeAccountId={wallet.stripe_account_id}
        payoutsEnabled={wallet.payouts_enabled}
      />

      {/* Running commission this month — sits above the missions list */}
      <InvoicingCurrentMonth />

      {/* Missions — escrow / available / transferred + history */}
      <WalletTransactionsList
        escrow={wallet.escrow_amount}
        available={wallet.available_amount}
        transferred={wallet.transferred_amount}
        records={records}
      />

      {/* Commissions — apporteur section, hidden when no activity */}
      <WalletCommissionList
        summary={wallet.commissions}
        records={commissionRecords}
      />
    </div>
  )
}

function WalletSkeleton() {
  return (
    <div className="mx-auto w-full max-w-[1100px] space-y-6 px-4 py-8 sm:px-6">
      <div className="h-40 animate-shimmer rounded-2xl bg-border" />
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-28 animate-shimmer rounded-2xl bg-border"
          />
        ))}
      </div>
      <div className="h-64 animate-shimmer rounded-2xl bg-border" />
    </div>
  )
}
