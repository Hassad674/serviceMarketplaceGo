"use client"

import { AlertCircle, CheckCircle2, Clock, CreditCard, Send } from "lucide-react"
import { useTranslations } from "next-intl"

export type AccountStatus = {
  account_id: string
  country: string
  business_type: string
  charges_enabled: boolean
  payouts_enabled: boolean
  details_submitted: boolean
  requirements_currently_due: string[]
  requirements_past_due: string[]
  requirements_eventually_due: string[]
  requirements_pending_verification: string[]
  requirements_count: number
  disabled_reason?: string
}

type AccountStatusCardProps = {
  status: AccountStatus
}

export function AccountStatusCard({ status }: AccountStatusCardProps) {
  const t = useTranslations("paymentInfo")
  const fullyActive =
    status.charges_enabled && status.payouts_enabled && status.requirements_count === 0
  const hasPastDue = status.requirements_past_due.length > 0

  return (
    <section
      aria-label="Statut du compte de paiement"
      className="overflow-hidden rounded-2xl border border-slate-100 bg-white shadow-sm"
    >
      {/* Header gradient */}
      <div
        className={`relative overflow-hidden px-6 py-5 ${
          fullyActive
            ? "bg-gradient-to-br from-emerald-500 to-emerald-600"
            : hasPastDue
              ? "bg-gradient-to-br from-red-500 to-red-600"
              : "bg-gradient-to-br from-rose-500 via-rose-600 to-purple-600"
        }`}
      >
        <div className="relative flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm">
              {fullyActive ? (
                <CheckCircle2 className="h-5 w-5 text-white" aria-hidden />
              ) : hasPastDue ? (
                <AlertCircle className="h-5 w-5 text-white" aria-hidden />
              ) : (
                <Clock className="h-5 w-5 text-white" aria-hidden />
              )}
            </div>
            <div>
              <h2 className="text-lg font-bold text-white">
                {fullyActive
                  ? t("accountActive")
                  : hasPastDue
                    ? t("urgentAction")
                    : t("verificationInProgress")}
              </h2>
              <p className="text-[13px] text-white/90">
                {fullyActive
                  ? t("accountActiveDesc")
                  : status.requirements_count > 0
                    ? t("itemsToComplete", { count: status.requirements_count })
                    : t("processingByStripe")}
              </p>
            </div>
          </div>
          <code className="hidden rounded-md bg-white/20 px-2 py-1 font-mono text-[11px] text-white backdrop-blur-sm sm:block">
            {status.account_id}
          </code>
        </div>
      </div>

      {/* Capabilities grid */}
      <div className="grid grid-cols-1 divide-y divide-slate-100 sm:grid-cols-2 sm:divide-x sm:divide-y-0">
        <CapabilityRow
          icon={CreditCard}
          label={t("incomingPayments")}
          enabled={status.charges_enabled}
          activeLabel={t("active")}
          pendingLabel={t("pending")}
        />
        <CapabilityRow
          icon={Send}
          label={t("outgoingTransfers")}
          enabled={status.payouts_enabled}
          activeLabel={t("active")}
          pendingLabel={t("pending")}
        />
      </div>
    </section>
  )
}

function CapabilityRow({
  icon: Icon,
  label,
  enabled,
  activeLabel,
  pendingLabel,
}: {
  icon: typeof CreditCard
  label: string
  enabled: boolean
  activeLabel: string
  pendingLabel: string
}) {
  return (
    <div className="flex items-center justify-between px-6 py-4">
      <div className="flex items-center gap-3">
        <span
          className={`flex h-8 w-8 items-center justify-center rounded-lg ${
            enabled ? "bg-emerald-50 text-emerald-600" : "bg-slate-100 text-slate-400"
          }`}
          aria-hidden
        >
          <Icon className="h-4 w-4" />
        </span>
        <span className="text-[14px] font-semibold text-slate-900">{label}</span>
      </div>
      <span
        className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold ${
          enabled
            ? "border-emerald-200 bg-emerald-50 text-emerald-700"
            : "border-amber-200 bg-amber-50 text-amber-800"
        }`}
      >
        <span
          className={`h-1.5 w-1.5 rounded-full ${enabled ? "bg-emerald-500" : "bg-amber-500"}`}
          aria-hidden
        />
        {enabled ? activeLabel : pendingLabel}
      </span>
    </div>
  )
}
