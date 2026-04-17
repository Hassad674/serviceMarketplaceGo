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
  Briefcase,
  Sparkles,
  ChevronRight,
  UserCheck,
  Undo2,
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

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("fr-FR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  })
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
  const commissionRecords = wallet.commission_records ?? []
  const totalEarned = wallet.transferred_amount + wallet.commissions.paid_cents

  function handlePayout() {
    payoutMutation.mutate(undefined, {
      onSuccess: (result) => setPayoutMessage(result.message),
    })
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 space-y-8">
      {/* Hero — total earned + Stripe status + withdraw CTA */}
      <WalletHero
        wallet={wallet}
        totalEarned={totalEarned}
        canWithdraw={canWithdraw}
        payoutPending={payoutMutation.isPending}
        payoutMessage={payoutMessage}
        payoutError={payoutMutation.isError ? (payoutMutation.error as Error) : null}
        onPayout={handlePayout}
      />

      {/* Missions — escrow / available / transferred + history */}
      <MissionsSection
        escrow={wallet.escrow_amount}
        available={wallet.available_amount}
        transferred={wallet.transferred_amount}
        records={records}
      />

      {/* Commissions — apporteur section, hidden when no activity */}
      <CommissionSection
        summary={wallet.commissions}
        records={commissionRecords}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Hero: title + total earned + compact Stripe status + withdraw CTA
// ---------------------------------------------------------------------------
function WalletHero({
  wallet,
  totalEarned,
  canWithdraw,
  payoutPending,
  payoutMessage,
  payoutError,
  onPayout,
}: {
  wallet: {
    stripe_account_id: string
    charges_enabled: boolean
    payouts_enabled: boolean
    available_amount: number
  }
  totalEarned: number
  canWithdraw: boolean
  payoutPending: boolean
  payoutMessage: string
  payoutError: Error | null
  onPayout: () => void
}) {
  const hasAccount = Boolean(wallet.stripe_account_id)
  const canClick = canWithdraw && wallet.available_amount > 0

  return (
    <section className="rounded-2xl border border-slate-100 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
            <Wallet className="h-5 w-5 text-rose-600 dark:text-rose-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Portefeuille</h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Vos revenus issus des missions et commissions d&apos;apport
            </p>
          </div>
        </div>
      </div>

      {/* Total earned — hero number */}
      <div className="mt-6">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">
          Revenus totaux
        </p>
        <p className="mt-1 font-mono text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">
          {formatEur(totalEarned)}
        </p>
      </div>

      {/* Stripe status line + withdraw CTA row */}
      <div className="mt-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2 text-sm">
          {hasAccount && wallet.payouts_enabled ? (
            <>
              <CheckCircle2 className="h-4 w-4 text-green-500" aria-hidden="true" />
              <span className="text-slate-700 dark:text-slate-300">
                Compte Stripe prêt — virements activés
              </span>
            </>
          ) : hasAccount ? (
            <>
              <AlertTriangle className="h-4 w-4 text-amber-500" aria-hidden="true" />
              <span className="text-slate-700 dark:text-slate-300">
                Compte Stripe en cours de vérification
              </span>
              <Link
                href="/payment-info"
                className="text-rose-600 hover:underline dark:text-rose-400"
              >
                Finaliser
              </Link>
            </>
          ) : (
            <>
              <XCircle className="h-4 w-4 text-red-500" aria-hidden="true" />
              <span className="text-slate-700 dark:text-slate-300">
                Compte Stripe non configuré
              </span>
              <Link
                href="/payment-info"
                className="text-rose-600 hover:underline dark:text-rose-400"
              >
                Configurer
              </Link>
            </>
          )}
        </div>

        <div className="flex flex-col items-stretch gap-1 sm:items-end">
          <button
            type="button"
            onClick={onPayout}
            disabled={payoutPending || !canClick}
            title={!canWithdraw ? "Seul le propriétaire peut demander un retrait" : undefined}
            className={cn(
              "inline-flex items-center justify-center gap-2 rounded-xl px-5 py-2.5",
              "text-sm font-semibold text-white transition-all duration-200",
              "gradient-primary hover:shadow-glow active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {payoutPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <ArrowDownToLine className="h-4 w-4" />
            )}
            Retirer {formatEur(wallet.available_amount)}
          </button>
          {wallet.available_amount === 0 && (
            <span className="text-xs text-slate-400 dark:text-slate-500 sm:text-right">
              Aucun fonds disponible
            </span>
          )}
        </div>
      </div>

      {payoutMessage && (
        <p className="mt-3 text-sm text-green-600 dark:text-green-400">{payoutMessage}</p>
      )}
      {payoutError && (
        <div className="mt-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-500/20 dark:bg-red-500/10">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
          {payoutError instanceof ApiError && payoutError.code === "stripe_account_missing" ? (
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
    </section>
  )
}

// ---------------------------------------------------------------------------
// Missions section — 3 balance cards + history
// ---------------------------------------------------------------------------
function MissionsSection({
  escrow,
  available,
  transferred,
  records,
}: {
  escrow: number
  available: number
  transferred: number
  records: WalletRecord[]
}) {
  return (
    <section className="space-y-4">
      <SectionHeader icon={Briefcase} title="Mes missions" />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <BalanceCard
          icon={ShieldCheck}
          label="En séquestre"
          amount={escrow}
          description="Missions payées, en attente de complétion"
          color="amber"
        />
        <BalanceCard
          icon={DollarSign}
          label="Disponible"
          amount={available}
          description="Prêt à être retiré sur votre compte bancaire"
          color="green"
        />
        <BalanceCard
          icon={Send}
          label="Transféré"
          amount={transferred}
          description="Total déjà versé sur votre compte bancaire"
          color="blue"
        />
      </div>

      <div className="overflow-hidden rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-700">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            Historique des missions
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Toutes vos missions — du séquestre au transfert
          </p>
        </div>
        {records.length === 0 ? (
          <div className="p-8 text-center text-sm text-slate-500">
            Aucune mission pour le moment
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {records.map((record) => (
              // Defensive key: id is the canonical source (payment_record PK,
              // added to the backend DTO so every row is unique), but if an
              // older backend omits it we fall back to milestone_id (also
              // unique per proposal) and then to a composite of proposal_id
              // and created_at. Keeping the fallback guards against future
              // regressions — React's duplicate-key warning costs us nothing
              // here and the triple-lookup is O(1).
              <RecordRow
                key={record.id || record.milestone_id || `${record.proposal_id}-${record.created_at}`}
                record={record}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function SectionHeader({ icon: Icon, title }: { icon: React.ElementType; title: string }) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="h-4 w-4 text-rose-500" aria-hidden="true" />
      <h2 className="text-base font-semibold text-slate-900 dark:text-white">{title}</h2>
    </div>
  )
}

function BalanceCard({
  icon: Icon,
  label,
  amount,
  description,
  color,
}: {
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
    <div className="rounded-xl border border-slate-100 bg-white p-4 shadow-sm transition-all duration-200 hover:shadow-md dark:border-slate-700 dark:bg-slate-800">
      <div className="mb-2 flex items-center gap-3">
        <div className={cn("flex h-8 w-8 items-center justify-center rounded-lg", c.bg)}>
          <Icon className={cn("h-4 w-4", c.icon)} />
        </div>
        <span className="text-xs text-slate-500 dark:text-slate-400">{label}</span>
      </div>
      <p className="font-mono text-xl font-bold text-slate-900 dark:text-white">
        {formatEur(amount)}
      </p>
      <p className="mt-1 text-xs text-slate-400">{description}</p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Mission record row — distinct visual treatment for escrow (pending transfer)
// ---------------------------------------------------------------------------
function RecordRow({ record }: { record: WalletRecord }) {
  const retryMutation = useRetryTransfer()
  const canWithdraw = useHasPermission("wallet.withdraw")

  const isFailed = record.transfer_status === "failed"
  const isInEscrow =
    record.transfer_status === "pending" || record.transfer_status === ""
  const pendingForThisRow =
    retryMutation.isPending && retryMutation.variables === record.id
  const errorForThisRow =
    retryMutation.isError && retryMutation.variables === record.id
      ? retryMutation.error
      : null

  function handleRetry() {
    retryMutation.mutate(record.id)
  }

  // Escrow rows get a muted amber left border to make "still in escrow"
  // visually obvious without relying on the status badge alone.
  const leftAccent = isFailed
    ? "border-l-4 border-l-red-400 dark:border-l-red-500/60"
    : isInEscrow
      ? "border-l-4 border-l-amber-400 dark:border-l-amber-500/60"
      : "border-l-4 border-l-transparent"

  return (
    <div className={cn("px-5 py-3", leftAccent)}>
      <div className="flex items-center gap-4">
        <PaymentStatusIcon status={record.payment_status} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-slate-900 dark:text-white">
            Mission du {formatDate(record.created_at)}
          </p>
          <div className="mt-0.5 flex flex-wrap items-center gap-2">
            {isInEscrow && !isFailed && (
              <span className="text-[11px] font-medium text-amber-700 dark:text-amber-400">
                En séquestre — mission en cours
              </span>
            )}
            <MissionBadge status={record.mission_status} />
          </div>
        </div>
        <div className="text-right">
          <p className="font-mono text-sm font-semibold text-slate-900 dark:text-white">
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
              "inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-0.5",
              "text-xs font-medium transition-colors duration-200",
              "bg-red-50 text-red-700 hover:bg-red-100",
              "dark:bg-red-500/10 dark:text-red-400 dark:hover:bg-red-500/20",
              "disabled:cursor-not-allowed disabled:opacity-60",
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
  if (status === "succeeded") return <CheckCircle2 className="h-5 w-5 shrink-0 text-green-500" />
  if (status === "failed") return <XCircle className="h-5 w-5 shrink-0 text-red-500" />
  return <Clock className="h-5 w-5 shrink-0 text-amber-500" />
}

function TransferBadge({ status }: { status: string }) {
  if (status === "completed") {
    return (
      <span className="shrink-0 rounded-full bg-green-50 px-2.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-500/10 dark:text-green-400">
        Transféré
      </span>
    )
  }
  if (status === "failed") {
    return (
      <span className="shrink-0 rounded-full bg-red-50 px-2.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-500/10 dark:text-red-400">
        Échec
      </span>
    )
  }
  return (
    <span className="shrink-0 rounded-full bg-amber-50 px-2.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-400">
      En séquestre
    </span>
  )
}

// ---------------------------------------------------------------------------
// Commissions section — apporteur d'affaires
// ---------------------------------------------------------------------------
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
    <section className="space-y-4">
      <SectionHeader icon={Sparkles} title="Mes commissions d'apport" />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
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

      <div className="overflow-hidden rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-700">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            Historique des commissions
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Chaque mise en relation que vous avez facilitée
          </p>
        </div>
        {records.length === 0 ? (
          <div className="p-6 text-center text-xs text-slate-500">
            Aucune commission pour le moment
          </div>
        ) : (
          <div className="divide-y divide-slate-100 dark:divide-slate-700">
            {records.map((r) => (
              <CommissionRow key={r.id} record={r} />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function CommissionRow({ record }: { record: WalletCommissionRecord }) {
  const status = commissionStatusLabel(record.status)
  const isPending =
    record.status === "pending" || record.status === "pending_kyc"

  const leftAccent = isPending
    ? "border-l-4 border-l-amber-400 dark:border-l-amber-500/60"
    : record.status === "clawed_back"
      ? "border-l-4 border-l-blue-400 dark:border-l-blue-500/60"
      : "border-l-4 border-l-transparent"

  const rowContent = (
    <div className={cn("flex items-center gap-4 px-5 py-3", leftAccent)}>
      <CommissionStatusIcon status={record.status} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-slate-900 dark:text-white">
          Commission du {formatDate(record.created_at)}
        </p>
        <div className="mt-0.5 flex items-center gap-2">
          <span className="text-xs text-slate-500">
            sur {formatEur(record.gross_amount_cents)} de mission
          </span>
        </div>
      </div>
      <div className="text-right">
        <p className="font-mono text-sm font-semibold text-slate-900 dark:text-white">
          {formatEur(record.commission_cents)}
        </p>
        <span className={cn("mt-0.5 inline-block rounded-full px-2 py-0.5 text-[10px] font-medium", status.cls)}>
          {status.label}
        </span>
      </div>
      {record.referral_id ? (
        <ChevronRight
          className="h-4 w-4 shrink-0 text-slate-400 transition-colors group-hover:text-rose-500"
          aria-hidden="true"
        />
      ) : (
        <div className="w-4 shrink-0" aria-hidden="true" />
      )}
    </div>
  )

  if (record.referral_id) {
    return (
      <Link
        href={`/referrals/${record.referral_id}`}
        aria-label="Voir la mise en relation"
        className="group block transition-colors hover:bg-slate-50 dark:hover:bg-slate-800"
      >
        {rowContent}
      </Link>
    )
  }
  return rowContent
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
  if (status === "paid") return <CheckCircle2 className="h-5 w-5 shrink-0 text-green-500" />
  if (status === "failed") return <XCircle className="h-5 w-5 shrink-0 text-red-500" />
  if (status === "clawed_back") return <Undo2 className="h-5 w-5 shrink-0 text-blue-500" />
  return <Clock className="h-5 w-5 shrink-0 text-amber-500" />
}

function WalletSkeleton() {
  return (
    <div className="mx-auto max-w-4xl space-y-6 px-4 py-8">
      <div className="h-32 animate-shimmer rounded-2xl bg-slate-200 dark:bg-slate-700" />
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-28 animate-shimmer rounded-xl bg-slate-200 dark:bg-slate-700" />
        ))}
      </div>
      <div className="h-64 animate-shimmer rounded-2xl bg-slate-200 dark:bg-slate-700" />
    </div>
  )
}
