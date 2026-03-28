"use client"

import { useEffect, useRef } from "react"
import type { RemoteTrack, LocalTrack } from "livekit-client"
import { cn } from "@/shared/lib/utils"

interface VideoRendererProps {
  track: RemoteTrack | LocalTrack | null
  mirror?: boolean
  className?: string
  objectFit?: "cover" | "contain"
}

export function VideoRenderer({
  track,
  mirror = false,
  className,
  objectFit = "cover",
}: VideoRendererProps) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    const el = videoRef.current
    if (!el || !track) return

    track.attach(el)
    return () => {
      track.detach(el)
    }
  }, [track])

  return (
    <video
      ref={videoRef}
      autoPlay
      playsInline
      muted={mirror}
      className={cn(
        "h-full w-full",
        objectFit === "cover" ? "object-cover" : "object-contain",
        mirror && "scale-x-[-1]",
        className,
      )}
    />
  )
}
