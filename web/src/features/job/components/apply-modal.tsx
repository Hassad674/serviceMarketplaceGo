"use client"

import { useState } from "react"
import { Send, Loader2, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useApplyToJob } from "../hooks/use-job-applications"

interface ApplyModalProps {
  open: boolean
  onClose: () => void
  jobId: string
}

export function ApplyModal({ open, onClose, jobId }: ApplyModalProps) {
  const t = useTranslations("opportunity")
  const [message, setMessage] = useState("")
  const [videoUrl, setVideoUrl] = useState("")
  const applyMutation = useApplyToJob()

  if (!open) return null

  async function handleSubmit() {
    if (!message.trim()) return
    applyMutation.mutate(
      { jobId, message: message.trim(), videoUrl: videoUrl.trim() || undefined },
      {
        onSuccess: () => {
          setMessage("")
          setVideoUrl("")
          onClose()
        },
      },
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-6 w-full max-w-lg mx-4 animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">{t("apply")}</h3>
          <button type="button" onClick={onClose} className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700">
            <X className="h-5 w-5 text-slate-400" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("yourMessage")} <span className="text-rose-500">*</span>
            </label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={t("messagePlaceholder")}
              rows={5}
              maxLength={5000}
              className={cn(
                "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm shadow-xs resize-none",
                "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 outline-none transition-all",
                "dark:border-slate-600 dark:bg-slate-700 dark:text-white",
              )}
            />
            <p className="text-xs text-slate-400 mt-1 text-right">{message.length}/5000</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("optionalVideo")}
            </label>
            <input
              type="url"
              value={videoUrl}
              onChange={(e) => setVideoUrl(e.target.value)}
              placeholder="https://..."
              className={cn(
                "w-full h-10 rounded-lg border border-slate-200 bg-white px-3 text-sm shadow-xs",
                "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 outline-none transition-all",
                "dark:border-slate-600 dark:bg-slate-700 dark:text-white",
              )}
            />
          </div>

          {applyMutation.isError && (
            <p className="text-sm text-red-500">{applyMutation.error?.message || t("error")}</p>
          )}

          <button
            type="button"
            onClick={handleSubmit}
            disabled={!message.trim() || applyMutation.isPending}
            className={cn(
              "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition-all",
              "gradient-primary text-white hover:shadow-glow active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {applyMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            {t("apply")}
          </button>
        </div>
      </div>
    </div>
  )
}
