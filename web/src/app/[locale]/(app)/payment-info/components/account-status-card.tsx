"use client"

import { AlertCircle, CheckCircle2, Clock, CreditCard, Send } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"

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

  const tone = fullyActive ? "success" : hasPastDue ? "danger" : "pending"
  const HeaderIcon = fullyActive
    ? CheckCircle2
    : hasPastDue
      ? AlertCircle
      : Clock

  const headerBg = {
    success: "bg-success-soft",
    danger: "bg-primary-soft",
    pending: "bg-amber-soft",
  }[tone]

  const headerIconColor = {
    success: "text-success",
    danger: "text-destructive",
    pending: "text-warning",
  }[tone]

  return (
    <section
      aria-label={t("subheader")}
      className={cn(
        "overflow-hidden border-y border-border bg-card",
        "sm:rounded-2xl sm:border sm:shadow-card",
      )}
    >
      {/* Header — soft tinted band, no glaring gradient */}
      <div className={cn("relative px-4 py-5 sm:px-6 sm:py-6", headerBg)}>
        <div className="relative flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <span
              className={cn(
                "flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-card shadow-card",
                headerIconColor,
              )}
              aria-hidden
            >
              <HeaderIcon className="h-5 w-5" />
            </span>
            <div>
              <h2 className="font-serif text-[20px] font-medium tracking-[-0.01em] text-foreground">
                {fullyActive
                  ? t("accountActive")
                  : hasPastDue
                    ? t("urgentAction")
                    : t("verificationInProgress")}
              </h2>
              <p className="mt-1 text-[13px] leading-relaxed text-muted-foreground">
                {fullyActive
                  ? t("accountActiveDesc")
                  : status.requirements_count > 0
                    ? t("itemsToComplete", { count: status.requirements_count })
                    : t("processingByStripe")}
              </p>
            </div>
          </div>
          <code className="hidden rounded-full border border-border-strong bg-card/80 px-2.5 py-1 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-subtle-foreground sm:inline-block">
            {status.account_id}
          </code>
        </div>
      </div>

      {/* Capabilities grid */}
      <div className="grid grid-cols-1 divide-y divide-border sm:grid-cols-2 sm:divide-x sm:divide-y-0">
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
    <div className="flex items-center justify-between gap-3 px-4 py-3.5 sm:px-6 sm:py-4">
      <div className="flex items-center gap-3">
        <span
          className={cn(
            "flex h-9 w-9 items-center justify-center rounded-xl",
            enabled
              ? "bg-success-soft text-success"
              : "bg-amber-soft text-warning",
          )}
          aria-hidden
        >
          <Icon className="h-4 w-4" />
        </span>
        <span className="text-[14px] font-semibold text-foreground">{label}</span>
      </div>
      <span
        className={cn(
          "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold",
          enabled
            ? "border-success/30 bg-success-soft text-success"
            : "border-warning/30 bg-amber-soft text-warning",
        )}
      >
        <span
          className={cn(
            "h-1.5 w-1.5 rounded-full",
            enabled ? "bg-success" : "bg-warning",
          )}
          aria-hidden
        />
        {enabled ? activeLabel : pendingLabel}
      </span>
    </div>
  )
}
