"use client"

import { useState } from "react"
import {
  Wallet,
  ArrowDownToLine,
  Clock,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertTriangle,
  DollarSign,
  ShieldCheck,
  Send,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { useWallet, useRequestPayout } from "../hooks/use-wallet"
import type { WalletRecord } from "../api/wallet-api"

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}

export function WalletPage() {
  const { data: wallet, isLoading, isError } = useWallet()
  const payoutMutation = useRequestPayout()
  const [payoutMessage, setPayoutMessage] = useState("")

  if (isLoading) return <WalletSkeleton />
  if (isError || !wallet) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-8">
        <p className="text-center text-slate-500">Impossible de charger le wallet</p>
      </div>
    )
  }

  const records = wallet.records ?? []

  function handlePayout() {
    payoutMutation.mutate(undefined, {
      onSuccess: (result) => setPayoutMessage(result.message),
    })
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
          <Wallet className="h-5 w-5 text-rose-600 dark:text-rose-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Wallet (Test)</h1>
          <p className="text-xs text-slate-500 dark:text-slate-400">Page de test — Stripe Connect Custom</p>
        </div>
      </div>

      {/* Stripe Account Status */}
      <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <h2 className="text-sm font-medium uppercase tracking-wide text-slate-400 mb-3">Compte Stripe Connect</h2>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            {wallet.stripe_account_id ? (
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            ) : (
              <XCircle className="h-4 w-4 text-red-500" />
            )}
            <span className="text-sm text-slate-700 dark:text-slate-300">
              {wallet.stripe_account_id || "Non créé"}
            </span>
          </div>
          <StatusBadge label="Charges" enabled={wallet.charges_enabled} />
          <StatusBadge label="Payouts" enabled={wallet.payouts_enabled} />
        </div>
      </div>

      {/* Balance Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <BalanceCard
          icon={ShieldCheck}
          label="En escrow"
          amount={wallet.escrow_amount}
          description="Missions payées, en attente de complétion ou transfert"
          color="amber"
        />
        <BalanceCard
          icon={DollarSign}
          label="Disponible"
          amount={wallet.available_amount}
          description="Prêt à être transféré sur votre compte bancaire"
          color="green"
        />
        <BalanceCard
          icon={Send}
          label="Transféré"
          amount={wallet.transferred_amount}
          description="Total versé sur votre compte bancaire"
          color="blue"
        />
      </div>

      {/* Payout Button */}
      <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-sm font-semibold text-slate-900 dark:text-white">Retrait</h2>
            <p className="text-xs text-slate-500">Transférer les fonds disponibles vers votre compte bancaire</p>
          </div>
          <button
            type="button"
            onClick={handlePayout}
            disabled={payoutMutation.isPending || wallet.available_amount === 0}
            className={cn(
              "flex items-center gap-2 rounded-xl px-5 py-2.5",
              "text-sm font-semibold text-white transition-all duration-200",
              "gradient-primary hover:shadow-glow active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {payoutMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <ArrowDownToLine className="h-4 w-4" />
            )}
            Retirer {formatEur(wallet.available_amount)}
          </button>
        </div>
        {payoutMessage && (
          <p className="mt-3 text-sm text-green-600 dark:text-green-400">{payoutMessage}</p>
        )}
        {payoutMutation.isError && (
          <p className="mt-3 text-sm text-red-500">Erreur lors du retrait</p>
        )}
      </div>

      {/* Payment Records */}
      <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
        <div className="p-5 border-b border-slate-100 dark:border-slate-700">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white">Historique des paiements</h2>
        </div>
        {records.length === 0 ? (
          <div className="p-8 text-center text-sm text-slate-500">Aucun paiement</div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {records.map((record) => (
              <RecordRow key={record.proposal_id} record={record} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function StatusBadge({ label, enabled }: { label: string; enabled: boolean }) {
  return (
    <span className={cn(
      "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium",
      enabled
        ? "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400"
        : "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400"
    )}>
      {enabled ? <CheckCircle2 className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
      {label}
    </span>
  )
}

function BalanceCard({ icon: Icon, label, amount, description, color }: {
  icon: React.ElementType
  label: string
  amount: number
  description: string
  color: "amber" | "green" | "blue"
}) {
  const colorMap = {
    amber: { bg: "bg-amber-50 dark:bg-amber-500/10", icon: "text-amber-600 dark:text-amber-400" },
    green: { bg: "bg-green-50 dark:bg-green-500/10", icon: "text-green-600 dark:text-green-400" },
    blue: { bg: "bg-blue-50 dark:bg-blue-500/10", icon: "text-blue-600 dark:text-blue-400" },
  }
  const c = colorMap[color]

  return (
    <div className="rounded-xl border border-slate-100 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <div className="flex items-center gap-3 mb-2">
        <div className={cn("flex h-8 w-8 items-center justify-center rounded-lg", c.bg)}>
          <Icon className={cn("h-4 w-4", c.icon)} />
        </div>
        <span className="text-xs text-slate-500 dark:text-slate-400">{label}</span>
      </div>
      <p className="text-xl font-bold text-slate-900 dark:text-white">{formatEur(amount)}</p>
      <p className="mt-1 text-xs text-slate-400">{description}</p>
    </div>
  )
}

function RecordRow({ record }: { record: WalletRecord }) {
  return (
    <div className="flex items-center gap-4 px-5 py-3">
      <PaymentStatusIcon status={record.payment_status} />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
          Proposition {record.proposal_id.slice(0, 8)}...
        </p>
        <p className="text-xs text-slate-500">
          {new Date(record.created_at).toLocaleDateString("fr-FR")}
        </p>
      </div>
      <div className="text-right">
        <p className="text-sm font-semibold text-slate-900 dark:text-white">
          {formatEur(record.provider_payout)}
        </p>
        <p className="text-xs text-slate-500">
          -{formatEur(record.platform_fee)} commission
        </p>
      </div>
      <TransferBadge status={record.transfer_status} />
    </div>
  )
}

function PaymentStatusIcon({ status }: { status: string }) {
  if (status === "succeeded") return <CheckCircle2 className="h-5 w-5 text-green-500 shrink-0" />
  if (status === "failed") return <XCircle className="h-5 w-5 text-red-500 shrink-0" />
  return <Clock className="h-5 w-5 text-amber-500 shrink-0" />
}

function TransferBadge({ status }: { status: string }) {
  if (status === "completed") {
    return <span className="shrink-0 rounded-full bg-green-50 px-2.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-500/10 dark:text-green-400">Transféré</span>
  }
  if (status === "failed") {
    return <span className="shrink-0 rounded-full bg-red-50 px-2.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-500/10 dark:text-red-400">Échec</span>
  }
  return <span className="shrink-0 rounded-full bg-amber-50 px-2.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-400">En attente</span>
}

function WalletSkeleton() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 space-y-6">
      <div className="h-8 w-48 animate-shimmer rounded bg-slate-200 dark:bg-slate-700" />
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-28 animate-shimmer rounded-xl bg-slate-200 dark:bg-slate-700" />
        ))}
      </div>
    </div>
  )
}
