"use client"

import { useState } from "react"
import { AlertTriangle, Loader2, Paperclip, FileText, X } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"
import { FileUploadModal } from "@/shared/components/file-upload-modal"
import { uploadFiles } from "@/shared/lib/upload"
import { useOpenDispute } from "../hooks/use-disputes"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
import { Select } from "@/shared/components/ui/select"
interface DisputeFormProps {
  proposalId: string
  proposalAmount: number
  userRole: "client" | "provider"
  onSuccess: () => void
  onCancel: () => void
}

const CLIENT_REASONS = ["work_not_conforming", "non_delivery", "insufficient_quality", "other"] as const
const PROVIDER_REASONS = ["client_ghosting", "scope_creep", "refusal_to_validate", "harassment", "other"] as const

export function DisputeForm({ proposalId, proposalAmount, userRole, onSuccess, onCancel }: DisputeFormProps) {
  const t = useTranslations("disputes")
  const mutation = useOpenDispute()

  const [reason, setReason] = useState("")
  const [messageToParty, setMessageToParty] = useState("")
  const [description, setDescription] = useState("")
  const [amountType, setAmountType] = useState<"total" | "partial">("total")
  const [partialAmount, setPartialAmount] = useState(0)

  // File upload modals — store actual File objects so we can upload on submit
  const [partyModalOpen, setPartyModalOpen] = useState(false)
  const [mediationModalOpen, setMediationModalOpen] = useState(false)
  const [partyFiles, setPartyFiles] = useState<File[]>([])
  const [mediationFiles, setMediationFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)

  const reasons = userRole === "client" ? CLIENT_REASONS : PROVIDER_REASONS
  const requestedAmount = amountType === "total" ? proposalAmount : partialAmount

  async function handlePartyUpload(files: File[]) {
    setPartyFiles((prev) => [...prev, ...files])
    setPartyModalOpen(false)
  }

  async function handleMediationUpload(files: File[]) {
    setMediationFiles((prev) => [...prev, ...files])
    setMediationModalOpen(false)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!reason || !messageToParty) return

    setUploading(true)
    try {
      // Upload all files (party + mediation) and collect metadata
      const allFiles = [...partyFiles, ...mediationFiles]
      const attachments = allFiles.length > 0 ? await uploadFiles(allFiles) : []

      mutation.mutate(
        {
          proposal_id: proposalId,
          reason,
          description,
          message_to_party: messageToParty,
          requested_amount: requestedAmount,
          attachments,
        },
        { onSuccess },
      )
    } finally {
      setUploading(false)
    }
  }

  return (
    <>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div className="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-3 dark:border-amber-500/20 dark:bg-amber-500/10">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-500" aria-hidden />
          <p className="text-sm text-amber-700 dark:text-amber-400">{t("formWarning")}</p>
        </div>

        {/* Reason */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">
            {t("reasonLabel")}
          </label>
          <Select
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            required
            className="h-10 w-full rounded-lg border border-border bg-card px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10"
          >
            <option value="">{t("reasonPlaceholder")}</option>
            {reasons.map((r) => (
              <option key={r} value={r}>{t(`reason.${r}`)}</option>
            ))}
          </Select>
        </div>

        {/* Amount */}
        <div>
          <label
            htmlFor="dispute-amount-slider"
            className="mb-1.5 block text-sm font-medium text-foreground"
          >
            {t("amountLabel")}
          </label>
          <div className="space-y-3">
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-sm font-medium text-foreground">
                {amountType === "total"
                  ? userRole === "client"
                    ? t("totalRefund", { amount: formatEur(proposalAmount) })
                    : t("totalRelease", { amount: formatEur(proposalAmount) })
                  : `${t("partialAmount")} (${formatEur(partialAmount)})`}
              </span>
              <span className="text-xs text-muted-foreground">
                / {formatEur(proposalAmount)}
              </span>
            </div>
            <input
              id="dispute-amount-slider"
              type="range"
              min={1}
              max={proposalAmount}
              step={1}
              value={amountType === "total" ? proposalAmount : partialAmount || 1}
              onChange={(e) => {
                const value = Number(e.target.value)
                if (value >= proposalAmount) {
                  setAmountType("total")
                  setPartialAmount(0)
                } else {
                  setAmountType("partial")
                  setPartialAmount(value)
                }
              }}
              aria-label={t("amountLabel")}
              className={cn(
                "h-2 w-full cursor-pointer appearance-none rounded-full bg-border-strong outline-none",
                "accent-primary",
                "focus-visible:ring-2 focus-visible:ring-primary/40",
                // Webkit thumb
                "[&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:h-5 [&::-webkit-slider-thumb]:w-5",
                "[&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-primary [&::-webkit-slider-thumb]:shadow-md",
                "[&::-webkit-slider-thumb]:cursor-grab active:[&::-webkit-slider-thumb]:cursor-grabbing",
                // Firefox thumb
                "[&::-moz-range-thumb]:h-5 [&::-moz-range-thumb]:w-5 [&::-moz-range-thumb]:rounded-full",
                "[&::-moz-range-thumb]:bg-primary [&::-moz-range-thumb]:border-0 [&::-moz-range-thumb]:shadow-md",
                "[&::-moz-range-thumb]:cursor-grab",
              )}
            />
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min={1}
                max={proposalAmount / 100}
                value={Math.round((amountType === "total" ? proposalAmount : partialAmount) / 100) || ""}
                onChange={(e) => {
                  const cents = Math.round(Number(e.target.value) * 100)
                  if (cents >= proposalAmount) {
                    setAmountType("total")
                    setPartialAmount(0)
                  } else {
                    setAmountType("partial")
                    setPartialAmount(Math.max(1, cents))
                  }
                }}
                className="h-9 w-32 rounded-lg border border-border bg-card px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10"
                placeholder="0"
              />
              <span className="text-sm text-muted-foreground">EUR</span>
            </div>
          </div>
        </div>

        {/* Message to the other party */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">
            {t("messageToPartyLabel")}
          </label>
          <p className="mb-2 text-xs text-muted-foreground">{t("messageToPartyHint")}</p>
          <textarea
            value={messageToParty}
            onChange={(e) => setMessageToParty(e.target.value)}
            required
            maxLength={2000}
            rows={3}
            placeholder={t("messageToPartyPlaceholder")}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10"
          />
          <FileChips files={partyFiles} onRemove={(i) => setPartyFiles((f) => f.filter((_, j) => j !== i))} />
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => setPartyModalOpen(true)}
            className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <Paperclip className="h-3.5 w-3.5" />
            {t("addFiles")}
          </Button>
        </div>

        {/* Detailed description for admin mediation */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">
            {t("descriptionLabel")}
          </label>
          <p className="mb-2 text-xs text-muted-foreground">{t("descriptionHint")}</p>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            maxLength={5000}
            rows={4}
            placeholder={t("descriptionPlaceholder")}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10"
          />
          <div className="flex items-center justify-between mt-1">
            <div>
              <FileChips files={mediationFiles} onRemove={(i) => setMediationFiles((f) => f.filter((_, j) => j !== i))} />
              <Button variant="ghost" size="auto"
                type="button"
                onClick={() => setMediationModalOpen(true)}
                className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                <Paperclip className="h-3.5 w-3.5" />
                {t("addFiles")}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">{description.length}/5000</p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onCancel}
            className="rounded-lg border border-border-strong px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-muted transition-colors"
          >
            {t("cancelBtn")}
          </Button>
          <Button variant="ghost" size="auto"
            type="submit"
            disabled={!reason || !messageToParty || mutation.isPending || uploading}
            className={cn(
              "flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-all",
              "bg-red-600 hover:bg-red-700 active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {(mutation.isPending || uploading) && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("submitDispute")}
          </Button>
        </div>

        {mutation.isError && (
          <p className="text-sm text-red-500">{t("submitError")}</p>
        )}
      </form>

      {/* File upload modals */}
      <FileUploadModal
        open={partyModalOpen}
        onClose={() => setPartyModalOpen(false)}
        onUploadFiles={handlePartyUpload}
        uploading={false}
      />
      <FileUploadModal
        open={mediationModalOpen}
        onClose={() => setMediationModalOpen(false)}
        onUploadFiles={handleMediationUpload}
        uploading={false}
      />
    </>
  )
}

// ---------------------------------------------------------------------------
// File chips display
// ---------------------------------------------------------------------------

function FileChips({ files, onRemove }: { files: File[]; onRemove: (index: number) => void }) {
  if (files.length === 0) return null
  return (
    <div className="flex flex-wrap gap-2 mt-2">
      {files.map((f, i) => (
        <div
          key={`${f.name}-${i}`}
          className="flex items-center gap-1.5 rounded-lg border border-border bg-muted px-2.5 py-1 text-xs"
        >
          <FileText className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="max-w-[150px] truncate text-muted-foreground">{f.name}</span>
          <span className="text-muted-foreground">({(f.size / 1024).toFixed(0)} KB)</span>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => onRemove(i)}
            className="ml-0.5 text-muted-foreground hover:text-red-500 transition-colors"
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      ))}
    </div>
  )
}

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}
