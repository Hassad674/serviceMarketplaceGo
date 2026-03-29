"use client"

import { useState } from "react"
import {
  Shield,
  Upload,
  CheckCircle2,
  Clock,
  XCircle,
  Trash2,
  Loader2,
  FileText,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { UploadModal } from "@/shared/components/upload-modal"
import { useIdentityDocuments, useUploadIdentityDocument, useDeleteIdentityDocument } from "../hooks/use-identity-documents"
import type { IdentityDocumentResponse } from "../api/identity-document-api"

const DOC_TYPES = [
  { value: "passport", sides: ["single"] },
  { value: "id_card", sides: ["front", "back"] },
  { value: "driving_license", sides: ["front", "back"] },
] as const

export function IdentityVerificationSection() {
  const t = useTranslations("paymentInfo")
  const { data: docs, isLoading } = useIdentityDocuments()
  const [selectedType, setSelectedType] = useState<string>("passport")
  const [uploadSide, setUploadSide] = useState<string | null>(null)

  const uploadMutation = useUploadIdentityDocument()
  const deleteMutation = useDeleteIdentityDocument()

  const docType = DOC_TYPES.find((d) => d.value === selectedType)
  const existingDocs = docs ?? []

  function findDoc(side: string): IdentityDocumentResponse | undefined {
    return existingDocs.find(
      (d) => d.category === "identity" && d.document_type === selectedType && d.side === side,
    )
  }

  async function handleUpload(file: File) {
    if (!uploadSide) return
    uploadMutation.mutate(
      { file, category: "identity", documentType: selectedType, side: uploadSide },
      { onSuccess: () => setUploadSide(null) },
    )
  }

  return (
    <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
      <div className="h-1 bg-gradient-to-r from-rose-500 to-purple-500" />
      <div className="p-6 space-y-5">
        {/* Header */}
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
            <Shield className="h-5 w-5 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
          </div>
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              {t("identityVerification")}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t("identityVerificationDesc")}
            </p>
          </div>
        </div>

        {/* Document type selector */}
        <div>
          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2 block">
            {t("documentType")}
          </label>
          <div className="flex gap-2">
            {DOC_TYPES.map((dt) => (
              <button
                key={dt.value}
                type="button"
                onClick={() => setSelectedType(dt.value)}
                className={cn(
                  "rounded-lg px-4 py-2 text-sm font-medium transition-all",
                  selectedType === dt.value
                    ? "bg-rose-500 text-white shadow-sm"
                    : "bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600",
                )}
              >
                {t(dt.value)}
              </button>
            ))}
          </div>
        </div>

        {/* Upload slots */}
        {isLoading ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
          </div>
        ) : (
          <div className={cn("grid gap-4", docType?.sides.length === 2 ? "grid-cols-2" : "grid-cols-1")}>
            {docType?.sides.map((side) => {
              const doc = findDoc(side)
              const sideLabel = side === "single" ? t("documentType") : side === "front" ? t("frontSide") : t("backSide")
              return (
                <DocumentSlot
                  key={side}
                  label={sideLabel}
                  document={doc}
                  uploading={uploadMutation.isPending && uploadSide === side}
                  deleting={deleteMutation.isPending}
                  onUploadClick={() => setUploadSide(side)}
                  onDelete={() => doc && deleteMutation.mutate(doc.id)}
                />
              )
            })}
          </div>
        )}

        {/* Upload error */}
        {uploadMutation.isError && (
          <p className="text-sm text-red-500 text-center">
            {uploadMutation.error?.message || "Upload failed"}
          </p>
        )}
      </div>

      {/* Upload modal */}
      <UploadModal
        open={uploadSide !== null}
        onClose={() => setUploadSide(null)}
        onUpload={handleUpload}
        accept="image/*,application/pdf"
        maxSize={10 * 1024 * 1024}
        title={t("uploadDocumentTitle")}
        description={t("uploadDocumentDesc")}
      />
    </div>
  )
}

function DocumentSlot({ label, document, uploading, deleting, onUploadClick, onDelete }: {
  label: string
  document?: IdentityDocumentResponse
  uploading: boolean
  deleting: boolean
  onUploadClick: () => void
  onDelete: () => void
}) {
  const t = useTranslations("paymentInfo")

  if (document) {
    return (
      <div className="rounded-xl border border-slate-200 dark:border-slate-600 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{label}</span>
          <StatusBadge status={document.status} />
        </div>

        <div className="flex items-center gap-3">
          {document.file_url ? (
            <img
              src={document.file_url}
              alt={label}
              className="h-16 w-16 rounded-lg object-cover border border-slate-200 dark:border-slate-600"
              onError={(e) => { (e.target as HTMLImageElement).style.display = "none" }}
            />
          ) : (
            <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-700">
              <FileText className="h-6 w-6 text-slate-400" />
            </div>
          )}
          <div className="flex-1 min-w-0">
            <p className="text-xs text-slate-500 truncate">{document.id.slice(0, 12)}...</p>
            {document.rejection_reason && (
              <p className="text-xs text-red-500 mt-1">{t("rejectionReason", { reason: document.rejection_reason })}</p>
            )}
          </div>
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={onUploadClick}
            className="flex-1 flex items-center justify-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium border border-slate-200 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
          >
            <Upload className="h-3.5 w-3.5" />
            {t("replaceDocument")}
          </button>
          <button
            type="button"
            onClick={onDelete}
            disabled={deleting}
            className="flex items-center justify-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors"
          >
            {deleting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
          </button>
        </div>
      </div>
    )
  }

  return (
    <button
      type="button"
      onClick={onUploadClick}
      disabled={uploading}
      className={cn(
        "rounded-xl border-2 border-dashed border-slate-200 dark:border-slate-600 p-6",
        "flex flex-col items-center gap-2 transition-colors",
        "hover:border-rose-300 hover:bg-rose-50/50 dark:hover:border-rose-500/30 dark:hover:bg-rose-500/5",
        "disabled:opacity-50 disabled:cursor-not-allowed",
      )}
    >
      {uploading ? (
        <Loader2 className="h-8 w-8 animate-spin text-rose-500" />
      ) : (
        <Upload className="h-8 w-8 text-slate-400" />
      )}
      <span className="text-sm font-medium text-slate-600 dark:text-slate-400">{label}</span>
      <span className="text-xs text-slate-400">{t("uploadDocument")}</span>
    </button>
  )
}

function StatusBadge({ status }: { status: string }) {
  const t = useTranslations("paymentInfo")
  const configs: Record<string, { icon: React.ElementType; label: string; cls: string }> = {
    pending: { icon: Clock, label: t("documentPending"), cls: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400" },
    verified: { icon: CheckCircle2, label: t("documentVerified"), cls: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400" },
    rejected: { icon: XCircle, label: t("documentRejected"), cls: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400" },
  }
  const c = configs[status] ?? configs.pending
  const Icon = c.icon

  return (
    <span className={cn("inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium", c.cls)}>
      <Icon className="h-3 w-3" />
      {c.label}
    </span>
  )
}
