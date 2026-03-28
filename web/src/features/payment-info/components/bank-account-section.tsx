"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { PaymentInfoFormData, BankAccountMode } from "../types"

type BankAccountSectionProps = {
  data: PaymentInfoFormData
  onChange: (field: keyof PaymentInfoFormData, value: string) => void
  onChangeBankMode: (mode: BankAccountMode) => void
}

function InputField({
  label,
  value,
  onChange,
  placeholder,
  required,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  required?: boolean
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className={cn(
          "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
        )}
      />
    </div>
  )
}

export function BankAccountSection({ data, onChange, onChangeBankMode }: BankAccountSectionProps) {
  const t = useTranslations("paymentInfo")
  const isIbanMode = data.bankMode === "iban"

  return (
    <section className="rounded-2xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
        {t("bankAccount")}
      </h2>

      <div className="grid gap-4 sm:grid-cols-2">
        {isIbanMode ? (
          <>
            <div className="sm:col-span-2">
              <InputField
                label={t("iban")}
                value={data.iban}
                onChange={(v) => onChange("iban", v)}
                placeholder={t("ibanPlaceholder")}
                required
              />
            </div>
            <div className="sm:col-span-2">
              <button
                type="button"
                onClick={() => onChangeBankMode("local")}
                className="text-sm font-medium text-rose-500 transition-colors hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300"
              >
                {t("noIban")}
              </button>
            </div>
          </>
        ) : (
          <>
            <InputField
              label={t("accountNumber")}
              value={data.accountNumber}
              onChange={(v) => onChange("accountNumber", v)}
              required
            />
            <InputField
              label={t("routingNumber")}
              value={data.routingNumber}
              onChange={(v) => onChange("routingNumber", v)}
              required
            />
            <div className="sm:col-span-2">
              <button
                type="button"
                onClick={() => onChangeBankMode("iban")}
                className="text-sm font-medium text-rose-500 transition-colors hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300"
              >
                {t("useIban")}
              </button>
            </div>
          </>
        )}

        <div className="sm:col-span-2">
          <InputField
            label={t("accountHolder")}
            value={data.accountHolder}
            onChange={(v) => onChange("accountHolder", v)}
            required
          />
        </div>
      </div>
    </section>
  )
}
