"use client"

import { useState } from "react"
import { Video } from "lucide-react"
import { UploadModal } from "@/shared/components/upload-modal"

interface ProfileVideoProps {
  videoUrl: string | undefined
  title?: string
  emptyLabel?: string
  emptyDescription?: string
  onUploadVideo: (file: File) => Promise<void>
  uploadingVideo?: boolean
}

const VIDEO_MAX_SIZE = 50 * 1024 * 1024 // 50 MB

export function ProfileVideo({
  videoUrl,
  title = "Presentation Video",
  emptyLabel = "No presentation video",
  emptyDescription = "Add a video to present your activity",
  onUploadVideo,
  uploadingVideo = false,
}: ProfileVideoProps) {
  const [videoModalOpen, setVideoModalOpen] = useState(false)

  async function handleVideoUpload(file: File) {
    await onUploadVideo(file)
    setVideoModalOpen(false)
  }

  return (
    <>
      <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">{title}</h2>
          {videoUrl && (
            <button
              type="button"
              onClick={() => setVideoModalOpen(true)}
              className="text-sm font-medium text-primary hover:opacity-80 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              Change video
            </button>
          )}
        </div>

        {videoUrl ? (
          <div className="aspect-video rounded-lg overflow-hidden bg-muted">
            <video
              src={videoUrl}
              controls
              className="w-full h-full object-cover"
              aria-label={title}
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
              {emptyLabel}
            </p>
            <p className="text-sm text-muted-foreground italic mb-3">
              {emptyDescription}
            </p>
            <button
              type="button"
              onClick={() => setVideoModalOpen(true)}
              className="bg-primary text-primary-foreground rounded-md h-10 px-4 text-sm font-medium hover:opacity-90 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              Add a video
            </button>
          </div>
        )}
      </section>

      <UploadModal
        open={videoModalOpen}
        onClose={() => setVideoModalOpen(false)}
        onUpload={handleVideoUpload}
        accept="video/*"
        maxSize={VIDEO_MAX_SIZE}
        title="Add a video"
        description="Accepted formats: MP4, WebM. 50 MB maximum."
        uploading={uploadingVideo}
      />
    </>
  )
}
