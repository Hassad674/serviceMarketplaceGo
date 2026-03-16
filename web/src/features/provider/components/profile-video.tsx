"use client"

import { Video } from "lucide-react"

interface ProfileVideoProps {
  videoUrl: string | undefined
  title?: string
  emptyLabel?: string
  emptyDescription?: string
}

export function ProfileVideo({
  videoUrl,
  title = "Video de presentation",
  emptyLabel = "Aucune video de presentation",
  emptyDescription = "Ajoutez une video pour presenter votre activite",
}: ProfileVideoProps) {
  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-foreground mb-4">{title}</h2>

      {videoUrl ? (
        <div className="aspect-video rounded-lg overflow-hidden bg-muted">
          <video
            src={videoUrl}
            controls
            className="w-full h-full object-cover"
            aria-label={title}
          >
            <track kind="captions" />
            Votre navigateur ne supporte pas la lecture video.
          </video>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
            <Video className="w-7 h-7 text-muted-foreground" aria-hidden="true" />
          </div>
          <p className="text-sm font-medium text-foreground mb-1">
            {emptyLabel}
          </p>
          <p className="text-sm text-muted-foreground italic mb-4">
            {emptyDescription}
          </p>
          <button
            type="button"
            className="bg-primary text-primary-foreground rounded-md h-10 px-4 text-sm font-medium hover:opacity-90 transition-opacity focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
          >
            Ajouter une video
          </button>
        </div>
      )}
    </section>
  )
}
