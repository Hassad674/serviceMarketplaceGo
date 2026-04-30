"use client"

import Link from "next/link"
import {
  Briefcase,
  CheckCircle2,
  Clock,
  DollarSign,
  Loader2,
  RefreshCw,
  Send,
  ShieldCheck,
  XCircle,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useRetryTransfer } from "../hooks/use-wallet"
import type { WalletRecord } from "../api/wallet-api"

// Transactions list = "Mes missions" section: 3 balance cards
// (escrow / available / transferred) + a chronological history of
// payment records with retry affordance for failed transfers.

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", {
    style: "currency",
    currency: "EUR",
  }).format(centimes / 100)
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("fr-FR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  })
}

export interface WalletTransactionsListProps {
  escrow: number
  available: number
  transferred: number
  records: WalletRecord[]
}

export function WalletTransactionsList({
  escrow,
  available,
  transferred,
  records,
}: WalletTransactionsListProps) {
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
              // Defensive key: id is the canonical source, with milestone_id /
              // composite as fallbacks so a future regression can't trigger a
              // duplicate-key warning.
              <RecordRow
                key={
                  record.id ||
                  record.milestone_id ||
                  `${record.proposal_id}-${record.created_at}`
                }
                record={record}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

export function SectionHeader({
  icon: Icon,
  title,
}: {
  icon: React.ElementType
  title: string
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="h-4 w-4 text-rose-500" aria-hidden="true" />
      <h2 className="text-base font-semibold text-slate-900 dark:text-white">
        {title}
      </h2>
    </div>
  )
}

export function BalanceCard({
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
    amber: {
      bg: "bg-amber-50 dark:bg-amber-500/10",
      icon: "text-amber-600 dark:text-amber-400",
    },
    green: {
      bg: "bg-green-50 dark:bg-green-500/10",
      icon: "text-green-600 dark:text-green-400",
    },
    blue: {
      bg: "bg-blue-50 dark:bg-blue-500/10",
      icon: "text-blue-600 dark:text-blue-400",
    },
  }
  const c = colorMap[color]

  return (
    <div className="rounded-xl border border-slate-100 bg-white p-4 shadow-sm transition-all duration-200 hover:shadow-md dark:border-slate-700 dark:bg-slate-800">
      <div className="mb-2 flex items-center gap-3">
        <div
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-lg",
            c.bg,
          )}
        >
          <Icon className={cn("h-4 w-4", c.icon)} />
        </div>
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {label}
        </span>
      </div>
      <p className="font-mono text-xl font-bold text-slate-900 dark:text-white">
        {formatEur(amount)}
      </p>
      <p className="mt-1 text-xs text-slate-400">{description}</p>
    </div>
  )
}

// Mission record row — distinct visual treatment for escrow rows
// (pending transfer) and failed rows (retry affordance).
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
            -{formatEur(record.platform_fee)} Frais plateforme
          </p>
        </div>
        {isFailed ? (
          <RetryButton
            disabled={pendingForThisRow || !canWithdraw}
            pending={pendingForThisRow}
            canWithdraw={canWithdraw}
            onRetry={handleRetry}
          />
        ) : (
          <TransferBadge status={record.transfer_status} />
        )}
      </div>
      {errorForThisRow && <RetryErrorRow error={errorForThisRow} />}
    </div>
  )
}

function RetryButton({
  disabled,
  pending,
  canWithdraw,
  onRetry,
}: {
  disabled: boolean
  pending: boolean
  canWithdraw: boolean
  onRetry: () => void
}) {
  return (
    <button
      type="button"
      onClick={onRetry}
      disabled={disabled}
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
      {pending ? (
        <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
      ) : (
        <RefreshCw className="h-3 w-3" aria-hidden="true" />
      )}
      Échec — Réessayer
    </button>
  )
}

function RetryErrorRow({ error }: { error: Error }) {
  if (!(error instanceof ApiError)) {
    return (
      <div role="alert" className="mt-2 pl-9 text-xs text-red-700 dark:text-red-400">
        <p>Erreur lors de la nouvelle tentative</p>
      </div>
    )
  }
  return (
    <div role="alert" className="mt-2 pl-9 text-xs text-red-700 dark:text-red-400">
      <RetryErrorMessage code={error.code} />
    </div>
  )
}

function RetryErrorMessage({ code }: { code: string }) {
  if (code === "provider_kyc_incomplete") {
    // 412: account exists but payouts_enabled=false. Distinct copy
    // from "no account" because the user already started onboarding
    // — they need to FINISH it, not start over.
    return (
      <p>
        Termine ton onboarding Stripe (page{" "}
        <Link
          href="/payment-info"
          className="font-semibold underline hover:text-red-900 dark:hover:text-red-300"
        >
          Infos paiement
        </Link>
        ) pour pouvoir recevoir le virement.
      </p>
    )
  }
  if (code === "transfer_not_retriable") {
    return (
      <p>
        Ce transfert ne peut plus être relancé — la mission doit être
        terminée et le précédent transfert en échec.
      </p>
    )
  }
  if (code === "stripe_account_missing") {
    return (
      <p>
        Configure d&apos;abord tes informations de paiement avant de
        relancer ce transfert.{" "}
        <Link
          href="/payment-info"
          className="font-semibold underline hover:text-red-900 dark:hover:text-red-300"
        >
          Configurer
        </Link>
      </p>
    )
  }
  if (code === "retry_failed") {
    // 502: upstream Stripe error. Tell the user to retry later
    // instead of treating it as terminal — most of these clear
    // up on their own within minutes.
    return (
      <p>
        Le virement a de nouveau échoué côté Stripe. Réessaie dans
        quelques minutes ou contacte le support.
      </p>
    )
  }
  if (code === "payment_record_not_found") {
    return <p>Ce transfert est introuvable.</p>
  }
  return <p>Erreur lors de la nouvelle tentative</p>
}

function MissionBadge({ status }: { status: string }) {
  if (!status) return null
  const configs: Record<string, { label: string; cls: string }> = {
    active: {
      label: "En cours",
      cls: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
    },
    completion_requested: {
      label: "Complétion demandée",
      cls: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
    },
    completed: {
      label: "Terminée",
      cls: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
    },
    paid: {
      label: "Payée",
      cls: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400",
    },
  }
  const c = configs[status] ?? { label: status, cls: "bg-slate-50 text-slate-700" }
  return (
    <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", c.cls)}>
      {c.label}
    </span>
  )
}

function PaymentStatusIcon({ status }: { status: string }) {
  if (status === "succeeded")
    return <CheckCircle2 className="h-5 w-5 shrink-0 text-green-500" />
  if (status === "failed")
    return <XCircle className="h-5 w-5 shrink-0 text-red-500" />
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
