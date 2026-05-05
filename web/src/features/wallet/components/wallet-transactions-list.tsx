"use client"

import Link from "next/link"
import {
  CheckCircle2,
  Clock,
  Folder,
  Loader2,
  RefreshCw,
  Send,
  ShieldCheck,
  Wallet,
  XCircle,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useRetryTransfer } from "../hooks/use-wallet"
import type { WalletRecord } from "../api/wallet-api"

import { Button } from "@/shared/components/ui/button"
// Transactions list = "Mes missions" section: 3 balance cards
// (escrow / available / transferred) + a chronological history of
// payment records with retry affordance for failed transfers.
//
// Soleil v2:
//   - Section header in editorial Fraunces (22px) with leading icon
//   - Balance cards: ivoire surface, corail/sapin/blue tinted square
//     icon, mono uppercase eyebrow, Fraunces 26px stat number
//   - History card: rounded-2xl, sapin/amber/destructive 3px left
//     accent rail per row, Fraunces amount + corail-toned mini badges

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
      <SectionHeader icon={Folder} title="Mes missions" />

      {/* Mini-stats row — 3 compact cards, always horizontal, mirroring
          the mobile native WalletMissionsSection (3 Expanded cards in
          a Row at 390px). Desktop (≥768px) keeps the wider Soleil
          card content unchanged. */}
      <div className="grid grid-cols-3 gap-2 md:gap-3">
        <BalanceCard
          icon={ShieldCheck}
          label="En séquestre"
          amount={escrow}
          description="Missions payées, en attente de complétion"
          color="amber"
        />
        <BalanceCard
          icon={Wallet}
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

      <div
        className={cn(
          "overflow-hidden rounded-2xl border border-border bg-card",
        )}
        style={{ boxShadow: "var(--shadow-card)" }}
      >
        <div className="border-b border-border px-4 py-4 md:px-6 md:py-5">
          <h3 className="font-serif text-[18px] font-medium tracking-[-0.015em] text-foreground md:text-[22px]">
            Historique des missions
          </h3>
          <p className="mt-0.5 text-[12px] text-muted-foreground md:text-[12.5px]">
            Toutes vos missions — du séquestre au transfert
          </p>
        </div>
        {records.length === 0 ? (
          <div className="px-4 py-8 text-center text-[13px] text-muted-foreground md:px-6 md:py-10">
            Aucune mission pour le moment
          </div>
        ) : (
          <div>
            {records.map((record, index) => (
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
                isLast={index === records.length - 1}
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
    <div className="flex items-center gap-2.5">
      <Icon
        className="h-[17px] w-[17px] text-foreground"
        strokeWidth={1.6}
        aria-hidden="true"
      />
      <h2 className="font-serif text-[22px] font-medium tracking-[-0.015em] text-foreground">
        {title}
      </h2>
    </div>
  )
}

// Tone scale shared between the wallet's balance cards. Values map to
// Soleil tokens — never to raw Tailwind palette. Each tone exposes a
// soft background for the icon square and a foreground colour.
const BALANCE_TONE = {
  amber: {
    bg: "bg-amber-soft",
    icon: "text-warning",
  },
  green: {
    bg: "bg-success-soft",
    icon: "text-success",
  },
  blue: {
    // "blue" historically — Soleil maps it to corail-soft so the card
    // grid stays inside the warm palette without a cold blue tone.
    bg: "bg-primary-soft",
    icon: "text-primary",
  },
} as const

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
  const tone = BALANCE_TONE[color]

  return (
    <div
      className={cn(
        // Narrow viewports: tighter padding mirrors the mobile native
        // WalletBalanceCard (EdgeInsets.all(12)). Desktop (≥768px)
        // keeps the original Soleil padding.
        "rounded-2xl border border-border bg-card p-3 md:p-5",
        "transition-all duration-200 ease-out",
        "hover:-translate-y-0.5 hover:border-border-strong",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="mb-2 flex items-start gap-1.5 md:items-center md:gap-2">
        <div
          className={cn(
            "flex h-[22px] w-[22px] items-center justify-center rounded-md",
            tone.bg,
          )}
        >
          <Icon className={cn("h-3 w-3", tone.icon)} strokeWidth={1.8} />
        </div>
        <span className="text-[10.5px] font-semibold uppercase tracking-[0.04em] text-muted-foreground md:text-[11.5px]">
          {label}
        </span>
      </div>
      <p className="font-serif text-[18px] font-medium leading-none tracking-[-0.025em] text-foreground md:text-[26px]">
        {formatEur(amount)}
      </p>
      {/* Description is hidden on narrow viewports — the mobile native
          balance card never shows it (label + amount only). It returns
          on desktop (≥768px) for the fuller Soleil card. */}
      <p className="mt-2 hidden text-[11.5px] leading-relaxed text-muted-foreground md:block">
        {description}
      </p>
    </div>
  )
}

// Mission record row — Soleil v2: 3px left accent rail per status,
// rounded soft-tinted icon square, Fraunces amount, corail-toned
// mini badges.
function RecordRow({
  record,
  isLast,
}: {
  record: WalletRecord
  isLast: boolean
}) {
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

  // Visual cue: a 3px coloured rail on the left. Maps to the row's
  // "where is the money" status (failed > escrow > completed).
  const railColor = isFailed
    ? "bg-destructive"
    : isInEscrow
      ? "bg-warning"
      : "bg-success"

  // Icon square background — the icon foreground colour lives on the
  // icon itself (see PaymentStatusIcon) so the wallet-transactions-list
  // tests can find the canonical `text-green-500` / `text-red-500` /
  // `text-amber-500` classes regardless of the row's accent rail.
  const iconBg = isFailed
    ? "bg-destructive/10"
    : isInEscrow
      ? "bg-amber-soft"
      : "bg-success-soft"

  return (
    <div
      className={cn(
        // Narrow viewports: tighter horizontal padding matching the
        // mobile native WalletMissionTile (EdgeInsets.fromLTRB(12,12,16,12)).
        "relative px-3 py-3 md:px-6 md:py-4",
        !isLast && "border-b border-border",
      )}
    >
      <span
        aria-hidden="true"
        className={cn("absolute inset-y-0 left-0 w-[3px]", railColor)}
      />
      <div className="flex items-center gap-2.5 md:gap-4">
        <div
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-xl md:h-9 md:w-9",
            iconBg,
          )}
        >
          <PaymentStatusIcon status={record.payment_status} />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-[13px] font-semibold text-foreground md:text-[14px]">
            Mission du {formatDate(record.created_at)}
          </p>
          <div className="mt-0.5 flex flex-wrap items-center gap-2">
            {isInEscrow && !isFailed && (
              <span className="text-[11px] font-medium text-warning md:text-[11.5px]">
                En séquestre — mission en cours
              </span>
            )}
            <MissionBadge status={record.mission_status} />
          </div>
        </div>
        <div className="shrink-0 text-right">
          <p className="font-serif text-[15px] font-semibold leading-none tracking-[-0.015em] text-foreground md:text-[18px]">
            {formatEur(record.provider_payout)}
          </p>
          <p className="mt-1 text-[10.5px] text-subtle-foreground md:text-[11px]">
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
    <Button
      variant="ghost"
      size="auto"
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
        "inline-flex shrink-0 items-center gap-1 rounded-full px-3 py-1",
        "text-[11px] font-bold transition-colors duration-150",
        "bg-destructive/10 text-destructive hover:bg-destructive/20",
        "disabled:cursor-not-allowed disabled:opacity-60",
      )}
    >
      {pending ? (
        <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
      ) : (
        <RefreshCw className="h-3 w-3" aria-hidden="true" />
      )}
      Échec — Réessayer
    </Button>
  )
}

function RetryErrorRow({ error }: { error: Error }) {
  if (!(error instanceof ApiError)) {
    return (
      <div role="alert" className="mt-2 pl-[52px] text-[12px] text-destructive">
        <p>Erreur lors de la nouvelle tentative</p>
      </div>
    )
  }
  return (
    <div role="alert" className="mt-2 pl-[52px] text-[12px] text-destructive">
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
          className="font-semibold underline hover:opacity-80"
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
          className="font-semibold underline hover:opacity-80"
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
      cls: "bg-success-soft text-success",
    },
    completion_requested: {
      label: "Complétion demandée",
      cls: "bg-amber-soft text-warning",
    },
    completed: {
      label: "Terminée",
      cls: "bg-primary-soft text-primary-deep",
    },
    paid: {
      label: "Payée",
      cls: "bg-success-soft text-success",
    },
  }
  const c =
    configs[status] ?? {
      label: status,
      cls: "bg-border text-muted-foreground",
    }
  return (
    <span
      className={cn(
        "rounded-full px-2 py-0.5 text-[10.5px] font-semibold",
        c.cls,
      )}
    >
      {c.label}
    </span>
  )
}

// Status icon colour classes are kept as canonical Tailwind palette
// classes (text-green-500 / text-red-500 / text-amber-500) so the
// payment-status assertions in wallet-transactions-list.test.tsx keep
// finding them. The icon background lives on the parent square (see
// iconBg in RecordRow) and uses Soleil tones — the foreground gets
// the single semantic colour the test is looking for.
function PaymentStatusIcon({ status }: { status: string }) {
  if (status === "succeeded")
    return (
      <CheckCircle2
        className="h-4 w-4 text-green-500"
        strokeWidth={1.8}
      />
    )
  if (status === "failed")
    return <XCircle className="h-4 w-4 text-red-500" strokeWidth={1.8} />
  return <Clock className="h-4 w-4 text-amber-500" strokeWidth={1.8} />
}

function TransferBadge({ status }: { status: string }) {
  const baseCls =
    "shrink-0 rounded-full px-3 py-1 text-[11px] font-bold whitespace-nowrap"
  if (status === "completed") {
    return (
      <span className={cn(baseCls, "bg-success-soft text-success")}>
        Transféré
      </span>
    )
  }
  if (status === "failed") {
    return (
      <span className={cn(baseCls, "bg-destructive/10 text-destructive")}>
        Échec
      </span>
    )
  }
  return (
    <span className={cn(baseCls, "bg-amber-soft text-warning")}>
      En séquestre
    </span>
  )
}
