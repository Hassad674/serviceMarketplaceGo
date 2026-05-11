"use client"

import Link from "next/link"
import {
  CheckCircle2,
  ChevronRight,
  Clock,
  Loader2,
  Sparkles,
  Undo2,
  UserCheck,
  XCircle,
} from "lucide-react"
import { useState } from "react"

import { ApiError } from "@/shared/lib/api-client"
import { cn } from "@/shared/lib/utils"
import {
  BalanceCard,
  SectionHeader,
} from "./wallet-transactions-list"
import { CommissionKYCRequiredModal } from "./commission-kyc-required-modal"
import { useRetryCommission } from "../hooks/use-wallet"
import type {
  CommissionWallet,
  WalletCommissionRecord,
} from "../api/wallet-api"

// Commissions list = "Mes commissions d'apport" section. Hidden when
// the provider has no apporteur activity at all (zero pending, paid,
// clawed back, AND no historical record).
//
// Soleil v2: shares the BalanceCard + SectionHeader primitives from
// wallet-transactions-list so the two sections feel like siblings.
// Row treatment: 3px coloured rail + corail-toned mini badge, same
// pattern as the missions history.

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

export interface WalletCommissionListProps {
  summary: CommissionWallet
  records: WalletCommissionRecord[]
}

export function WalletCommissionList({
  summary,
  records,
}: WalletCommissionListProps) {
  const totalActivity =
    summary.pending_cents +
    summary.pending_kyc_cents +
    summary.paid_cents +
    summary.clawed_back_cents

  // KYC modal state for the Retirer fallback (D1+D2). When the 422
  // response comes back from POST /wallet/commissions/{id}/retry, we
  // open this modal with the onboarding URL embedded in the error
  // body so the apporteur can deep-link straight to Stripe to finish
  // KYC. The URL is optional — the modal falls back to the in-app
  // /payment-info redirect when absent.
  const [kycModalOpen, setKycModalOpen] = useState(false)
  const [kycOnboardingURL, setKycOnboardingURL] = useState<string | undefined>(
    undefined,
  )
  // Track which row's Retirer button is currently in-flight so we
  // render a spinner on that specific row (instead of disabling every
  // Retirer button while one is loading).
  const [retryingID, setRetryingID] = useState<string | null>(null)
  const retryMutation = useRetryCommission()

  if (totalActivity === 0 && records.length === 0) {
    return null
  }

  function handleRetire(commissionID: string) {
    setRetryingID(commissionID)
    retryMutation.mutate(commissionID, {
      onSettled: () => setRetryingID(null),
      onError: (err: unknown) => {
        if (err instanceof ApiError && err.status === 422 && err.code === "kyc_required") {
          const url =
            (err.body as { onboarding_url?: string } | null)?.onboarding_url
          setKycOnboardingURL(typeof url === "string" && url ? url : undefined)
          setKycModalOpen(true)
        }
        // 409 / 502 → silent; the global toaster (if wired) will pick
        // up generic errors. We don't render an inline error to keep
        // the row compact — the wallet refresh after onSettled will
        // update the row to its actual state.
      },
    })
  }

  return (
    <section className="space-y-4">
      <SectionHeader icon={Sparkles} title="Mes commissions d'apport" />

      {/* Mini-stats row — 3 compact cards, always horizontal, mirroring
          the mobile native WalletCommissionsSection. */}
      <div className="grid grid-cols-3 gap-2 md:gap-3">
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

      <div
        className={cn(
          "overflow-hidden rounded-2xl border border-border bg-card",
        )}
        style={{ boxShadow: "var(--shadow-card)" }}
      >
        <div className="border-b border-border px-4 py-4 md:px-6 md:py-5">
          <h3 className="font-serif text-[18px] font-medium tracking-[-0.015em] text-foreground md:text-[22px]">
            Historique des commissions
          </h3>
          <p className="mt-0.5 text-[12px] text-muted-foreground md:text-[12.5px]">
            Chaque mise en relation que vous avez facilitée
          </p>
        </div>
        {records.length === 0 ? (
          <div className="px-4 py-6 text-center text-[12.5px] text-muted-foreground md:px-6 md:py-8">
            Aucune commission pour le moment
          </div>
        ) : (
          <div>
            {records.map((r, index) => (
              <CommissionRow
                key={r.id}
                record={r}
                isLast={index === records.length - 1}
                isRetrying={retryingID === r.id}
                onRetire={() => handleRetire(r.id)}
              />
            ))}
          </div>
        )}
      </div>

      <CommissionKYCRequiredModal
        open={kycModalOpen}
        onClose={() => setKycModalOpen(false)}
        onboardingURL={kycOnboardingURL}
      />
    </section>
  )
}

function CommissionRow({
  record,
  isLast,
  isRetrying,
  onRetire,
}: {
  record: WalletCommissionRecord
  isLast: boolean
  isRetrying: boolean
  onRetire: () => void
}) {
  const statusBadge = commissionStatusLabel(record.status)
  const isPending =
    record.status === "pending" || record.status === "pending_kyc"

  // D1+D2 — show the "Retirer" button only when the row is retire-
  // eligible. We trust the backend flag when present and fall back
  // to deriving from status so older API responses still surface
  // the button correctly.
  const retireEligible =
    typeof record.retire_eligible === "boolean"
      ? record.retire_eligible
      : record.status === "pending_kyc" || record.status === "failed"

  // 3px left accent rail per status — same pattern as the missions
  // history. Pending = amber, clawed_back = corail (warm), failed =
  // destructive, paid = sapin, anything else = neutral border.
  const railColor = isPending
    ? "bg-warning"
    : record.status === "clawed_back"
      ? "bg-primary"
      : record.status === "failed"
        ? "bg-destructive"
        : record.status === "paid"
          ? "bg-success"
          : "bg-border-strong"

  // Icon-square background — the icon foreground colour lives on the
  // icon itself (see CommissionStatusIcon below) so the surface stays
  // calm in Soleil tones while the icon keeps a single semantic tint.
  const iconBg =
    record.status === "paid"
      ? "bg-success-soft"
      : record.status === "failed"
        ? "bg-destructive/10"
        : record.status === "clawed_back"
          ? "bg-primary-soft"
          : "bg-amber-soft"

  const rowContent = (
    <div
      className={cn(
        // Narrow viewports: tighter horizontal padding/gap matching
        // the mobile native WalletCommissionTile.
        "relative flex items-center gap-2.5 px-3 py-3 md:gap-4 md:px-6 md:py-4",
        !isLast && "border-b border-border",
      )}
    >
      <span
        aria-hidden="true"
        className={cn("absolute inset-y-0 left-0 w-[3px]", railColor)}
      />
      <div
        className={cn(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-xl md:h-9 md:w-9",
          iconBg,
        )}
      >
        <CommissionStatusIcon status={record.status} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-semibold text-foreground md:text-[14px]">
          Commission du {formatDate(record.created_at)}
        </p>
        <p className="mt-0.5 text-[11.5px] text-muted-foreground md:text-[12px]">
          sur {formatEur(record.gross_amount_cents)} de mission
        </p>
      </div>
      <div className="shrink-0 text-right">
        <p className="font-serif text-[15px] font-semibold leading-none tracking-[-0.015em] text-foreground md:text-[18px]">
          {formatEur(record.commission_cents)}
        </p>
        <span
          className={cn(
            "mt-1 inline-block rounded-full px-2 py-0.5 text-[10.5px] font-semibold",
            statusBadge.cls,
          )}
        >
          {statusBadge.label}
        </span>
      </div>
      {retireEligible ? (
        <button
          type="button"
          onClick={(event) => {
            // Stop the click from bubbling to the parent <Link> so
            // pressing Retirer doesn't also navigate to the parent
            // referral.
            event.preventDefault()
            event.stopPropagation()
            if (!isRetrying) onRetire()
          }}
          disabled={isRetrying}
          aria-label="Retirer cette commission"
          className={cn(
            "ml-1 inline-flex shrink-0 items-center justify-center gap-1 rounded-full px-2.5 py-1",
            "text-[11px] font-semibold transition-colors",
            "bg-primary text-primary-foreground hover:bg-primary-deep",
            isRetrying && "cursor-wait opacity-70",
          )}
        >
          {isRetrying ? (
            <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
          ) : null}
          Retirer
        </button>
      ) : null}
      {record.referral_id ? (
        <ChevronRight
          className="h-4 w-4 shrink-0 text-subtle-foreground transition-colors group-hover:text-primary"
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
        className="group block transition-colors duration-150 hover:bg-primary-soft/30"
      >
        {rowContent}
      </Link>
    )
  }
  return rowContent
}

function commissionStatusLabel(status: string): {
  label: string
  cls: string
} {
  switch (status) {
    case "paid":
      return {
        label: "Reçue",
        cls: "bg-success-soft text-success",
      }
    case "pending":
      return {
        label: "En attente",
        cls: "bg-amber-soft text-warning",
      }
    case "pending_kyc":
      return {
        label: "KYC requis",
        cls: "bg-amber-soft text-warning",
      }
    case "clawed_back":
      return {
        label: "Reprise",
        cls: "bg-primary-soft text-primary-deep",
      }
    case "failed":
      return {
        label: "Échec",
        cls: "bg-destructive/10 text-destructive",
      }
    case "cancelled":
      return {
        label: "Annulée",
        cls: "bg-border text-muted-foreground",
      }
    default:
      return {
        label: status,
        cls: "bg-border text-muted-foreground",
      }
  }
}

// Icon foreground = single semantic tint (matches PaymentStatusIcon
// pattern in wallet-transactions-list). Background stays in Soleil
// tones via the iconBg square in CommissionRow.
function CommissionStatusIcon({ status }: { status: string }) {
  if (status === "paid")
    return (
      <CheckCircle2
        className="h-4 w-4 text-green-500"
        strokeWidth={1.8}
      />
    )
  if (status === "failed")
    return <XCircle className="h-4 w-4 text-red-500" strokeWidth={1.8} />
  if (status === "clawed_back")
    return <Undo2 className="h-4 w-4 text-blue-500" strokeWidth={1.8} />
  return <Clock className="h-4 w-4 text-amber-500" strokeWidth={1.8} />
}
