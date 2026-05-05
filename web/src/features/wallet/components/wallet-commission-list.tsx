"use client"

import Link from "next/link"
import {
  CheckCircle2,
  ChevronRight,
  Clock,
  Sparkles,
  Undo2,
  UserCheck,
  XCircle,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import {
  BalanceCard,
  SectionHeader,
} from "./wallet-transactions-list"
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
  if (totalActivity === 0 && records.length === 0) {
    return null
  }

  return (
    <section className="space-y-4">
      <SectionHeader icon={Sparkles} title="Mes commissions d'apport" />

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
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
        <div className="border-b border-border px-6 py-5">
          <h3 className="font-serif text-[22px] font-medium tracking-[-0.015em] text-foreground">
            Historique des commissions
          </h3>
          <p className="mt-0.5 text-[12.5px] text-muted-foreground">
            Chaque mise en relation que vous avez facilitée
          </p>
        </div>
        {records.length === 0 ? (
          <div className="px-6 py-8 text-center text-[12.5px] text-muted-foreground">
            Aucune commission pour le moment
          </div>
        ) : (
          <div>
            {records.map((r, index) => (
              <CommissionRow
                key={r.id}
                record={r}
                isLast={index === records.length - 1}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function CommissionRow({
  record,
  isLast,
}: {
  record: WalletCommissionRecord
  isLast: boolean
}) {
  const statusBadge = commissionStatusLabel(record.status)
  const isPending =
    record.status === "pending" || record.status === "pending_kyc"

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
        "relative flex items-center gap-4 px-6 py-4",
        !isLast && "border-b border-border",
      )}
    >
      <span
        aria-hidden="true"
        className={cn("absolute inset-y-0 left-0 w-[3px]", railColor)}
      />
      <div
        className={cn(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl",
          iconBg,
        )}
      >
        <CommissionStatusIcon status={record.status} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[14px] font-semibold text-foreground">
          Commission du {formatDate(record.created_at)}
        </p>
        <p className="mt-0.5 text-[12px] text-muted-foreground">
          sur {formatEur(record.gross_amount_cents)} de mission
        </p>
      </div>
      <div className="shrink-0 text-right">
        <p className="font-serif text-[18px] font-semibold leading-none tracking-[-0.015em] text-foreground">
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
