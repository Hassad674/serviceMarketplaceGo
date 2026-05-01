"use client"

import { useState } from "react"
import { Video, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"

import { Button } from "@/shared/components/ui/button"
const VIDEO_MAX_SIZE = 50 * 1024 * 1024 // 50 MB

interface ProfileVideoCardProps {
  videoUrl: string
  labels: {
    title: string
    emptyLabel: string
    emptyDescription: string
  }
  actions?: {
    onUpload?: (file: File) => Promise<void>
    uploading?: boolean
    onDelete?: () => void
    deleting?: boolean
  }
  readOnly?: boolean
}

// ProfileVideoCard renders the presentation video block: embedded
// player + upload/change/delete actions when editable. Collapses to
// nothing in read-only mode with an empty video URL so listing-card
// contexts can safely include it without extra guards.
export function ProfileVideoCard(props: ProfileVideoCardProps) {
  const { videoUrl, labels, actions, readOnly = false } = props
  const [videoModalOpen, setVideoModalOpen] = useState(false)
  const tUpload = useTranslations("upload")

  if (readOnly && !videoUrl) return null

  async function handleVideoUpload(file: File) {
    if (!actions?.onUpload) return
    await actions.onUpload(file)
    setVideoModalOpen(false)
  }

  return (
    <>
      <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <VideoHeader
          title={labels.title}
          hasVideo={Boolean(videoUrl)}
          readOnly={readOnly}
          actions={actions}
          onOpenUpload={() => setVideoModalOpen(true)}
        />

        {videoUrl ? (
          <VideoPlayer videoUrl={videoUrl} title={labels.title} />
        ) : (
          <VideoEmptyState
            labels={labels}
            canUpload={!readOnly && Boolean(actions?.onUpload)}
            onOpenUpload={() => setVideoModalOpen(true)}
          />
        )}
      </section>

      {!readOnly && actions?.onUpload ? (
        <UploadModal
          open={videoModalOpen}
          onClose={() => setVideoModalOpen(false)}
          onUpload={handleVideoUpload}
          accept="video/*"
          maxSize={VIDEO_MAX_SIZE}
          title={tUpload("addVideo")}
          description={tUpload("videoFormats")}
          uploading={actions.uploading ?? false}
        />
      ) : null}
    </>
  )
}

interface VideoHeaderProps {
  title: string
  hasVideo: boolean
  readOnly: boolean
  actions: ProfileVideoCardProps["actions"]
  onOpenUpload: () => void
}

function VideoHeader({
  title,
  hasVideo,
  readOnly,
  actions,
  onOpenUpload,
}: VideoHeaderProps) {
  const t = useTranslations("profile")
  return (
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-lg font-semibold text-foreground">{title}</h2>
      {hasVideo && !readOnly ? (
        <div className="flex items-center gap-3">
          {actions?.onDelete ? (
            <Button variant="ghost" size="auto"
              type="button"
              onClick={actions.onDelete}
              disabled={actions.deleting}
              className="flex items-center gap-1 text-sm font-medium text-destructive hover:opacity-80 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
            >
              <Trash2 className="w-4 h-4" aria-hidden="true" />
              {t("removeVideo")}
            </Button>
          ) : null}
          {actions?.onUpload ? (
            <Button variant="ghost" size="auto"
              type="button"
              onClick={onOpenUpload}
              className="text-sm font-medium text-primary hover:opacity-80 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              {t("changeVideo")}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

interface VideoPlayerProps {
  videoUrl: string
  title: string
}

function VideoPlayer({ videoUrl, title }: VideoPlayerProps) {
  return (
    <div className="aspect-video max-h-[300px] overflow-hidden rounded-lg bg-black">
      <video
        src={videoUrl}
        controls
        className="h-full w-full object-contain"
        aria-label={title}
      >
        <track kind="captions" />
        Your browser does not support video playback.
      </video>
    </div>
  )
}

interface VideoEmptyStateProps {
  labels: ProfileVideoCardProps["labels"]
  canUpload: boolean
  onOpenUpload: () => void
}

function VideoEmptyState({
  labels,
  canUpload,
  onOpenUpload,
}: VideoEmptyStateProps) {
  const t = useTranslations("profile")
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center max-h-[200px]">
      <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3">
        <Video className="w-6 h-6 text-muted-foreground" aria-hidden="true" />
      </div>
      <p className="text-base font-medium text-foreground mb-1">
        {labels.emptyLabel}
      </p>
      <p className="text-sm text-muted-foreground italic mb-3">
        {labels.emptyDescription}
      </p>
      {canUpload ? (
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onOpenUpload}
          className="bg-primary text-primary-foreground rounded-md h-10 px-4 text-sm font-medium hover:opacity-90 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          {t("addVideo")}
        </Button>
      ) : null}
    </div>
  )
}
