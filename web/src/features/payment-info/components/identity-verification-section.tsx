"use client"

import { useState } from "react"
import {
  Shield,
  Upload,
  CheckCircle2,
  Clock,
  XCircle,
  Loader2,
  Pencil,
  AlertTriangle,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { UploadModal } from "@/shared/components/upload-modal"
import { useIdentityDocuments, useUploadIdentityDocument } from "../hooks/use-identity-documents"
import type { IdentityDocumentResponse } from "../api/identity-document-api"

const DOC_TYPES = [
  { value: "passport", labelKey: "passport", sides: ["single"] },
  { value: "id_card", labelKey: "id_card", sides: ["front", "back"] },
  { value: "driving_license", labelKey: "driving_license", sides: ["front", "back"] },
] as const

type UploadStep = { type: string; side: string; sideIndex: number; totalSides: number }

interface IdentityVerificationSectionProps {
  category?: "identity" | "company"
}

export function IdentityVerificationSection({ category = "identity" }: IdentityVerificationSectionProps) {
  const t = useTranslations("paymentInfo")
  const { data: docs, isLoading } = useIdentityDocuments()
  const uploadMutation = useUploadIdentityDocument()

  // Modal state
  const [modalOpen, setModalOpen] = useState(false)
  const [selectedType, setSelectedType] = useState<string | null>(null)
  const [currentStep, setCurrentStep] = useState<UploadStep | null>(null)

  const allDocs = docs ?? []
  const identityDocs = allDocs.filter((d) => d.category === category)
  const status = getOverallStatus(identityDocs)

  function openModal() {
    setSelectedType(null)
    setCurrentStep(null)
    setModalOpen(true)
  }

  function handleSelectType(type: string) {
    setSelectedType(type)
    const dt = DOC_TYPES.find((d) => d.value === type)
    if (!dt) return
    setCurrentStep({ type, side: dt.sides[0], sideIndex: 0, totalSides: dt.sides.length })
  }

  async function handleUpload(file: File) {
    if (!currentStep) return
    const step = currentStep
    const dt = DOC_TYPES.find((d) => d.value === step.type)
    const hasNext = dt && step.sideIndex + 1 < dt.sides.length

    if (hasNext) {
      // Advance to next side immediately, upload in background
      const nextIdx = step.sideIndex + 1
      setCurrentStep({
        type: step.type,
        side: dt.sides[nextIdx],
        sideIndex: nextIdx,
        totalSides: dt.sides.length,
      })
    } else {
      // Last/only side — close modal immediately
      setModalOpen(false)
      setCurrentStep(null)
      setSelectedType(null)
    }

    // Upload in background — UI already moved on
    uploadMutation.mutate(
      { file, category, documentType: step.type, side: step.side },
    )
  }

  if (isLoading) {
    return (
      <SectionShell category={category}>
        <div className="flex justify-center py-4">
          <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
        </div>
      </SectionShell>
    )
  }

  return (
    <SectionShell category={category}>
      {/* State: no documents uploaded yet */}
      {status === "none" && (
        <button
          type="button"
          onClick={openModal}
          className={cn(
            "w-full rounded-xl border-2 border-dashed border-slate-200 dark:border-slate-600 p-8",
            "flex flex-col items-center gap-3 transition-colors",
            "hover:border-rose-300 hover:bg-rose-50/50 dark:hover:border-rose-500/30 dark:hover:bg-rose-500/5",
          )}
        >
          <Upload className="h-10 w-10 text-slate-400" />
          <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
            {t("uploadDocument")}
          </span>
          <span className="text-xs text-slate-400">{t("uploadDocumentDesc")}</span>
        </button>
      )}

      {/* State: pending verification */}
      {status === "pending" && (
        <div className="flex items-center gap-3 rounded-xl bg-amber-50 dark:bg-amber-500/10 p-4">
          <Clock className="h-5 w-5 text-amber-600 dark:text-amber-400 shrink-0" />
          <div className="flex-1">
            <p className="text-sm font-medium text-amber-700 dark:text-amber-300">
              {t("documentPending")}
            </p>
            <p className="text-xs text-amber-600/70 dark:text-amber-400/70 mt-0.5">
              {t(identityDocs[0]?.document_type ?? "passport")} — {formatDate(identityDocs[0]?.created_at)}
            </p>
          </div>
        </div>
      )}

      {/* State: verified */}
      {status === "verified" && (
        <div className="flex items-center gap-3 rounded-xl bg-green-50 dark:bg-green-500/10 p-4">
          <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400 shrink-0" />
          <div className="flex-1">
            <p className="text-sm font-medium text-green-700 dark:text-green-300">
              {t("documentVerified")}
            </p>
            <p className="text-xs text-green-600/70 dark:text-green-400/70 mt-0.5">
              {t(identityDocs[0]?.document_type ?? "passport")} — {formatDate(identityDocs[0]?.updated_at)}
            </p>
          </div>
          <button
            type="button"
            onClick={openModal}
            className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-slate-500 hover:bg-green-100 dark:hover:bg-green-500/20 transition-colors"
          >
            <Pencil className="h-3.5 w-3.5" />
            {t("replaceDocument")}
          </button>
        </div>
      )}

      {/* State: rejected */}
      {status === "rejected" && (
        <div className="space-y-3">
          <div className="flex items-center gap-3 rounded-xl bg-red-50 dark:bg-red-500/10 p-4">
            <AlertTriangle className="h-5 w-5 text-red-600 dark:text-red-400 shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-red-700 dark:text-red-300">
                {t("documentRejected")}
              </p>
              {identityDocs[0]?.rejection_reason && (
                <p className="text-xs text-red-600/70 dark:text-red-400/70 mt-0.5">
                  {t("rejectionReason", { reason: identityDocs[0].rejection_reason })}
                </p>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={openModal}
            className={cn(
              "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
              "text-sm font-medium text-white transition-all",
              "gradient-primary hover:shadow-glow active:scale-[0.98]",
            )}
          >
            <Upload className="h-4 w-4" />
            {t("uploadDocument")}
          </button>
        </div>
      )}

      {/* Upload error */}
      {uploadMutation.isError && (
        <p className="text-sm text-red-500 text-center">
          {uploadMutation.error?.message || "Upload failed"}
        </p>
      )}

      {/* Modal: type selector + upload */}
      {modalOpen && !currentStep && (
        <ModalOverlay onClose={() => setModalOpen(false)}>
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-1">
            {t("documentType")}
          </h3>
          <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
            {t("uploadDocumentDesc")}
          </p>
          <div className="space-y-2">
            {DOC_TYPES.map((dt) => (
              <button
                key={dt.value}
                type="button"
                onClick={() => handleSelectType(dt.value)}
                className={cn(
                  "w-full flex items-center gap-3 rounded-xl border p-4 text-left transition-all",
                  "border-slate-200 dark:border-slate-600",
                  "hover:border-rose-300 hover:bg-rose-50/50 dark:hover:border-rose-500/30 dark:hover:bg-rose-500/5",
                )}
              >
                <Shield className="h-5 w-5 text-slate-400 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-white">{t(dt.labelKey)}</p>
                  <p className="text-xs text-slate-500">
                    {dt.sides.length === 1 ? t("singlePage") : t("frontAndBack")}
                  </p>
                </div>
              </button>
            ))}
          </div>
        </ModalOverlay>
      )}

      {/* Upload modal for file */}
      {currentStep && (
        <UploadModal
          open={true}
          onClose={() => { setCurrentStep(null); setModalOpen(false) }}
          onUpload={handleUpload}
          accept="image/*,application/pdf"
          maxSize={10 * 1024 * 1024}
          title={currentStep.totalSides > 1
            ? `${t(currentStep.type)} — ${currentStep.side === "front" ? t("frontSide") : t("backSide")} (${currentStep.sideIndex + 1}/${currentStep.totalSides})`
            : t(currentStep.type)
          }
          description={t("uploadDocumentDesc")}
        />
      )}
    </SectionShell>
  )
}

function SectionShell({ children, category = "identity" }: { children: React.ReactNode; category?: "identity" | "company" }) {
  const t = useTranslations("paymentInfo")
  const isCompany = category === "company"
  return (
    <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
      <div className="h-1 bg-gradient-to-r from-rose-500 to-purple-500" />
      <div className="p-6 space-y-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
            <Shield className="h-5 w-5 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
          </div>
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              {isCompany ? t("businessDocument") : t("identityVerification")}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {isCompany ? t("businessDocumentDesc") : t("identityVerificationDesc")}
            </p>
          </div>
        </div>
        {children}
      </div>
    </div>
  )
}

function ModalOverlay({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-6 w-full max-w-md mx-4 animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  )
}

function getOverallStatus(docs: IdentityDocumentResponse[]): "none" | "pending" | "verified" | "rejected" {
  if (docs.length === 0) return "none"
  if (docs.some((d) => d.status === "rejected")) return "rejected"
  if (docs.some((d) => d.status === "verified")) return "verified"
  return "pending"
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return ""
  return new Date(dateStr).toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" })
}
