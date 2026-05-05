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
import { FeePreview } from "@/shared/components/billing/fee-preview"
import { UpgradeCta } from "@/shared/components/subscription/upgrade-cta"
import { UpgradeModal } from "@/shared/components/subscription/upgrade-modal"
import { useUser } from "@/shared/hooks/use-user"
import { Portrait } from "@/shared/components/ui/portrait"
import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

// Soleil v2 — Proposal creation page (W-09 equivalent for proposals).
// Editorial header (corail eyebrow + Fraunces italic-corail title + tabac
// subtitle), Soleil card sections, Fraunces section heads, ivoire
// inputs with corail focus ring, corail rounded-full pill submit.

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
  const subscriptionRole: "freelance" | "agency" =
    user.data?.role === "agency" ? "agency" : "freelance"
  const monthlyPrice = subscriptionRole === "agency" ? 49 : 19
  const createMutation = useCreateProposal()
  const modifyMutation = useModifyProposal()
  const isSubmitting =
    createMutation.isPending || modifyMutation.isPending || isUploading

  // Pre-fill form when modifying an existing proposal
  useEffect(() => {
    if (!modifyId) return
    getProposal(modifyId)
      .then((p) => {
        setFormData((prev) => ({
          ...prev,
          title: p.title,
          description: p.description,
          amount: (p.amount / 100).toString(),
          deadline: p.deadline ? p.deadline.split("T")[0] : "",
        }))
      })
      .catch(() => {
        setSubmitError(t("fetchError"))
      })
  }, [modifyId, t])

  // Sync query params into form data when they change.
  const [lastQueryKey, setLastQueryKey] = useState(
    `${recipientId}|${conversationId}`,
  )
  const queryKey = `${recipientId}|${conversationId}`
  if (queryKey !== lastQueryKey) {
    setLastQueryKey(queryKey)
    setFormData((prev) => ({ ...prev, recipientId, conversationId }))
    if (recipientId) {
      setRecipientName(`User ${recipientId.slice(0, 8)}`)
    }
  }

  const updateField = useCallback(
    <K extends keyof ProposalFormData>(field: K, value: ProposalFormData[K]) => {
      setFormData((prev) => ({ ...prev, [field]: value }))
    },
    [],
  )

  const isValid = (() => {
    if (formData.title.trim().length === 0) return false
    if (formData.description.trim().length === 0) return false
    if (formData.paymentMode === "milestone") {
      if (formData.milestones.length === 0) return false
      for (const m of formData.milestones) {
        if (m.title.trim().length === 0) return false
        if (m.description.trim().length === 0) return false
        if (Number(m.amount) <= 0) return false
      }
      if (sumMilestoneAmounts(formData.milestones) <= 0) return false
      const dlErrors = validateMilestoneDeadlines(
        formData.milestones,
        formData.deadline || undefined,
      )
      if (Object.keys(dlErrors).length > 0) return false
      return true
    }
    return Number(formData.amount) > 0
  })()

  async function uploadFiles(files: File[]) {
    const uploaded: NonNullable<CreateProposalData["documents"]> = []
    for (const file of files) {
      const { upload_url, public_url } = await getUploadURL(
        file.name,
        file.type || "application/octet-stream",
      )
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

  const eyebrowKey = "proposalFlow_create_eyebrow"
  const titlePrefixKey = "proposalFlow_create_titlePrefix"
  const titleAccentKey = modifyId
    ? "proposalFlow_create_modify_titleAccent"
    : "proposalFlow_create_titleAccent"
  const subtitleKey = modifyId
    ? "proposalFlow_create_modify_subtitle"
    : "proposalFlow_create_subtitle"

  return (
    <div className="min-h-screen bg-background">
      {/* Soleil top bar */}
      <header
        className={cn(
          "sticky top-0 z-10 flex h-16 items-center justify-between gap-4 border-b border-border px-4 sm:px-6",
          "glass-strong",
        )}
      >
        <Button
          variant="ghost"
          size="auto"
          type="button"
          onClick={handleCancel}
          className={cn(
            "rounded-full p-2 text-subtle-foreground transition-colors duration-150",
            "hover:bg-primary-soft hover:text-primary",
          )}
          aria-label={t("proposalCancel")}
        >
          <X className="h-5 w-5" strokeWidth={1.7} aria-hidden="true" />
        </Button>

        <h1 className="truncate font-serif text-[16px] font-medium text-foreground">
          {modifyId ? t("modify") : t("createProposal")}
        </h1>

        <Button
          variant="ghost"
          size="auto"
          type="submit"
          form="proposal-form"
          disabled={!isValid || isSubmitting || !canCreate}
          className={cn(
            "inline-flex items-center gap-2 rounded-full px-5 py-2.5 text-[13.5px] font-bold",
            "transition-all duration-200 ease-out",
            isValid && !isSubmitting && canCreate
              ? "bg-primary text-primary-foreground hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]"
              : "cursor-not-allowed bg-border text-subtle-foreground",
          )}
        >
          {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
          {t("proposalSend")}
        </Button>
      </header>

      {/* Body */}
      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Editorial header */}
        <div className="mb-8 max-w-2xl space-y-2">
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
            {t(eyebrowKey)}
          </p>
          <h2 className="font-serif text-[28px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[36px]">
            {t(titlePrefixKey)}{" "}
            <span className="italic text-primary">{t(titleAccentKey)}</span>
          </h2>
          <p className="text-[14.5px] leading-relaxed text-muted-foreground">
            {t(subtitleKey)}
          </p>
        </div>

        {submitError && (
          <div
            role="alert"
            className="mb-6 rounded-2xl border border-destructive/40 bg-primary-soft px-4 py-3 text-[13.5px] text-destructive"
          >
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
            {/* Brief section */}
            <FormSection eyebrow={t("proposalFlow_create_sectionBrief")}>
              <RecipientField name={recipientName} />

              <div className="space-y-2">
                <Label htmlFor="proposal-title" required>
                  {t("proposalTitle")}
                </Label>
                <div className="relative">
                  <Input
                    id="proposal-title"
                    type="text"
                    value={formData.title}
                    onChange={(e) =>
                      updateField("title", e.target.value.slice(0, TITLE_MAX_LENGTH))
                    }
                    placeholder={t("proposalTitlePlaceholder")}
                    maxLength={TITLE_MAX_LENGTH}
                    className={cn(
                      "h-11 w-full rounded-xl border border-border bg-card px-4 pr-16 text-[14.5px]",
                      "transition-all duration-200 ease-out",
                      "placeholder:text-subtle-foreground",
                      "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
                    )}
                    aria-required="true"
                  />
                  <span className="absolute right-3 top-1/2 -translate-y-1/2 font-mono text-[11px] text-subtle-foreground">
                    {formData.title.length}/{TITLE_MAX_LENGTH}
                  </span>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="proposal-description" required>
                  {t("proposalDescription")}
                </Label>
                <textarea
                  id="proposal-description"
                  value={formData.description}
                  onChange={(e) => updateField("description", e.target.value)}
                  placeholder={t("proposalDescriptionPlaceholder")}
                  rows={5}
                  className={cn(
                    "w-full rounded-xl border border-border bg-card px-4 py-3 text-[14.5px] resize-none",
                    "transition-all duration-200 ease-out",
                    "placeholder:text-subtle-foreground",
                    "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
                  )}
                  aria-required="true"
                />
              </div>
            </FormSection>

            {/* Payment section */}
            <FormSection eyebrow={t("proposalFlow_create_sectionPayment")}>
              <PaymentModeToggle
                value={formData.paymentMode}
                onChange={(mode) => updateField("paymentMode", mode)}
                disabled={isSubmitting}
              />

              {formData.paymentMode === "one_time" ? (
                <div className="space-y-2" id="payment-mode-panel-one_time">
                  <Label htmlFor="proposal-amount" required>
                    {t("proposalAmount")}
                  </Label>
                  <div className="relative">
                    <span className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 font-mono text-[14px] font-medium text-subtle-foreground">
                      &euro;
                    </span>
                    <Input
                      id="proposal-amount"
                      type="number"
                      min="0"
                      step="0.01"
                      value={formData.amount}
                      onChange={(e) => updateField("amount", e.target.value)}
                      placeholder={t("proposalAmountPlaceholder")}
                      className={cn(
                        "h-11 w-full rounded-xl border border-border bg-card pl-9 pr-4 text-[14.5px] font-mono",
                        "transition-all duration-200 ease-out",
                        "placeholder:text-subtle-foreground",
                        "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
                        "[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none",
                      )}
                      aria-required="true"
                    />
                  </div>
                </div>
              ) : (
                <MilestoneEditor
                  milestones={formData.milestones}
                  onChange={(milestones: MilestoneFormItem[]) =>
                    updateField("milestones", milestones)
                  }
                  disabled={isSubmitting}
                  projectDeadline={formData.deadline || undefined}
                />
              )}

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
            </FormSection>

            {/* Deadline section */}
            <FormSection eyebrow={t("proposalFlow_create_sectionDeadline")}>
              <div className="space-y-2">
                <Label htmlFor="proposal-deadline">{t("proposalDeadline")}</Label>
                <Input
                  id="proposal-deadline"
                  type="date"
                  value={formData.deadline}
                  onChange={(e) => updateField("deadline", e.target.value)}
                  min={new Date().toISOString().split("T")[0]}
                  className={cn(
                    "h-11 w-full rounded-xl border border-border bg-card px-4 text-[14.5px] font-mono",
                    "transition-all duration-200 ease-out text-foreground",
                    "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
                  )}
                />
              </div>
            </FormSection>

            {/* Documents section */}
            <FormSection eyebrow={t("proposalFlow_create_sectionDocuments")}>
              <div className="space-y-2">
                <Label>{t("proposalDocuments")}</Label>
                <FileDropZone
                  files={formData.files}
                  onFilesChange={(files) => updateField("files", files)}
                />
              </div>
            </FormSection>

            {/* Footer buttons (mobile only) */}
            <div className="flex gap-3 pt-4 lg:hidden">
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={handleCancel}
                className={cn(
                  "flex-1 rounded-full px-5 py-3 text-[13.5px] font-medium",
                  "border border-border-strong text-foreground transition-all duration-200 ease-out",
                  "hover:border-primary hover:bg-primary-soft hover:text-primary-deep",
                )}
              >
                {t("proposalCancel")}
              </Button>
              <Button
                variant="ghost"
                size="auto"
                type="submit"
                disabled={!isValid || isSubmitting || !canCreate}
                className={cn(
                  "flex-1 inline-flex items-center justify-center gap-2 rounded-full px-5 py-3",
                  "text-[13.5px] font-bold transition-all duration-200 ease-out",
                  isValid && !isSubmitting && canCreate
                    ? "bg-primary text-primary-foreground hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)] active:scale-[0.98]"
                    : "cursor-not-allowed bg-border text-subtle-foreground",
                )}
              >
                {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
                {t("proposalSend")}
              </Button>
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

interface FormSectionProps {
  eyebrow: string
  children: React.ReactNode
}

function FormSection({ eyebrow, children }: FormSectionProps) {
  return (
    <section
      className="space-y-5 rounded-2xl border border-border bg-card p-6"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.12em] text-primary">
        {eyebrow}
      </p>
      {children}
    </section>
  )
}

interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
  required?: boolean
}

function Label({ required, children, ...rest }: LabelProps) {
  return (
    <label
      {...rest}
      className={cn(
        "block text-[13.5px] font-medium text-foreground",
        rest.className,
      )}
    >
      {children}
      {required && <span className="ml-0.5 text-primary">*</span>}
    </label>
  )
}

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

  return (
    <div className="space-y-2">
      <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
        {t("proposalRecipient")}
      </p>
      <div className="flex items-center gap-3 rounded-2xl border border-border bg-background p-3.5">
        <Portrait id={2} size={36} />
        <p className="text-[14px] font-medium text-foreground">
          {name || "—"}
        </p>
      </div>
    </div>
  )
}
