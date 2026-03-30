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
    console.log("[Video] VideoRenderer effect - el:", !!el, "track:", !!track, "track.sid:", track?.sid, "mirror:", mirror)
    if (!el || !track) return

    track.attach(el)
    console.log("[Video] VideoRenderer attached track to <video> element, readyState:", el.readyState, "videoWidth:", el.videoWidth)

    // Explicitly call play() as a fallback — some browsers block autoPlay
    // on dynamically created video elements even when muted.
    el.play().catch((playErr) => {
      console.warn("[Video] VideoRenderer play() rejected:", playErr)
    })

    // Monitor the video element for actual playback
    const onPlaying = () => console.log("[Video] VideoRenderer <video> playing, videoWidth:", el.videoWidth, "videoHeight:", el.videoHeight)
    const onLoadedMetadata = () => console.log("[Video] VideoRenderer <video> loadedmetadata, videoWidth:", el.videoWidth, "videoHeight:", el.videoHeight)
    const onError = () => console.error("[Video] VideoRenderer <video> error:", el.error)
    el.addEventListener("playing", onPlaying)
    el.addEventListener("loadedmetadata", onLoadedMetadata)
    el.addEventListener("error", onError)

    return () => {
      el.removeEventListener("playing", onPlaying)
      el.removeEventListener("loadedmetadata", onLoadedMetadata)
      el.removeEventListener("error", onError)
      track.detach(el)
    }
  }, [track, mirror])

  return (
    <video
      ref={videoRef}
      autoPlay
      playsInline
      muted
      className={cn(
        "h-full w-full",
        objectFit === "cover" ? "object-cover" : "object-contain",
        mirror && "scale-x-[-1]",
        className,
      )}
    />
  )
}
