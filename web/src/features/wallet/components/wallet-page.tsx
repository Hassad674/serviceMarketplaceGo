"use client"

import { useState } from "react"
import Link from "next/link"
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
  RefreshCw,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useWallet, useRequestPayout, useRetryTransfer } from "../hooks/use-wallet"
import type {
  WalletRecord,
  CommissionWallet,
  WalletCommissionRecord,
} from "../api/wallet-api"
import { Sparkles, TrendingUp, UserCheck, Undo2 } from "lucide-react"

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}

export function WalletPage() {
  const { data: wallet, isLoading, isError } = useWallet()
  const payoutMutation = useRequestPayout()
  const canWithdraw = useHasPermission("wallet.withdraw")
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
            disabled={payoutMutation.isPending || wallet.available_amount === 0 || !canWithdraw}
            title={!canWithdraw ? "Seul le propri\u00e9taire peut demander un retrait" : undefined}
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
          <div className="mt-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-500/20 dark:bg-red-500/10">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
            {payoutMutation.error instanceof ApiError && payoutMutation.error.code === "stripe_account_missing" ? (
              <p className="text-sm text-red-700 dark:text-red-400">
                Vous devez d&apos;abord configurer vos informations de paiement pour effectuer un retrait.{" "}
                <Link href="/payment-info" className="font-semibold underline hover:text-red-900 dark:hover:text-red-300">
                  Configurer maintenant
                </Link>
              </p>
            ) : (
              <p className="text-sm text-red-700 dark:text-red-400">Erreur lors du retrait</p>
            )}
          </div>
        )}
      </div>

      {/* Apporteur commissions — rendered only when the viewer has
          commission activity. Apporteurs that are also providers see
          BOTH this section and the provider-side section below. */}
      <CommissionSection
        summary={wallet.commissions}
        records={wallet.commission_records ?? []}
      />

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
  const retryMutation = useRetryTransfer()
  const canWithdraw = useHasPermission("wallet.withdraw")

  const isFailed = record.transfer_status === "failed"
  const pendingForThisRow =
    retryMutation.isPending && retryMutation.variables === record.proposal_id
  const errorForThisRow =
    retryMutation.isError && retryMutation.variables === record.proposal_id
      ? retryMutation.error
      : null

  function handleRetry() {
    retryMutation.mutate(record.proposal_id)
  }

  return (
    <div className="px-5 py-3">
      <div className="flex items-center gap-4">
        <PaymentStatusIcon status={record.payment_status} />
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
            Proposition {record.proposal_id.slice(0, 8)}...
          </p>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-xs text-slate-500">
              {new Date(record.created_at).toLocaleDateString("fr-FR")}
            </span>
            <MissionBadge status={record.mission_status} />
          </div>
        </div>
        <div className="text-right">
          <p className="text-sm font-semibold text-slate-900 dark:text-white">
            {formatEur(record.provider_payout)}
          </p>
          <p className="text-xs text-slate-500">
            -{formatEur(record.platform_fee)} commission
          </p>
        </div>
        {isFailed ? (
          <button
            type="button"
            onClick={handleRetry}
            disabled={pendingForThisRow || !canWithdraw}
            title={
              !canWithdraw
                ? "Seul le propriétaire peut relancer un transfert"
                : "Relancer le transfert"
            }
            aria-label="Relancer le transfert"
            className={cn(
              "shrink-0 inline-flex items-center gap-1 rounded-full px-2.5 py-0.5",
              "text-xs font-medium transition-colors duration-200",
              "bg-red-50 text-red-700 hover:bg-red-100",
              "dark:bg-red-500/10 dark:text-red-400 dark:hover:bg-red-500/20",
              "disabled:opacity-60 disabled:cursor-not-allowed",
            )}
          >
            {pendingForThisRow ? (
              <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
            ) : (
              <RefreshCw className="h-3 w-3" aria-hidden="true" />
            )}
            Échec — Réessayer
          </button>
        ) : (
          <TransferBadge status={record.transfer_status} />
        )}
      </div>
      {errorForThisRow && (
        <p
          role="alert"
          className="mt-2 pl-9 text-xs text-red-700 dark:text-red-400"
        >
          {errorForThisRow instanceof ApiError &&
          errorForThisRow.code === "transfer_not_retriable"
            ? "Ce transfert ne peut plus être relancé — la mission doit être terminée et le précédent transfert en échec."
            : errorForThisRow instanceof ApiError &&
                errorForThisRow.code === "stripe_account_missing"
              ? "Complétez votre configuration Stripe avant de relancer ce transfert."
              : "Erreur lors de la nouvelle tentative"}
        </p>
      )}
    </div>
  )
}

function MissionBadge({ status }: { status: string }) {
  if (!status) return null
  const configs: Record<string, { label: string; cls: string }> = {
    active: { label: "En cours", cls: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400" },
    completion_requested: { label: "Complétion demandée", cls: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400" },
    completed: { label: "Terminée", cls: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400" },
    paid: { label: "Payée", cls: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400" },
  }
  const c = configs[status] ?? { label: status, cls: "bg-slate-50 text-slate-700" }
  return <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", c.cls)}>{c.label}</span>
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

// CommissionSection renders the apporteur's commission wallet — three
// summary cards + a history list. Hidden when the viewer has no
// commission activity so provider-only users keep the same compact UI.
function CommissionSection({
  summary,
  records,
}: {
  summary: CommissionWallet
  records: WalletCommissionRecord[]
}) {
  const totalActivity =
    summary.pending_cents +
    summary.pending_kyc_cents +
    summary.paid_cents +
    summary.clawed_back_cents
  if (totalActivity === 0 && records.length === 0) {
    return null
  }
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Sparkles className="h-4 w-4 text-rose-500" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
          Commissions d&apos;apport
        </h2>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <BalanceCard
          icon={Clock}
          label="En attente"
          amount={summary.pending_cents + summary.pending_kyc_cents}
          description={
            summary.pending_kyc_cents > 0
              ? "Dont KYC à compléter — complétez votre KYC pour les recevoir"
              : "Commissions en file d'attente de virement"
          }
          color="amber"
        />
        <BalanceCard
          icon={UserCheck}
          label="Reçues"
          amount={summary.paid_cents}
          description="Déjà versées sur votre compte Stripe Connect"
          color="green"
        />
        <BalanceCard
          icon={Undo2}
          label="Reprises"
          amount={summary.clawed_back_cents}
          description="Reversées suite à un remboursement client"
          color="blue"
        />
      </div>

      <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
        <div className="p-5 border-b border-slate-100 dark:border-slate-700 flex items-center gap-2">
          <TrendingUp className="h-4 w-4 text-rose-500" aria-hidden="true" />
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            Historique des commissions
          </h3>
        </div>
        {records.length === 0 ? (
          <div className="p-6 text-center text-xs text-slate-500">
            Aucune commission pour le moment.
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {records.map((r) => (
              <CommissionRow key={r.id} record={r} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function CommissionRow({ record }: { record: WalletCommissionRecord }) {
  const status = commissionStatusLabel(record.status)
  return (
    <div className="flex items-center gap-4 px-5 py-3">
      <CommissionStatusIcon status={record.status} />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
          Milestone {record.milestone_id ? record.milestone_id.slice(0, 8) + "…" : "—"}
        </p>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-xs text-slate-500">
            {new Date(record.created_at).toLocaleDateString("fr-FR")}
          </span>
          {record.referral_id && (
            <Link
              href={`/referrals/${record.referral_id}`}
              className="text-[11px] font-medium text-rose-600 hover:underline"
            >
              Voir la mise en relation
            </Link>
          )}
        </div>
      </div>
      <div className="text-right">
        <p className="text-sm font-semibold text-slate-900 dark:text-white">
          {formatEur(record.commission_cents)}
        </p>
        <p className="text-xs text-slate-500">
          sur {formatEur(record.gross_amount_cents)}
        </p>
      </div>
      <span className={cn("shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium", status.cls)}>
        {status.label}
      </span>
    </div>
  )
}

function commissionStatusLabel(status: string): { label: string; cls: string } {
  switch (status) {
    case "paid":
      return { label: "Reçue", cls: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400" }
    case "pending":
      return { label: "En attente", cls: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400" }
    case "pending_kyc":
      return { label: "KYC requis", cls: "bg-orange-50 text-orange-700 dark:bg-orange-500/10 dark:text-orange-400" }
    case "clawed_back":
      return { label: "Reprise", cls: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400" }
    case "failed":
      return { label: "Échec", cls: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400" }
    case "cancelled":
      return { label: "Annulée", cls: "bg-slate-100 text-slate-600" }
    default:
      return { label: status, cls: "bg-slate-100 text-slate-600" }
  }
}

function CommissionStatusIcon({ status }: { status: string }) {
  if (status === "paid") return <CheckCircle2 className="h-5 w-5 text-green-500 shrink-0" />
  if (status === "failed") return <XCircle className="h-5 w-5 text-red-500 shrink-0" />
  if (status === "clawed_back") return <Undo2 className="h-5 w-5 text-blue-500 shrink-0" />
  return <Clock className="h-5 w-5 text-amber-500 shrink-0" />
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
