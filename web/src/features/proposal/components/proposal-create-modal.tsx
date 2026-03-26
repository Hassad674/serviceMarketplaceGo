"use client"

import { useState, useEffect, useCallback } from "react"
import { X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalFormData } from "../types"
import { createDefaultProposalForm } from "../types"
import { ProposalPaymentSelector } from "./proposal-payment-selector"
import { ProposalStructureSection } from "./proposal-structure-section"
import { ProposalDetailsSection } from "./proposal-details-section"
import { ProposalTimelineSection } from "./proposal-timeline-section"
import { ProposalPreview } from "./proposal-preview"

type ProposalCreateModalProps = {
  open: boolean
  onClose: () => void
}

export function ProposalCreateModal({ open, onClose }: ProposalCreateModalProps) {
  const t = useTranslations("proposal")
  const [formData, setFormData] = useState<ProposalFormData>(createDefaultProposalForm)

  // Reset form when modal opens
  useEffect(() => {
    if (open) {
      setFormData(createDefaultProposalForm())
    }
  }, [open])

  // Close on Escape key
  useEffect(() => {
    if (!open) return

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [open, onClose])

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden"
    } else {
      document.body.style.overflow = ""
    }
    return () => {
      document.body.style.overflow = ""
    }
  }, [open])

  const updateField = useCallback(<K extends keyof ProposalFormData>(
    field: K,
    value: ProposalFormData[K],
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }, [])

  function handleSend() {
    // Mock: just log the data. Backend integration will come later.
    console.log("Sending proposal:", formData)
    onClose()
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 px-4 py-8"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
      role="dialog"
      aria-modal="true"
      aria-label={t("createProposal")}
    >
      <div
        className={cn(
          "w-full max-w-5xl rounded-2xl bg-white dark:bg-gray-900",
          "border border-gray-200 dark:border-gray-700",
          "shadow-xl animate-scale-in",
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            {t("createProposal")}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className={cn(
              "rounded-lg p-2 text-gray-400 transition-colors",
              "hover:bg-gray-100 hover:text-gray-600",
              "dark:hover:bg-gray-800 dark:hover:text-gray-300",
            )}
            aria-label={t("cancel")}
          >
            <X className="h-5 w-5" strokeWidth={1.5} />
          </button>
        </div>

        {/* Body */}
        <div className="max-h-[calc(100vh-12rem)] overflow-y-auto px-6 py-6">
          <div className="flex flex-col gap-8 lg:flex-row">
            {/* Left: Form */}
            <div className="flex-1 space-y-8 min-w-0">
              <ProposalPaymentSelector
                value={formData.paymentType}
                onChange={(v) => updateField("paymentType", v)}
              />

              <ProposalStructureSection formData={formData} updateField={updateField} />

              <ProposalDetailsSection formData={formData} updateField={updateField} />

              <ProposalTimelineSection formData={formData} updateField={updateField} />
            </div>

            {/* Right: Preview */}
            <div className="hidden w-[300px] shrink-0 lg:block">
              <ProposalPreview formData={formData} />
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 border-t border-gray-200 dark:border-gray-700 px-6 py-4">
          <button
            type="button"
            onClick={onClose}
            className={cn(
              "rounded-xl px-5 py-2.5 text-sm font-medium",
              "text-gray-600 dark:text-gray-400 transition-all duration-200",
              "hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white",
            )}
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            onClick={handleSend}
            className={cn(
              "rounded-xl px-6 py-2.5 text-sm font-semibold text-white",
              "gradient-primary transition-all duration-200",
              "hover:shadow-glow active:scale-[0.98]",
            )}
          >
            {t("sendProposal")}
          </button>
        </div>
      </div>
    </div>
  )
}
