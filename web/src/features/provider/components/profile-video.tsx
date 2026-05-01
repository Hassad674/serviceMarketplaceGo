"use client"

import { useState } from "react"
import { Video, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"
import { Button } from "@/shared/components/ui/button"

interface ProfileVideoProps {
  videoUrl: string | undefined
  title?: string
  emptyLabel?: string
  emptyDescription?: string
  onUploadVideo?: (file: File) => Promise<void>
  uploadingVideo?: boolean
  onDeleteVideo?: () => void
  deletingVideo?: boolean
  readOnly?: boolean
}

const VIDEO_MAX_SIZE = 50 * 1024 * 1024 // 50 MB

export function ProfileVideo({
  videoUrl,
  title,
  emptyLabel,
  emptyDescription,
  onUploadVideo,
  uploadingVideo = false,
  onDeleteVideo,
  deletingVideo = false,
  readOnly = false,
}: ProfileVideoProps) {
  const [videoModalOpen, setVideoModalOpen] = useState(false)
  const t = useTranslations("profile")
  const tUpload = useTranslations("upload")

  const displayTitle = title ?? t("videoTitle")
  const displayEmptyLabel = emptyLabel ?? t("noVideo")
  const displayEmptyDescription = emptyDescription ?? t("addVideoDesc")

  // Hide entire section when readOnly and no video
  if (readOnly && !videoUrl) return null

  async function handleVideoUpload(file: File) {
    if (!onUploadVideo) return
    await onUploadVideo(file)
    setVideoModalOpen(false)
  }

  return (
    <>
      <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">{displayTitle}</h2>
          {videoUrl && !readOnly && (
            <div className="flex items-center gap-3">
              {onDeleteVideo && (
                <Button variant="ghost" size="auto"
                  type="button"
                  onClick={onDeleteVideo}
                  disabled={deletingVideo}
                  className="flex items-center gap-1 text-sm font-medium text-destructive hover:opacity-80 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
                >
                  <Trash2 className="w-4 h-4" aria-hidden="true" />
                  {t("removeVideo")}
                </Button>
              )}
              <Button variant="ghost" size="auto"
                type="button"
                onClick={() => setVideoModalOpen(true)}
                className="text-sm font-medium text-primary hover:opacity-80 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
              >
                {t("changeVideo")}
              </Button>
            </div>
          )}
        </div>

        {videoUrl ? (
          <div className="aspect-video max-h-[300px] overflow-hidden rounded-lg bg-black">
            <video
              src={videoUrl}
              controls
              className="h-full w-full object-contain"
              aria-label={displayTitle}
            >
              <track kind="captions" />
              Your browser does not support video playback.
            </video>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-8 text-center max-h-[200px]">
            <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3">
              <Video className="w-6 h-6 text-muted-foreground" aria-hidden="true" />
            </div>
            <p className="text-base font-medium text-foreground mb-1">
              {displayEmptyLabel}
            </p>
            <p className="text-sm text-muted-foreground italic mb-3">
              {displayEmptyDescription}
            </p>
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => setVideoModalOpen(true)}
              className="bg-primary text-primary-foreground rounded-md h-10 px-4 text-sm font-medium hover:opacity-90 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              {t("addVideo")}
            </Button>
          </div>
        )}
      </section>

      {!readOnly && (
        <UploadModal
          open={videoModalOpen}
          onClose={() => setVideoModalOpen(false)}
          onUpload={handleVideoUpload}
          accept="video/*"
          maxSize={VIDEO_MAX_SIZE}
          title={tUpload("addVideo")}
          description={tUpload("videoFormats")}
          uploading={uploadingVideo}
        />
      )}
    </>
  )
}
