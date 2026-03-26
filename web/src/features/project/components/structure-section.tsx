"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProjectFormData, EscrowStructure, InvoiceBillingType, InvoiceFrequency } from "../types"
import { MilestoneEditor } from "./milestone-editor"

type StructureSectionProps = {
  formData: ProjectFormData
  updateField: <K extends keyof ProjectFormData>(field: K, value: ProjectFormData[K]) => void
}

export function StructureSection({ formData, updateField }: StructureSectionProps) {
  const t = useTranslations("projects")

  if (formData.paymentType === "escrow") {
    return <EscrowStructureSection formData={formData} updateField={updateField} />
  }
  return <InvoiceStructureSection formData={formData} updateField={updateField} />
}

function EscrowStructureSection({ formData, updateField }: StructureSectionProps) {
  const t = useTranslations("projects")

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        {t("structure")}
      </h2>

      {/* Toggle: Milestone / One-time */}
      <div className="inline-flex rounded-lg bg-gray-100 dark:bg-gray-800 p-1">
        <ToggleButton
          active={formData.escrowStructure === "milestone"}
          onClick={() => updateField("escrowStructure", "milestone" as EscrowStructure)}
          label={t("milestone")}
        />
        <ToggleButton
          active={formData.escrowStructure === "one-time"}
          onClick={() => updateField("escrowStructure", "one-time" as EscrowStructure)}
          label={t("oneTime")}
        />
      </div>

      {formData.escrowStructure === "milestone" ? (
        <MilestoneEditor
          milestones={formData.milestones}
          onChange={(m) => updateField("milestones", m)}
        />
      ) : (
        <AmountInput
          value={formData.oneTimeAmount}
          onChange={(v) => updateField("oneTimeAmount", v)}
          label={t("amount")}
        />
      )}
    </section>
  )
}

function InvoiceStructureSection({ formData, updateField }: StructureSectionProps) {
  const t = useTranslations("projects")

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
        {t("structure")}
      </h2>

      {/* Toggle: Fixed / Hourly */}
      <div className="inline-flex rounded-lg bg-gray-100 dark:bg-gray-800 p-1">
        <ToggleButton
          active={formData.invoiceBillingType === "fixed"}
          onClick={() => updateField("invoiceBillingType", "fixed" as InvoiceBillingType)}
          label={t("fixed")}
        />
        <ToggleButton
          active={formData.invoiceBillingType === "hourly"}
          onClick={() => updateField("invoiceBillingType", "hourly" as InvoiceBillingType)}
          label={t("hourly")}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {/* Rate */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("rate")}
          </label>
          <div className="relative">
            <input
              type="number"
              min="0"
              step="0.01"
              value={formData.invoiceRate}
              onChange={(e) => updateField("invoiceRate", e.target.value)}
              placeholder="0.00"
              className={cn(
                "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
                "bg-gray-50 dark:bg-gray-800 pl-3 pr-14 text-sm",
                "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
                "transition-all duration-200",
                "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
              )}
            />
            <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">
              &euro;/{formData.invoiceBillingType === "hourly" ? "hr" : "wk"}
            </span>
          </div>
        </div>

        {/* Frequency */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("frequency")}
          </label>
          <select
            value={formData.invoiceFrequency}
            onChange={(e) => updateField("invoiceFrequency", e.target.value as InvoiceFrequency)}
            className={cn(
              "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
              "bg-gray-50 dark:bg-gray-800 px-3 text-sm",
              "text-gray-900 dark:text-white",
              "transition-all duration-200",
              "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            )}
          >
            <option value="weekly">{t("weekly")}</option>
            <option value="bi-weekly">{t("biWeekly")}</option>
            <option value="monthly">{t("monthly")}</option>
          </select>
        </div>

        {/* Amount */}
        <AmountInput
          value={formData.invoiceAmount}
          onChange={(v) => updateField("invoiceAmount", v)}
          label={t("amount")}
        />
      </div>
    </section>
  )
}

type ToggleButtonProps = {
  active: boolean
  onClick: () => void
  label: string
}

function ToggleButton({ active, onClick, label }: ToggleButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded-md px-4 py-1.5 text-sm font-medium transition-all duration-200",
        active
          ? "bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm"
          : "text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300",
      )}
    >
      {label}
    </button>
  )
}

type AmountInputProps = {
  value: string
  onChange: (value: string) => void
  label: string
}

function AmountInput({ value, onChange, label }: AmountInputProps) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      <div className="relative">
        <input
          type="number"
          min="0"
          step="0.01"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="0.00"
          className={cn(
            "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
            "bg-gray-50 dark:bg-gray-800 pl-3 pr-8 text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          )}
        />
        <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">
          &euro;
        </span>
      </div>
    </div>
  )
}
