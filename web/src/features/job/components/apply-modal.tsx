"use client"

import { useState, useRef } from "react"
import { Send, Loader2, X, Upload, Video, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useApplyToJob } from "../hooks/use-job-applications"
import { uploadVideo } from "@/features/provider/api/upload-api"

interface ApplyModalProps {
  open: boolean
  onClose: () => void
  jobId: string
}

export function ApplyModal({ open, onClose, jobId }: ApplyModalProps) {
  const t = useTranslations("opportunity")
  const [message, setMessage] = useState("")
  const [videoUrl, setVideoUrl] = useState<string | null>(null)
  const [videoName, setVideoName] = useState<string | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const applyMutation = useApplyToJob()

  if (!open) return null

  async function handleVideoSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setIsUploading(true)
    try {
      const { url } = await uploadVideo(file)
      setVideoUrl(url)
      setVideoName(file.name)
    } catch {
      // upload failed silently — user can retry
    } finally {
      setIsUploading(false)
    }
  }

  function removeVideo() {
    setVideoUrl(null)
    setVideoName(null)
    if (fileInputRef.current) fileInputRef.current.value = ""
  }

  async function handleSubmit() {
    applyMutation.mutate(
      { jobId, message: message.trim(), videoUrl: videoUrl || undefined },
      {
        onSuccess: () => {
          setMessage("")
          setVideoUrl(null)
          setVideoName(null)
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
          {/* Message (optional) */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("yourMessage")}
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

          {/* Video upload (optional) */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("optionalVideo")}
            </label>
            <input
              ref={fileInputRef}
              type="file"
              accept="video/*"
              onChange={handleVideoSelect}
              className="hidden"
            />

            {!videoUrl && !isUploading && (
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className={cn(
                  "w-full rounded-lg border-2 border-dashed border-slate-200 dark:border-slate-600 p-4",
                  "flex items-center justify-center gap-2 text-sm text-slate-500 transition-colors",
                  "hover:border-rose-300 hover:text-rose-600 dark:hover:border-rose-500/30",
                )}
              >
                <Upload className="h-4 w-4" />
                {t("uploadVideo")}
              </button>
            )}

            {isUploading && (
              <div className="flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-slate-600 p-4 text-sm text-slate-500">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("uploading")}
              </div>
            )}

            {videoUrl && (
              <div className="space-y-2">
                <div className="aspect-video max-h-[200px] overflow-hidden rounded-lg bg-black">
                  <video src={videoUrl} controls className="h-full w-full object-contain">
                    <track kind="captions" />
                  </video>
                </div>
                <button type="button" onClick={removeVideo} className="flex items-center gap-1.5 text-sm text-red-500 hover:text-red-600">
                  <Trash2 className="h-3.5 w-3.5" />
                  {videoName}
                </button>
              </div>
            )}
          </div>

          {/* Error */}
          {applyMutation.isError && (
            <p className="text-sm text-red-500">{applyMutation.error?.message || t("error")}</p>
          )}

          {/* Submit */}
          <button
            type="button"
            onClick={handleSubmit}
            disabled={applyMutation.isPending || isUploading}
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
