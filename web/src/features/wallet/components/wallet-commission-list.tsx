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
        <span
          className={cn(
            "mt-0.5 inline-block rounded-full px-2 py-0.5 text-[10px] font-medium",
            status.cls,
          )}
        >
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

function commissionStatusLabel(status: string): {
  label: string
  cls: string
} {
  switch (status) {
    case "paid":
      return {
        label: "Reçue",
        cls: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400",
      }
    case "pending":
      return {
        label: "En attente",
        cls: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
      }
    case "pending_kyc":
      return {
        label: "KYC requis",
        cls: "bg-orange-50 text-orange-700 dark:bg-orange-500/10 dark:text-orange-400",
      }
    case "clawed_back":
      return {
        label: "Reprise",
        cls: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
      }
    case "failed":
      return {
        label: "Échec",
        cls: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400",
      }
    case "cancelled":
      return { label: "Annulée", cls: "bg-slate-100 text-slate-600" }
    default:
      return { label: status, cls: "bg-slate-100 text-slate-600" }
  }
}

function CommissionStatusIcon({ status }: { status: string }) {
  if (status === "paid")
    return <CheckCircle2 className="h-5 w-5 shrink-0 text-green-500" />
  if (status === "failed")
    return <XCircle className="h-5 w-5 shrink-0 text-red-500" />
  if (status === "clawed_back")
    return <Undo2 className="h-5 w-5 shrink-0 text-blue-500" />
  return <Clock className="h-5 w-5 shrink-0 text-amber-500" />
}
