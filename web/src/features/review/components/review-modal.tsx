"use client"

import { useState, useCallback, useRef } from "react"
import { X, Loader2, Video, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useCreateReview, useUploadReviewVideo } from "../hooks/use-reviews"
import { StarRating } from "./star-rating"

const MAX_VIDEO_SIZE = 100 * 1024 * 1024 // 100 MB
const ACCEPTED_VIDEO_TYPES = ["video/mp4", "video/webm", "video/quicktime"]

interface ReviewModalProps {
  proposalId: string
  proposalTitle: string
  isOpen: boolean
  onClose: () => void
}

export function ReviewModal({
  proposalId,
  proposalTitle,
  isOpen,
  onClose,
}: ReviewModalProps) {
  const t = useTranslations("review")
  const [globalRating, setGlobalRating] = useState(0)
  const [timeliness, setTimeliness] = useState(0)
  const [communication, setCommunication] = useState(0)
  const [quality, setQuality] = useState(0)
  const [comment, setComment] = useState("")
  const [videoUrl, setVideoUrl] = useState("")
  const [videoFile, setVideoFile] = useState<File | null>(null)
  const [titleVisible, setTitleVisible] = useState(true)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const { mutate: submitReview, isPending } = useCreateReview()
  const { mutate: uploadVideo, isPending: isUploading } = useUploadReviewVideo()

  const resetForm = useCallback(() => {
    setGlobalRating(0)
    setTimeliness(0)
    setCommunication(0)
    setQuality(0)
    setComment("")
    setVideoUrl("")
    setVideoFile(null)
    setTitleVisible(true)
  }, [])

  const handleVideoSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (!ACCEPTED_VIDEO_TYPES.includes(file.type)) return
    if (file.size > MAX_VIDEO_SIZE) return

    setVideoFile(file)
    uploadVideo(file, {
      onSuccess: (url) => setVideoUrl(url),
      onError: () => setVideoFile(null),
    })
  }, [uploadVideo])

  const handleRemoveVideo = useCallback(() => {
    setVideoUrl("")
    setVideoFile(null)
    if (fileInputRef.current) fileInputRef.current.value = ""
  }, [])

  const handleSubmit = useCallback(() => {
    if (globalRating === 0) return

    submitReview(
      {
        proposal_id: proposalId,
        global_rating: globalRating,
        timeliness: timeliness > 0 ? timeliness : undefined,
        communication: communication > 0 ? communication : undefined,
        quality: quality > 0 ? quality : undefined,
        comment: comment.trim() || undefined,
        video_url: videoUrl || undefined,
        title_visible: titleVisible,
      },
      {
        onSuccess: () => {
          resetForm()
          onClose()
        },
      },
    )
  }, [
    proposalId, globalRating, timeliness, communication,
    quality, comment, videoUrl, titleVisible, submitReview, resetForm, onClose,
  ])

  if (!isOpen) return null

  const isBusy = isPending || isUploading

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      onClick={(e) => e.target === e.currentTarget && onClose()}
      role="dialog"
      aria-modal="true"
      aria-label={t("title")}
    >
      <div className="mx-4 w-full max-w-lg animate-scale-in rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-900">
        <ReviewModalHeader
          title={t("title")}
          subtitle={proposalTitle}
          closeLabel={t("close")}
          onClose={onClose}
        />

        <div className="space-y-5">
          <StarRating
            rating={globalRating}
            onRatingChange={setGlobalRating}
            size="lg"
            label={`${t("globalRating")} *`}
          />

          <DetailedCriteria
            timeliness={timeliness}
            communication={communication}
            quality={quality}
            onTimelinessChange={setTimeliness}
            onCommunicationChange={setCommunication}
            onQualityChange={setQuality}
            labels={{
              detailed: t("detailedCriteria"),
              timeliness: t("timeliness"),
              communication: t("communication"),
              quality: t("quality"),
            }}
          />

          <CommentField
            value={comment}
            onChange={setComment}
            label={t("comment")}
            placeholder={t("commentPlaceholder")}
            disabled={isBusy}
          />

          <VideoUploadField
            videoUrl={videoUrl}
            videoFile={videoFile}
            isUploading={isUploading}
            fileInputRef={fileInputRef}
            onSelect={handleVideoSelect}
            onRemove={handleRemoveVideo}
            labels={{
              add: t("addVideo"),
              remove: t("removeVideo"),
              uploading: t("uploadingVideo"),
            }}
          />

          <TitleVisibilityField
            checked={titleVisible}
            onChange={setTitleVisible}
            label={t("titleVisibleLabel")}
            hint={t("titleVisibleHint")}
          />

          <ReviewModalActions
            onSubmit={handleSubmit}
            onCancel={onClose}
            isPending={isPending}
            isBusy={isBusy}
            canSubmit={globalRating > 0}
            labels={{ submit: t("submit"), cancel: t("cancel") }}
          />
        </div>
      </div>
    </div>
  )
}

function ReviewModalHeader({
  title,
  subtitle,
  closeLabel,
  onClose,
}: {
  title: string
  subtitle: string
  closeLabel: string
  onClose: () => void
}) {
  return (
    <div className="mb-6 flex items-center justify-between">
      <div>
        <h2 className="text-lg font-semibold text-foreground">{title}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">{subtitle}</p>
      </div>
      <button
        type="button"
        onClick={onClose}
        className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted transition-colors"
        aria-label={closeLabel}
      >
        <X className="h-5 w-5" />
      </button>
    </div>
  )
}

function DetailedCriteria({
  timeliness,
  communication,
  quality,
  onTimelinessChange,
  onCommunicationChange,
  onQualityChange,
  labels,
}: {
  timeliness: number
  communication: number
  quality: number
  onTimelinessChange: (v: number) => void
  onCommunicationChange: (v: number) => void
  onQualityChange: (v: number) => void
  labels: { detailed: string; timeliness: string; communication: string; quality: string }
}) {
  return (
    <div className="space-y-3 border-t border-border pt-4">
      <p className="text-sm font-medium text-muted-foreground">
        {labels.detailed}
      </p>
      <StarRating rating={timeliness} onRatingChange={onTimelinessChange} size="md" label={labels.timeliness} />
      <StarRating rating={communication} onRatingChange={onCommunicationChange} size="md" label={labels.communication} />
      <StarRating rating={quality} onRatingChange={onQualityChange} size="md" label={labels.quality} />
    </div>
  )
}

function CommentField({
  value,
  onChange,
  label,
  placeholder,
  disabled,
}: {
  value: string
  onChange: (v: string) => void
  label: string
  placeholder: string
  disabled: boolean
}) {
  return (
    <div className="space-y-1.5">
      <label htmlFor="review-comment" className="text-sm font-medium text-foreground">
        {label}
      </label>
      <textarea
        id="review-comment"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={4}
        maxLength={2000}
        className={cn(
          "w-full resize-none rounded-lg border border-border bg-transparent",
          "px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground",
          "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          "transition-all duration-200",
        )}
        disabled={disabled}
      />
      <p className="text-right text-xs text-muted-foreground">{value.length}/2000</p>
    </div>
  )
}

function VideoUploadField({
  videoUrl,
  videoFile,
  isUploading,
  fileInputRef,
  onSelect,
  onRemove,
  labels,
}: {
  videoUrl: string
  videoFile: File | null
  isUploading: boolean
  fileInputRef: React.RefObject<HTMLInputElement | null>
  onSelect: (e: React.ChangeEvent<HTMLInputElement>) => void
  onRemove: () => void
  labels: { add: string; remove: string; uploading: string }
}) {
  if (videoUrl) {
    return (
      <div className="space-y-2">
        <video
          src={videoUrl}
          controls
          className="w-full rounded-lg border border-border"
          style={{ maxHeight: 200 }}
        />
        <button
          type="button"
          onClick={onRemove}
          className="flex items-center gap-1.5 text-sm text-destructive hover:underline"
        >
          <Trash2 className="h-3.5 w-3.5" />
          {labels.remove}
        </button>
      </div>
    )
  }

  return (
    <div>
      <input
        ref={fileInputRef}
        type="file"
        accept="video/mp4,video/webm,video/quicktime"
        onChange={onSelect}
        className="hidden"
        id="review-video-input"
      />
      <button
        type="button"
        onClick={() => fileInputRef.current?.click()}
        disabled={isUploading}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed",
          "border-border px-4 py-3 text-sm text-muted-foreground",
          "hover:border-rose-300 hover:text-foreground transition-all duration-200",
          "disabled:opacity-50 disabled:cursor-not-allowed",
        )}
      >
        {isUploading ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            {labels.uploading}
          </>
        ) : (
          <>
            <Video className="h-4 w-4" />
            {labels.add}
          </>
        )}
      </button>
      {videoFile && isUploading && (
        <p className="mt-1 text-xs text-muted-foreground">{videoFile.name}</p>
      )}
    </div>
  )
}

function TitleVisibilityField({
  checked,
  onChange,
  label,
  hint,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  label: string
  hint: string
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border bg-muted/30 px-3 py-3 transition-colors hover:bg-muted/50">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 flex-shrink-0 cursor-pointer rounded border-border text-rose-600 focus:ring-2 focus:ring-rose-500/20"
      />
      <span className="space-y-1 text-sm">
        <span className="block font-medium text-foreground">{label}</span>
        <span className="block text-xs text-muted-foreground">{hint}</span>
      </span>
    </label>
  )
}

function ReviewModalActions({
  onSubmit,
  onCancel,
  isPending,
  isBusy,
  canSubmit,
  labels,
}: {
  onSubmit: () => void
  onCancel: () => void
  isPending: boolean
  isBusy: boolean
  canSubmit: boolean
  labels: { submit: string; cancel: string }
}) {
  return (
    <div className="flex gap-3 pt-2">
      <button
        type="button"
        onClick={onSubmit}
        disabled={isBusy || !canSubmit}
        className={cn(
          "flex-1 rounded-lg px-4 py-2.5 text-sm font-semibold text-white",
          "gradient-primary hover:shadow-glow active:scale-[0.98]",
          "transition-all duration-200",
          "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none",
        )}
      >
        {isPending && <Loader2 className="mr-2 inline h-4 w-4 animate-spin" />}
        {labels.submit}
      </button>
      <button
        type="button"
        onClick={onCancel}
        disabled={isBusy}
        className={cn(
          "rounded-lg border border-border px-4 py-2.5 text-sm font-medium",
          "text-foreground hover:bg-muted transition-all duration-200",
        )}
      >
        {labels.cancel}
      </button>
    </div>
  )
}
