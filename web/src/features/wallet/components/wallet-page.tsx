"use client"

import { CurrentMonthAggregate as InvoicingCurrentMonth } from "@/features/invoicing/components/current-month-aggregate"
import { useWallet } from "../hooks/use-wallet"
import { WalletPayoutSection } from "./wallet-payout-section"
import { WalletTransactionsList } from "./wallet-transactions-list"
import { WalletCommissionList } from "./wallet-commission-list"

// Orchestrator for the /wallet page. The actual rendering is owned
// by four sub-components — this file only resolves the wallet
// snapshot, picks the loading / error states, and composes them.

export function WalletPage() {
  const { data: wallet, isLoading, isError } = useWallet()

  if (isLoading) return <WalletSkeleton />
  if (isError || !wallet) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8">
        <p className="text-center text-slate-500">
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
    <div className="mx-auto max-w-4xl px-4 py-8 space-y-8">
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
    <div className="mx-auto max-w-4xl space-y-6 px-4 py-8">
      <div className="h-32 animate-shimmer rounded-2xl bg-slate-200 dark:bg-slate-700" />
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-28 animate-shimmer rounded-xl bg-slate-200 dark:bg-slate-700"
          />
        ))}
      </div>
      <div className="h-64 animate-shimmer rounded-2xl bg-slate-200 dark:bg-slate-700" />
    </div>
  )
}
