"use client"

import { useState, useCallback, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { X, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalFormData } from "../types"
import { createEmptyProposalForm } from "../types"
import { ProposalPreview } from "./proposal-preview"
import { FileDropZone } from "./file-drop-zone"
import { useCreateProposal, useModifyProposal } from "../hooks/use-proposals"
import { getProposal } from "../api/proposal-api"

const TITLE_MAX_LENGTH = 100

export function CreateProposalPage() {
  const t = useTranslations("proposal")
  const router = useRouter()
  const searchParams = useSearchParams()

  const recipientId = searchParams.get("to") ?? ""
  const conversationId = searchParams.get("conversation") ?? ""
  const modifyId = searchParams.get("modify")

  const [formData, setFormData] = useState<ProposalFormData>(() => ({
    ...createEmptyProposalForm(),
    recipientId,
    conversationId,
  }))
  const [recipientName, setRecipientName] = useState("")
  const [submitError, setSubmitError] = useState<string | null>(null)

  const createMutation = useCreateProposal()
  const modifyMutation = useModifyProposal()
  const isSubmitting = createMutation.isPending || modifyMutation.isPending

  // Pre-fill form when modifying an existing proposal
  useEffect(() => {
    if (!modifyId) return
    getProposal(modifyId).then((p) => {
      setFormData((prev) => ({
        ...prev,
        title: p.title,
        description: p.description,
        amount: (p.amount / 100).toString(),
        deadline: p.deadline ? p.deadline.split("T")[0] : "",
      }))
    }).catch(() => {
      setSubmitError(t("fetchError"))
    })
  }, [modifyId, t])

  // Sync query params into form data when they change
  useEffect(() => {
    setFormData((prev) => ({ ...prev, recipientId, conversationId }))
  }, [recipientId, conversationId])

  // Mock recipient fetch
  useEffect(() => {
    if (!recipientId) return
    setRecipientName(`User ${recipientId.slice(0, 8)}`)
  }, [recipientId])

  const updateField = useCallback(<K extends keyof ProposalFormData>(
    field: K,
    value: ProposalFormData[K],
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }, [])

  const isValid =
    formData.title.trim().length > 0 &&
    formData.description.trim().length > 0 &&
    Number(formData.amount) > 0

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!isValid || isSubmitting) return
    setSubmitError(null)

    const amountCents = Math.round(Number(formData.amount) * 100)

    if (modifyId) {
      modifyMutation.mutate(
        {
          id: modifyId,
          data: {
            title: formData.title.trim(),
            description: formData.description.trim(),
            amount: amountCents,
            deadline: formData.deadline || undefined,
          },
        },
        {
          onSuccess: () => {
            router.push(`/messages?id=${conversationId}`)
          },
          onError: (err) => {
            setSubmitError(err.message)
          },
        },
      )
    } else {
      createMutation.mutate(
        {
          recipient_id: recipientId,
          conversation_id: conversationId,
          title: formData.title.trim(),
          description: formData.description.trim(),
          amount: amountCents,
          deadline: formData.deadline || undefined,
        },
        {
          onSuccess: () => {
            router.push(`/messages?id=${conversationId}`)
          },
          onError: (err) => {
            setSubmitError(err.message)
          },
        },
      )
    }
  }

  function handleCancel() {
    router.back()
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      {/* Top bar */}
      <header
        className={cn(
          "sticky top-0 z-10 flex h-16 items-center justify-between border-b px-4 sm:px-6",
          "border-gray-200 bg-white/80 backdrop-blur-xl",
          "dark:border-gray-800 dark:bg-gray-900/80",
        )}
      >
        <button
          type="button"
          onClick={handleCancel}
          className={cn(
            "rounded-lg p-2 text-gray-400 transition-colors",
            "hover:bg-gray-100 hover:text-gray-600",
            "dark:hover:bg-gray-800 dark:hover:text-gray-300",
          )}
          aria-label={t("proposalCancel")}
        >
          <X className="h-5 w-5" strokeWidth={1.5} />
        </button>

        <h1 className="text-base font-semibold text-gray-900 dark:text-white">
          {modifyId ? t("modify") : t("createProposal")}
        </h1>

        <button
          type="submit"
          form="proposal-form"
          disabled={!isValid || isSubmitting}
          className={cn(
            "rounded-xl px-5 py-2 text-sm font-semibold text-white transition-all duration-200",
            "flex items-center gap-2",
            isValid && !isSubmitting
              ? "gradient-primary hover:shadow-glow active:scale-[0.98]"
              : "cursor-not-allowed bg-gray-200 text-gray-400 dark:bg-gray-700 dark:text-gray-500",
          )}
        >
          {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
          {t("proposalSend")}
        </button>
      </header>

      {/* Body */}
      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
        {submitError && (
          <div className="mb-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
            {submitError}
          </div>
        )}

        <div className="flex flex-col gap-8 lg:flex-row">
          {/* Form column */}
          <form
            id="proposal-form"
            onSubmit={handleSubmit}
            className="min-w-0 flex-1 space-y-6"
          >
            {/* Recipient (read-only) */}
            <RecipientField name={recipientName} />

            <div className="border-t border-gray-200 dark:border-gray-800" />

            {/* Title */}
            <div className="space-y-2">
              <label
                htmlFor="proposal-title"
                className="text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {t("proposalTitle")} <span className="text-rose-500">*</span>
              </label>
              <div className="relative">
                <input
                  id="proposal-title"
                  type="text"
                  value={formData.title}
                  onChange={(e) => updateField("title", e.target.value.slice(0, TITLE_MAX_LENGTH))}
                  placeholder={t("proposalTitlePlaceholder")}
                  maxLength={TITLE_MAX_LENGTH}
                  className={cn(
                    "h-11 w-full rounded-lg border border-gray-200 bg-white px-4 text-sm",
                    "shadow-xs transition-all duration-200",
                    "placeholder:text-gray-400",
                    "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                    "dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder:text-gray-500",
                    "dark:focus:border-rose-400 dark:focus:ring-rose-400/10",
                  )}
                  aria-required="true"
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">
                  {formData.title.length}/{TITLE_MAX_LENGTH}
                </span>
              </div>
            </div>

            {/* Description */}
            <div className="space-y-2">
              <label
                htmlFor="proposal-description"
                className="text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {t("proposalDescription")} <span className="text-rose-500">*</span>
              </label>
              <textarea
                id="proposal-description"
                value={formData.description}
                onChange={(e) => updateField("description", e.target.value)}
                placeholder={t("proposalDescriptionPlaceholder")}
                rows={5}
                className={cn(
                  "w-full rounded-lg border border-gray-200 bg-white px-4 py-3 text-sm",
                  "shadow-xs transition-all duration-200 resize-none",
                  "placeholder:text-gray-400",
                  "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                  "dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder:text-gray-500",
                  "dark:focus:border-rose-400 dark:focus:ring-rose-400/10",
                )}
                aria-required="true"
              />
            </div>

            {/* Amount */}
            <div className="space-y-2">
              <label
                htmlFor="proposal-amount"
                className="text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {t("proposalAmount")} <span className="text-rose-500">*</span>
              </label>
              <div className="relative">
                <span className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-500 dark:text-gray-400">
                  &euro;
                </span>
                <input
                  id="proposal-amount"
                  type="number"
                  min="0"
                  step="0.01"
                  value={formData.amount}
                  onChange={(e) => updateField("amount", e.target.value)}
                  placeholder={t("proposalAmountPlaceholder")}
                  className={cn(
                    "h-11 w-full rounded-lg border border-gray-200 bg-white pl-9 pr-4 text-sm",
                    "shadow-xs transition-all duration-200",
                    "placeholder:text-gray-400",
                    "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                    "dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder:text-gray-500",
                    "dark:focus:border-rose-400 dark:focus:ring-rose-400/10",
                    "[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none",
                  )}
                  aria-required="true"
                />
              </div>
            </div>

            {/* Deadline */}
            <div className="space-y-2">
              <label
                htmlFor="proposal-deadline"
                className="text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {t("proposalDeadline")}
              </label>
              <input
                id="proposal-deadline"
                type="date"
                value={formData.deadline}
                onChange={(e) => updateField("deadline", e.target.value)}
                min={new Date().toISOString().split("T")[0]}
                className={cn(
                  "h-11 w-full rounded-lg border border-gray-200 bg-white px-4 text-sm",
                  "shadow-xs transition-all duration-200",
                  "text-gray-700 dark:text-gray-300",
                  "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                  "dark:border-gray-700 dark:bg-gray-800",
                  "dark:focus:border-rose-400 dark:focus:ring-rose-400/10",
                )}
              />
            </div>

            {/* Documents */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {t("proposalDocuments")}
              </label>
              <FileDropZone
                files={formData.files}
                onFilesChange={(files) => updateField("files", files)}
              />
            </div>

            {/* Footer buttons (mobile only, below form) */}
            <div className="flex gap-3 pt-4 lg:hidden">
              <button
                type="button"
                onClick={handleCancel}
                className={cn(
                  "flex-1 rounded-xl px-5 py-2.5 text-sm font-medium",
                  "text-gray-600 transition-all duration-200",
                  "hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800",
                )}
              >
                {t("proposalCancel")}
              </button>
              <button
                type="submit"
                disabled={!isValid || isSubmitting}
                className={cn(
                  "flex-1 rounded-xl px-5 py-2.5 text-sm font-semibold text-white transition-all duration-200",
                  "flex items-center justify-center gap-2",
                  isValid && !isSubmitting
                    ? "gradient-primary hover:shadow-glow active:scale-[0.98]"
                    : "cursor-not-allowed bg-gray-200 text-gray-400 dark:bg-gray-700 dark:text-gray-500",
                )}
              >
                {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
                {t("proposalSend")}
              </button>
            </div>
          </form>

          {/* Preview column (desktop) */}
          <div className="hidden w-[360px] shrink-0 lg:block">
            <div className="sticky top-24">
              <ProposalPreview formData={formData} recipientName={recipientName} />
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}

function RecipientField({ name }: { name: string }) {
  const t = useTranslations("proposal")

  const initials = name
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {t("proposalRecipient")}
      </p>
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-xs font-semibold text-white">
          {initials || "?"}
        </div>
        <p className="text-sm font-medium text-gray-900 dark:text-white">
          {name || "\u2014"}
        </p>
      </div>
    </div>
  )
}
