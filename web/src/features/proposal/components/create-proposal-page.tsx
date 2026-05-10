"use client"

import { useState, useCallback, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { X, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import type { ProposalFormData, MilestoneFormItem, PaymentMode } from "../types"
import {
  MIN_MILESTONES_PER_MILESTONE_PROPOSAL,
  createEmptyProposalForm,
  ensureMinimumMilestones,
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
  const recipientNameParam = searchParams.get("name") ?? ""
  const modifyId = searchParams.get("modify")

  const [formData, setFormData] = useState<ProposalFormData>(() => ({
    ...createEmptyProposalForm(),
    recipientId,
    conversationId,
  }))
  const [recipientName, setRecipientName] = useState(recipientNameParam)
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

  // Pre-fill form when modifying an existing proposal. The fetched
  // proposal carries client_name/provider_name from the backend, so
  // we resolve the recipient name without ever falling back to the
  // raw user id.
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
        const resolvedName =
          recipientId === p.client_id ? p.client_name : p.provider_name
        if (resolvedName) {
          setRecipientName(resolvedName)
        }
      })
      .catch(() => {
        setSubmitError(t("fetchError"))
      })
  }, [modifyId, recipientId, t])

  // Sync query params into form data when they change. The displayed
  // recipient name comes from the `name` query param when available
  // (messaging entry points pass it explicitly); we only fall back to
  // a truncated id when no name is provided AND modify-mode is not
  // active (modify mode resolves the name from the fetched proposal).
  const [lastQueryKey, setLastQueryKey] = useState(
    `${recipientId}|${conversationId}|${recipientNameParam}`,
  )
  const queryKey = `${recipientId}|${conversationId}|${recipientNameParam}`
  if (queryKey !== lastQueryKey) {
    setLastQueryKey(queryKey)
    setFormData((prev) => ({ ...prev, recipientId, conversationId }))
    if (recipientNameParam) {
      setRecipientName(recipientNameParam)
    } else if (recipientId && !modifyId) {
      setRecipientName(`User ${recipientId.slice(0, 8)}`)
    }
  }

  const updateField = useCallback(
    <K extends keyof ProposalFormData>(field: K, value: ProposalFormData[K]) => {
      setFormData((prev) => ({ ...prev, [field]: value }))
    },
    [],
  )

  // Toggle handler for the payment-mode segmented control. When the
  // user picks "milestone" we top the milestones array up to
  // MIN_MILESTONES_PER_MILESTONE_PROPOSAL (mirrors Contra's UX: the
  // editor never starts below 2 empty rows). Switching back to
  // "one_time" leaves the milestone slots in state — toggling a third
  // time should not destroy what the user typed.
  const handlePaymentModeChange = useCallback(
    (mode: PaymentMode) => {
      setFormData((prev) => {
        if (mode === "milestone") {
          return {
            ...prev,
            paymentMode: mode,
            milestones: ensureMinimumMilestones(prev.milestones),
          }
        }
        return { ...prev, paymentMode: mode }
      })
    },
    [],
  )

  const isMilestoneMode = formData.paymentMode === "milestone"

  const isValid = (() => {
    if (isMilestoneMode) {
      // Milestone mode: NO global title/description/amount/deadline
      // inputs are shown — they are derived from the milestone list at
      // submit time. Validation focuses on the milestone slice itself.
      if (
        formData.milestones.length < MIN_MILESTONES_PER_MILESTONE_PROPOSAL
      ) {
        return false
      }
      for (const m of formData.milestones) {
        if (m.title.trim().length === 0) return false
        // Per-milestone description is OPTIONAL in milestone mode
        // (Contra-style — the project context is captured by the
        // milestone titles + amounts).
        if (Number(m.amount) <= 0) return false
      }
      if (sumMilestoneAmounts(formData.milestones) <= 0) return false
      const dlErrors = validateMilestoneDeadlines(
        formData.milestones,
        undefined,
      )
      if (Object.keys(dlErrors).length > 0) return false
      return true
    }
    // One-time mode: keep the legacy global-input contract (title,
    // description, positive amount).
    if (formData.title.trim().length === 0) return false
    if (formData.description.trim().length === 0) return false
    return Number(formData.amount) > 0
  })()

  async function uploadFiles(files: File[]) {
    const uploaded: NonNullable<CreateProposalData["documents"]> = []
    for (const file of files) {
      const { upload_url, public_url } = await getUploadURL(
        file.name,
        file.type || "application/octet-stream",
      )
      // fetch() does not throw on HTTP 4xx/5xx — without this guard a
      // failed upload would still push a broken URL into the proposal.
      const uploadRes = await fetch(upload_url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type || "application/octet-stream" },
      })
      if (!uploadRes.ok) {
        throw new Error(`upload failed: ${uploadRes.status}`)
      }
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
    // Title/description/deadline are derived from the milestones in
    // milestone mode (Contra-style: the global inputs are not shown).
    // The proposalTitleFallback / proposalDescriptionFallback i18n keys
    // are used when no milestone title is set yet — but isValid() has
    // already guaranteed every milestone has a title, so the fallback
    // path is only hit by tests.
    const submitTitle = isMilestoneMode
      ? deriveMilestoneTitle(formData, t("proposalTitleFallback"))
      : formData.title.trim()
    const submitDescription = isMilestoneMode
      ? deriveMilestoneDescription(formData)
      : formData.description.trim()
    const submitDeadline = isMilestoneMode
      ? latestMilestoneDeadlineString(formData.milestones)
      : formData.deadline || undefined

    if (modifyId) {
      modifyMutation.mutate(
        {
          id: modifyId,
          data: {
            title: submitTitle,
            description: submitDescription,
            amount: payload.amount,
            deadline: submitDeadline,
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
          title: submitTitle,
          description: submitDescription,
          amount: payload.amount,
          deadline: submitDeadline,
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
            {/* Brief section — recipient + payment-mode toggle come before
                title/description so the user picks the mode first.
                In milestone mode the global title + description inputs
                are HIDDEN: each milestone is its own self-contained
                step (Contra-style). The proposal-level title and
                description are derived from the milestones at submit
                time. */}
            <FormSection eyebrow={t("proposalFlow_create_sectionBrief")}>
              <RecipientField name={recipientName} />

              <PaymentModeToggle
                value={formData.paymentMode}
                onChange={handlePaymentModeChange}
                disabled={isSubmitting}
              />

              {!isMilestoneMode && (
                <>
                  <div className="space-y-2 min-w-0">
                    <Label htmlFor="proposal-title" required>
                      {t("proposalTitle")}
                    </Label>
                    <div className="relative min-w-0">
                      <Input
                        id="proposal-title"
                        type="text"
                        value={formData.title}
                        onChange={(e) =>
                          updateField(
                            "title",
                            e.target.value.slice(0, TITLE_MAX_LENGTH),
                          )
                        }
                        placeholder={t("proposalTitlePlaceholder")}
                        maxLength={TITLE_MAX_LENGTH}
                        className={cn(
                          "h-11 w-full min-w-0 rounded-xl border border-border bg-card px-4 pr-16 text-[14.5px] break-words",
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

                  <div className="space-y-2 min-w-0">
                    <Label htmlFor="proposal-description" required>
                      {t("proposalDescription")}
                    </Label>
                    <textarea
                      id="proposal-description"
                      value={formData.description}
                      onChange={(e) =>
                        updateField("description", e.target.value)
                      }
                      placeholder={t("proposalDescriptionPlaceholder")}
                      rows={5}
                      className={cn(
                        "w-full min-w-0 rounded-xl border border-border bg-card px-4 py-3 text-[14.5px] resize-none break-words",
                        "transition-all duration-200 ease-out",
                        "placeholder:text-subtle-foreground",
                        "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
                      )}
                      aria-required="true"
                    />
                  </div>
                </>
              )}
            </FormSection>

            {/* Payment section — mode-specific input(s) only. The toggle
                lives in the Brief section above. In milestone mode the
                global amount input is hidden (per-milestone amounts replace it). */}
            <FormSection eyebrow={t("proposalFlow_create_sectionPayment")}>
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

            {/* Deadline section — only in one_time mode. In milestone
                mode the project deadline is derived from the latest
                milestone due date (Contra-style). */}
            {!isMilestoneMode && (
              <FormSection
                eyebrow={t("proposalFlow_create_sectionDeadline")}
              >
                <div className="space-y-2">
                  <Label htmlFor="proposal-deadline">
                    {t("proposalDeadline")}
                  </Label>
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
            )}

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

// deriveMilestoneTitle synthesises a proposal-level title in milestone
// mode (where the global title input is hidden). Uses the first
// milestone title — by isValid() invariant every milestone has a title
// at submit time, but the fallback covers the test/edge case.
function deriveMilestoneTitle(form: ProposalFormData, fallback: string): string {
  for (const m of form.milestones) {
    const trimmed = m.title.trim()
    if (trimmed.length > 0) return trimmed
  }
  return fallback
}

// deriveMilestoneDescription synthesises a proposal-level description
// from the milestone titles. The backend stores the description on the
// proposal envelope (system message + audit + moderation); concatenating
// the milestone titles gives a meaningful summary without forcing the
// user to retype the brief.
function deriveMilestoneDescription(form: ProposalFormData): string {
  const titles = form.milestones
    .map((m) => m.title.trim())
    .filter((t) => t.length > 0)
  if (titles.length === 0) return ""
  return titles.map((t, i) => `${i + 1}. ${t}`).join("\n")
}

// latestMilestoneDeadlineString returns the latest YYYY-MM-DD value
// across the milestone slice, or undefined if no milestone has a
// deadline. Used to derive the proposal-level deadline server-side
// argument in milestone mode.
function latestMilestoneDeadlineString(
  milestones: MilestoneFormItem[],
): string | undefined {
  let latest: string | undefined
  for (const m of milestones) {
    if (!m.deadline) continue
    if (!latest || m.deadline > latest) {
      latest = m.deadline
    }
  }
  return latest
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
