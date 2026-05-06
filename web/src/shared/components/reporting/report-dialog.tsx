"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"
import { X, Flag, AlertCircle } from "lucide-react"
import { createPortal } from "react-dom"
import { useCreateReport } from "@/shared/hooks/reporting/use-report"
import {
  MESSAGE_REASONS,
  USER_REASONS,
  JOB_REASONS,
  APPLICATION_REASONS,
} from "@/shared/types/reporting"
import type { TargetType, ReportReason } from "@/shared/types/reporting"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
interface ReportDialogProps {
  open: boolean
  onClose: () => void
  targetType: TargetType
  targetId: string
  conversationId?: string
}

/**
 * Cross-feature report dialog (P9). Consumed by messaging (report a
 * message / user) and job (report a job posting / application). Lives
 * in `shared/components/reporting/` so neither feature has to import
 * from the reporting feature directly.
 */
export function ReportDialog({
  open,
  onClose,
  targetType,
  targetId,
  conversationId,
}: ReportDialogProps) {
  const t = useTranslations("reporting")
  const [reason, setReason] = useState<ReportReason | null>(null)
  const [description, setDescription] = useState("")
  const [alreadyReported, setAlreadyReported] = useState(false)
  const mutation = useCreateReport()

  if (!open) return null

  const reasonsByType: Record<TargetType, ReportReason[]> = {
    message: MESSAGE_REASONS,
    user: USER_REASONS,
    job: JOB_REASONS,
    application: APPLICATION_REASONS,
  }
  const reasons = reasonsByType[targetType]

  function handleSubmit() {
    if (!reason || alreadyReported) return
    mutation.mutate(
      {
        target_type: targetType,
        target_id: targetId,
        conversation_id: conversationId ?? "",
        reason,
        description,
      },
      {
        onSuccess: () => {
          toast.success(t("reportSent"))
          setReason(null)
          setDescription("")
          setAlreadyReported(false)
          onClose()
        },
        onError: (error) => {
          if (error instanceof ApiError && error.status === 409) {
            setAlreadyReported(true)
          } else {
            toast.error(t("reportError"))
          }
        },
      },
    )
  }

  const content = (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose()
        }}
        role="dialog"
        aria-modal="true"
      >
        {/* Header */}
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Flag className="h-5 w-5 text-red-500" />
            <h2 className="text-lg font-semibold text-foreground">
              {targetType === "message" && t("reportMessage")}
              {targetType === "user" && t("reportUser")}
              {targetType === "job" && t("reportJob")}
              {targetType === "application" && t("reportApplication")}
            </h2>
          </div>
          <Button variant="ghost" size="auto"
            onClick={onClose}
            className="rounded-lg p-1 text-muted-foreground hover:bg-muted"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Reasons */}
        <p className="mb-3 text-sm font-medium text-foreground">
          {t("selectReason")}
        </p>
        <div className="mb-4 space-y-2">
          {reasons.map((r) => (
            <label
              key={r}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 text-sm transition-colors",
                reason === r
                  ? "border-primary bg-primary-soft text-primary-deep"
                  : "border-border text-foreground hover:bg-muted",
              )}
            >
              <Input
                type="radio"
                name="reason"
                value={r}
                checked={reason === r}
                onChange={() => setReason(r)}
                className="sr-only"
              />
              <div
                className={cn(
                  "flex h-4 w-4 shrink-0 items-center justify-center rounded-full border-2",
                  reason === r
                    ? "border-primary"
                    : "border-border-strong",
                )}
              >
                {reason === r && (
                  <div className="h-2 w-2 rounded-full bg-primary" />
                )}
              </div>
              {t(`reason_${r}`)}
            </label>
          ))}
        </div>

        {/* Description */}
        <div className="mb-4">
          <label className="mb-1 block text-sm font-medium text-foreground">
            {t("description")}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t("descriptionPlaceholder")}
            maxLength={2000}
            rows={3}
            className="w-full resize-none rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
          />
          <p className="mt-1 text-right text-xs text-muted-foreground">
            {description.length} / 2000
          </p>
        </div>

        {/* Already reported warning */}
        {alreadyReported && (
          <div className="mb-4 flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400">
            <AlertCircle className="h-4 w-4 shrink-0" />
            {t("alreadyReported")}
          </div>
        )}

        {/* Submit */}
        <Button variant="ghost" size="auto"
          onClick={handleSubmit}
          disabled={!reason || mutation.isPending || alreadyReported}
          className="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-deep disabled:cursor-not-allowed disabled:opacity-50"
        >
          {mutation.isPending ? t("submitting") : t("submitReport")}
        </Button>
      </div>
    </div>
  )

  if (typeof window === "undefined") return null
  return createPortal(content, document.body)
}
