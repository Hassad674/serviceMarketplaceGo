"use client"

import { useState, useRef } from "react"
import { Send, Loader2, X, Upload, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useApplyToJob } from "../hooks/use-job-applications"
import { uploadVideo } from "@/shared/lib/upload-api"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"

interface ApplyModalProps {
  open: boolean
  onClose: () => void
  jobId: string
}

// W-13 · Apply modal — Soleil v2 dressing.
// Form behaviour, mutation, video-upload flow are preserved exactly
// (zod / react-hook-form bindings already at the field level via
// useApplyToJob). This file only re-skins the chrome: ivoire panel,
// Fraunces title, corail submit button, soft borders.
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
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-scale-in w-full max-w-lg overflow-hidden rounded-2xl border border-border bg-card p-6 sm:p-7"
        style={{ boxShadow: "var(--shadow-card-strong)" }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-5 flex items-center justify-between">
          <h3 className="font-serif text-[22px] font-medium tracking-[-0.015em] text-foreground">
            {t("apply")}
          </h3>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-full text-muted-foreground hover:bg-background hover:text-foreground"
            aria-label={t("close")}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        <div className="space-y-5">
          {/* Message (optional) */}
          <div>
            <label
              htmlFor="apply-message"
              className="mb-2 block text-[13px] font-semibold text-foreground"
            >
              {t("yourMessage")}
            </label>
            <textarea
              id="apply-message"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={t("messagePlaceholder")}
              rows={5}
              maxLength={5000}
              className={cn(
                "w-full resize-none rounded-xl border border-border bg-card px-4 py-3 text-sm text-foreground",
                "placeholder:text-muted-foreground",
                "focus:border-border-strong focus:ring-2 focus:ring-primary/20 outline-none transition-all",
              )}
            />
            <p className="mt-1.5 text-right font-mono text-[11px] text-subtle-foreground">
              {message.length}/5000
            </p>
          </div>

          {/* Video upload (optional) */}
          <div>
            <label className="mb-2 block text-[13px] font-semibold text-foreground">
              {t("optionalVideo")}
            </label>
            <Input
              ref={fileInputRef}
              type="file"
              accept="video/*"
              onChange={handleVideoSelect}
              className="hidden"
            />

            {!videoUrl && !isUploading && (
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className={cn(
                  "flex w-full items-center justify-center gap-2 rounded-xl border-2 border-dashed border-border-strong bg-background p-5 text-sm font-medium text-muted-foreground transition-colors",
                  "hover:border-primary hover:bg-primary-soft hover:text-primary-deep",
                )}
              >
                <Upload className="h-4 w-4" strokeWidth={1.6} />
                {t("uploadVideo")}
              </Button>
            )}

            {isUploading && (
              <div className="flex items-center justify-center gap-2 rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("uploading")}
              </div>
            )}

            {videoUrl && (
              <div className="space-y-2">
                <div className="aspect-video max-h-[200px] overflow-hidden rounded-xl bg-foreground">
                  <video src={videoUrl} controls className="h-full w-full object-contain">
                    <track kind="captions" />
                  </video>
                </div>
                <Button
                  variant="ghost"
                  size="auto"
                  type="button"
                  onClick={removeVideo}
                  className="inline-flex items-center gap-1.5 text-[13px] font-medium text-primary-deep hover:text-primary"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {videoName}
                </Button>
              </div>
            )}
          </div>

          {/* Error */}
          {applyMutation.isError && (
            <p className="text-sm text-primary-deep">
              {applyMutation.error?.message || t("error")}
            </p>
          )}

          {/* Submit — corail pill */}
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleSubmit}
            disabled={applyMutation.isPending || isUploading}
            className={cn(
              "flex w-full items-center justify-center gap-2 rounded-full px-4 py-3 text-[14px] font-semibold transition-all",
              "bg-primary text-white hover:bg-primary-deep active:scale-[0.98]",
              "disabled:cursor-not-allowed disabled:opacity-50",
            )}
            style={{ boxShadow: "var(--shadow-message)" }}
          >
            {applyMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" strokeWidth={1.8} />
            )}
            {t("apply")}
          </Button>
        </div>
      </div>
    </div>
  )
}
