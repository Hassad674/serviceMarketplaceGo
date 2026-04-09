"use client"

import { useState } from "react"
import { AlertTriangle, Loader2, Paperclip, FileText, X } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"
import { FileUploadModal } from "@/shared/components/file-upload-modal"
import { uploadFiles } from "@/shared/lib/upload"
import { useOpenDispute } from "../hooks/use-disputes"

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
          <label className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("reasonLabel")}
          </label>
          <select
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            required
            className="h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm shadow-xs focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-slate-700 dark:bg-slate-800"
          >
            <option value="">{t("reasonPlaceholder")}</option>
            {reasons.map((r) => (
              <option key={r} value={r}>{t(`reason.${r}`)}</option>
            ))}
          </select>
        </div>

        {/* Amount */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("amountLabel")}
          </label>
          <div className="flex flex-col gap-2">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="amountType"
                checked={amountType === "total"}
                onChange={() => setAmountType("total")}
                className="text-rose-500"
              />
              {userRole === "client"
                ? t("totalRefund", { amount: formatEur(proposalAmount) })
                : t("totalRelease", { amount: formatEur(proposalAmount) })}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="amountType"
                checked={amountType === "partial"}
                onChange={() => setAmountType("partial")}
                className="text-rose-500"
              />
              {t("partialAmount")}
            </label>
            {amountType === "partial" && (
              <div className="ml-6 flex items-center gap-2">
                <input
                  type="number"
                  min={1}
                  max={proposalAmount / 100}
                  value={partialAmount / 100 || ""}
                  onChange={(e) => setPartialAmount(Math.round(Number(e.target.value) * 100))}
                  className="h-9 w-32 rounded-lg border border-slate-200 px-3 text-sm shadow-xs focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-slate-700 dark:bg-slate-800"
                  placeholder="0"
                />
                <span className="text-sm text-slate-500">EUR</span>
              </div>
            )}
          </div>
        </div>

        {/* Message to the other party */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("messageToPartyLabel")}
          </label>
          <p className="mb-2 text-xs text-slate-400">{t("messageToPartyHint")}</p>
          <textarea
            value={messageToParty}
            onChange={(e) => setMessageToParty(e.target.value)}
            required
            maxLength={2000}
            rows={3}
            placeholder={t("messageToPartyPlaceholder")}
            className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm shadow-xs focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-slate-700 dark:bg-slate-800"
          />
          <FileChips files={partyFiles} onRemove={(i) => setPartyFiles((f) => f.filter((_, j) => j !== i))} />
          <button
            type="button"
            onClick={() => setPartyModalOpen(true)}
            className="mt-1 flex items-center gap-1.5 text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 transition-colors"
          >
            <Paperclip className="h-3.5 w-3.5" />
            {t("addFiles")}
          </button>
        </div>

        {/* Detailed description for admin mediation */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("descriptionLabel")}
          </label>
          <p className="mb-2 text-xs text-slate-400">{t("descriptionHint")}</p>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            maxLength={5000}
            rows={4}
            placeholder={t("descriptionPlaceholder")}
            className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm shadow-xs focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-slate-700 dark:bg-slate-800"
          />
          <div className="flex items-center justify-between mt-1">
            <div>
              <FileChips files={mediationFiles} onRemove={(i) => setMediationFiles((f) => f.filter((_, j) => j !== i))} />
              <button
                type="button"
                onClick={() => setMediationModalOpen(true)}
                className="mt-1 flex items-center gap-1.5 text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 transition-colors"
              >
                <Paperclip className="h-3.5 w-3.5" />
                {t("addFiles")}
              </button>
            </div>
            <p className="text-xs text-slate-400">{description.length}/5000</p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-2">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 transition-colors dark:border-slate-600 dark:text-slate-400"
          >
            {t("cancelBtn")}
          </button>
          <button
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
          </button>
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
          className="flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs dark:border-slate-700 dark:bg-slate-800"
        >
          <FileText className="h-3.5 w-3.5 text-slate-400" />
          <span className="max-w-[150px] truncate text-slate-600 dark:text-slate-400">{f.name}</span>
          <span className="text-slate-400">({(f.size / 1024).toFixed(0)} KB)</span>
          <button
            type="button"
            onClick={() => onRemove(i)}
            className="ml-0.5 text-slate-400 hover:text-red-500 transition-colors"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  )
}

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}
