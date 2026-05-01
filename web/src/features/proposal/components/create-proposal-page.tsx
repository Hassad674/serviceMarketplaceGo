"use client"

import { useState, useCallback, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { X, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import type { ProposalFormData, MilestoneFormItem } from "../types"
import {
  createEmptyProposalForm,
  sumMilestoneAmounts,
  validateMilestoneDeadlines,
} from "../types"
import { ProposalPreview } from "./proposal-preview"
import { FileDropZone } from "./file-drop-zone"
import { PaymentModeToggle } from "./payment-mode-toggle"
import { MilestoneEditor } from "./milestone-editor"
import { useCreateProposal, useModifyProposal } from "../hooks/use-proposals"
import { getProposal, getUploadURL } from "../api/proposal-api"
import type { CreateProposalData, MilestoneInputData } from "../api/proposal-api"
import { FeePreview } from "@/features/billing/components/fee-preview"
import { UpgradeCta } from "@/features/subscription/components/upgrade-cta"
import { UpgradeModal } from "@/features/subscription/components/upgrade-modal"
import { useUser } from "@/shared/hooks/use-user"

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
  const [isUploading, setIsUploading] = useState(false)
  const [upgradeOpen, setUpgradeOpen] = useState(false)

  const canCreate = useHasPermission("proposals.create")
  const user = useUser()
  // Subscription is provider-only. `enterprise` users never see the
  // FeePreview so the CTA role here can be derived solely from the two
  // prestataire roles; default to freelance when the role hasn't loaded
  // yet (UpgradeModal will hide if `viewer_is_subscribed` is already
  // true, so the default is safe).
  const subscriptionRole: "freelance" | "agency" =
    user.data?.role === "agency" ? "agency" : "freelance"
  const monthlyPrice = subscriptionRole === "agency" ? 49 : 19
  const createMutation = useCreateProposal()
  const modifyMutation = useModifyProposal()
  const isSubmitting = createMutation.isPending || modifyMutation.isPending || isUploading

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

  // Sync query params into form data when they change. Tracking the
  // previous recipient/conversation pair in render-time state lets us
  // mirror the URL into form state without a setState-in-effect cascade.
  const [lastQueryKey, setLastQueryKey] = useState(
    `${recipientId}|${conversationId}`,
  )
  const queryKey = `${recipientId}|${conversationId}`
  if (queryKey !== lastQueryKey) {
    setLastQueryKey(queryKey)
    setFormData((prev) => ({ ...prev, recipientId, conversationId }))
    // Mock recipient name derived synchronously from the new id so it
    // stays in sync with the form's recipientId.
    if (recipientId) {
      setRecipientName(`User ${recipientId.slice(0, 8)}`)
    }
  }

  const updateField = useCallback(<K extends keyof ProposalFormData>(
    field: K,
    value: ProposalFormData[K],
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }, [])

  const isValid = (() => {
    if (formData.title.trim().length === 0) return false
    if (formData.description.trim().length === 0) return false
    if (formData.paymentMode === "milestone") {
      // Milestone mode: at least one milestone with a positive
      // amount, and every milestone must have a title and a
      // description (the backend domain enforces both).
      if (formData.milestones.length === 0) return false
      for (const m of formData.milestones) {
        if (m.title.trim().length === 0) return false
        if (m.description.trim().length === 0) return false
        if (Number(m.amount) <= 0) return false
      }
      if (sumMilestoneAmounts(formData.milestones) <= 0) return false
      // Block submission while the deadline sequence is invalid —
      // mirrors the backend's strict-after rule. The MilestoneEditor
      // surfaces the per-row error inline so the user can see WHICH
      // row is wrong, but here we just need a global yes/no.
      const dlErrors = validateMilestoneDeadlines(
        formData.milestones,
        formData.deadline || undefined,
      )
      if (Object.keys(dlErrors).length > 0) return false
      return true
    }
    // One-time mode: a single positive amount.
    return Number(formData.amount) > 0
  })()

  async function uploadFiles(files: File[]) {
    const uploaded: NonNullable<CreateProposalData["documents"]> = []
    for (const file of files) {
      const { upload_url, public_url } = await getUploadURL(file.name, file.type || "application/octet-stream")
      await fetch(upload_url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type || "application/octet-stream" },
      })
      uploaded.push({
        filename: file.name,
        url: public_url,
        size: file.size,
        mime_type: file.type || "application/octet-stream",
      })
    }
    return uploaded
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!isValid || isSubmitting) return
    setSubmitError(null)

    let documents: CreateProposalData["documents"]

    // Upload files to storage before submitting the proposal
    if (formData.files.length > 0) {
      setIsUploading(true)
      try {
        documents = await uploadFiles(formData.files)
      } catch {
        setSubmitError(t("uploadError"))
        setIsUploading(false)
        return
      }
      setIsUploading(false)
    }

    // Build the API payload depending on the payment mode. In
    // milestone mode the amount field is derived from the milestone
    // sum and the milestones array is sent verbatim; in one-time
    // mode a single amount is sent and the backend synthesises a
    // single milestone server-side.
    const payload = buildCreatePayload(formData)

    if (modifyId) {
      modifyMutation.mutate(
        {
          id: modifyId,
          data: {
            title: formData.title.trim(),
            description: formData.description.trim(),
            amount: payload.amount,
            deadline: formData.deadline || undefined,
            documents,
            payment_mode: formData.paymentMode,
            milestones: payload.milestones,
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
          amount: payload.amount,
          deadline: formData.deadline || undefined,
          documents,
          payment_mode: formData.paymentMode,
          milestones: payload.milestones,
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
          disabled={!isValid || isSubmitting || !canCreate}
          className={cn(
            "rounded-xl px-5 py-2 text-sm font-semibold text-white transition-all duration-200",
            "flex items-center gap-2",
            isValid && !isSubmitting && canCreate
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

            {/* Payment mode toggle (phase 10) */}
            <PaymentModeToggle
              value={formData.paymentMode}
              onChange={(mode) => updateField("paymentMode", mode)}
              disabled={isSubmitting}
            />

            {formData.paymentMode === "one_time" ? (
              /* One-time mode: single amount field */
              <div className="space-y-2" id="payment-mode-panel-one_time">
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
            ) : (
              /* Milestone mode: repeatable editor */
              <MilestoneEditor
                milestones={formData.milestones}
                onChange={(milestones: MilestoneFormItem[]) =>
                  updateField("milestones", milestones)
                }
                disabled={isSubmitting}
                projectDeadline={formData.deadline || undefined}
              />
            )}

            {/* Platform fee preview — prestataire-only. The FeePreview
                component itself hides the section when the backend says
                `viewer_is_provider=false` (client-side viewers), so no
                role logic is needed here. We pass the recipient id so
                the backend can resolve the pair's roles, and an
                `UpgradeCta` that opens the subscription modal so
                prospects can convert from the exact moment they see
                the fee. The FeePreview only renders the CTA when
                `viewer_is_subscribed=false`. */}
            <FeePreview
              mode={formData.paymentMode}
              milestones={buildFeePreviewMilestones(formData)}
              recipientId={recipientId || undefined}
              renderPremiumCta={
                <UpgradeCta
                  variant="inline"
                  onClick={() => setUpgradeOpen(true)}
                  monthlyPrice={monthlyPrice}
                />
              }
            />

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
                disabled={!isValid || isSubmitting || !canCreate}
                className={cn(
                  "flex-1 rounded-xl px-5 py-2.5 text-sm font-semibold text-white transition-all duration-200",
                  "flex items-center justify-center gap-2",
                  isValid && !isSubmitting && canCreate
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
      <UpgradeModal
        open={upgradeOpen}
        role={subscriptionRole}
        onClose={() => setUpgradeOpen(false)}
      />
    </div>
  )
}

// buildCreatePayload turns the form state into the API payload shape.
// In one-time mode it forwards a single amount and leaves milestones
// undefined (the backend synthesises a single milestone server-side).
// In milestone mode it derives the total amount from the sum of
// milestone amounts and serialises the array with consecutive
// sequence numbers starting at 1.
function buildCreatePayload(form: ProposalFormData): {
  amount: number
  milestones: MilestoneInputData[] | undefined
} {
  if (form.paymentMode === "one_time") {
    const amount = Math.round(Number(form.amount) * 100)
    return { amount, milestones: undefined }
  }

  const milestones: MilestoneInputData[] = form.milestones.map((m, idx) => ({
    sequence: idx + 1,
    title: m.title.trim(),
    description: m.description.trim(),
    amount: Math.round(Number(m.amount) * 100),
    deadline: m.deadline || undefined,
  }))
  const amount = milestones.reduce((sum, m) => sum + m.amount, 0)
  return { amount, milestones }
}

// buildFeePreviewMilestones maps the current form state to the shape
// expected by <FeePreview>. In one-time mode we emit a single synthetic
// entry labelled "Paiement unique" so the component's milestone-agnostic
// one-time summary fires; in milestone mode each editable item is
// forwarded with its display label.
function buildFeePreviewMilestones(
  form: ProposalFormData,
): { key: string; label: string; amountCents: number }[] {
  if (form.paymentMode === "one_time") {
    const amountCents = Math.round(Number(form.amount) * 100)
    return [
      {
        key: "one-time",
        label: "Paiement unique",
        amountCents: Number.isFinite(amountCents) ? amountCents : 0,
      },
    ]
  }
  return form.milestones.map((m, idx) => {
    const parsed = Math.round(Number(m.amount) * 100)
    return {
      key: `milestone-${idx}`,
      label: m.title.trim() || `Jalon ${idx + 1}`,
      amountCents: Number.isFinite(parsed) ? parsed : 0,
    }
  })
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
