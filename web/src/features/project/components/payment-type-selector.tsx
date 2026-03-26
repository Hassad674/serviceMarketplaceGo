"use client"

import { Check, FileText, ShieldCheck } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { PaymentType } from "../types"

type PaymentTypeSelectorProps = {
  value: PaymentType
  onChange: (value: PaymentType) => void
}

export function PaymentTypeSelector({ value, onChange }: PaymentTypeSelectorProps) {
  const t = useTranslations("projects")

  return (
    <section>
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        {t("paymentType")}
      </h2>
      <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <PaymentCard
          selected={value === "invoice"}
          onClick={() => onChange("invoice")}
          icon={FileText}
          title={t("invoiceBilling")}
          description={t("invoiceBillingDesc")}
        />
        <PaymentCard
          selected={value === "escrow"}
          onClick={() => onChange("escrow")}
          icon={ShieldCheck}
          title={t("escrowPayments")}
          description={t("escrowPaymentsDesc")}
        />
      </div>
    </section>
  )
}

type PaymentCardProps = {
  selected: boolean
  onClick: () => void
  icon: React.ElementType
  title: string
  description: string
}

function PaymentCard({ selected, onClick, icon: Icon, title, description }: PaymentCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "relative flex flex-col items-start gap-2 rounded-xl border-2 p-5 text-left",
        "transition-all duration-200",
        selected
          ? "border-rose-500 bg-rose-50 dark:bg-rose-500/10 dark:border-rose-400"
          : "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-600",
      )}
    >
      {selected && (
        <span className="absolute right-3 top-3 flex h-6 w-6 items-center justify-center rounded-full bg-rose-500 text-white">
          <Check className="h-3.5 w-3.5" strokeWidth={2.5} />
        </span>
      )}
      <div
        className={cn(
          "flex h-10 w-10 items-center justify-center rounded-lg",
          selected
            ? "bg-rose-100 dark:bg-rose-500/20"
            : "bg-gray-100 dark:bg-gray-800",
        )}
      >
        <Icon
          className={cn(
            "h-5 w-5",
            selected
              ? "text-rose-600 dark:text-rose-400"
              : "text-gray-500 dark:text-gray-400",
          )}
          strokeWidth={1.5}
        />
      </div>
      <div>
        <p
          className={cn(
            "text-sm font-semibold",
            selected
              ? "text-rose-700 dark:text-rose-300"
              : "text-gray-900 dark:text-white",
          )}
        >
          {title}
        </p>
        <p className="mt-1 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
          {description}
        </p>
      </div>
    </button>
  )
}
