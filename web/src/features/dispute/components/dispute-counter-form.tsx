"use client"

import { useState } from "react"
import { Loader2, Paperclip, FileText, X } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/shared/lib/utils"
import { FileUploadModal } from "@/shared/components/file-upload-modal"
import { uploadFiles } from "@/shared/lib/upload"
import { useCounterPropose } from "../hooks/use-disputes"

import { Button } from "@/shared/components/ui/button"
interface DisputeCounterFormProps {
  disputeId: string
  proposalAmount: number
  onSuccess: () => void
  onCancel: () => void
}

export function DisputeCounterForm({ disputeId, proposalAmount, onSuccess, onCancel }: DisputeCounterFormProps) {
  const t = useTranslations("disputes")
  const mutation = useCounterPropose(disputeId)

  const [clientAmount, setClientAmount] = useState(0)
  const [message, setMessage] = useState("")
  const [files, setFiles] = useState<File[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [uploading, setUploading] = useState(false)

  const providerAmount = proposalAmount - clientAmount
  const isValid = clientAmount >= 0 && providerAmount >= 0

  function handleSlider(value: number) {
    setClientAmount(Math.round(value))
  }

  async function handleFiles(newFiles: File[]) {
    setFiles((prev) => [...prev, ...newFiles])
    setModalOpen(false)
  }

  function removeFile(index: number) {
    setFiles((prev) => prev.filter((_, i) => i !== index))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!isValid) return

    setUploading(true)
    try {
      const attachments = files.length > 0 ? await uploadFiles(files) : []
      mutation.mutate(
        {
          amount_client: clientAmount,
          amount_provider: providerAmount,
          message,
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
        {/* Slider */}
        <div>
          <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("counterSplitLabel")}
          </label>

          <input
            type="range"
            min={0}
            max={proposalAmount}
            step={100}
            value={clientAmount}
            onChange={(e) => handleSlider(Number(e.target.value))}
            className="w-full accent-rose-500"
          />

          <div className="mt-2 flex justify-between text-sm">
            <div className="text-slate-600 dark:text-slate-400">
              <span className="font-medium">{t("client")}:</span> {formatEur(clientAmount)}
            </div>
            <div className="text-slate-600 dark:text-slate-400">
              <span className="font-medium">{t("provider")}:</span> {formatEur(providerAmount)}
            </div>
          </div>
        </div>

        {/* Message */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("counterMessageLabel")}
          </label>
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            rows={3}
            maxLength={2000}
            placeholder={t("counterMessagePlaceholder")}
            className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm shadow-xs focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-slate-700 dark:bg-slate-800"
          />

          {/* File chips + add button */}
          {files.length > 0 && (
            <div className="flex flex-wrap gap-2 mt-2">
              {files.map((f, i) => (
                <div
                  key={`${f.name}-${i}`}
                  className="flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs dark:border-slate-700 dark:bg-slate-800"
                >
                  <FileText className="h-3.5 w-3.5 text-slate-400" />
                  <span className="max-w-[150px] truncate text-slate-600 dark:text-slate-400">{f.name}</span>
                  <span className="text-slate-400">({(f.size / 1024).toFixed(0)} KB)</span>
                  <Button variant="ghost" size="auto"
                    type="button"
                    onClick={() => removeFile(i)}
                    className="ml-0.5 text-slate-400 hover:text-red-500 transition-colors"
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ))}
            </div>
          )}
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => setModalOpen(true)}
            className="mt-2 flex items-center gap-1.5 text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 transition-colors"
          >
            <Paperclip className="h-3.5 w-3.5" />
            {t("addFiles")}
          </Button>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onCancel}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 transition-colors dark:border-slate-600 dark:text-slate-400"
          >
            {t("cancelBtn")}
          </Button>
          <Button variant="ghost" size="auto"
            type="submit"
            disabled={!isValid || mutation.isPending || uploading}
            className={cn(
              "flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-all",
              "bg-amber-600 hover:bg-amber-700 active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {(mutation.isPending || uploading) && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("submitCounter")}
          </Button>
        </div>
      </form>

      <FileUploadModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onUploadFiles={handleFiles}
        uploading={false}
      />
    </>
  )
}

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(centimes / 100)
}
