"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Play, Pause, Mic } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import type { VoiceMetadata } from "../types"
import { Button } from "@/shared/components/ui/button"

interface VoiceMessageProps {
  metadata: VoiceMetadata
  isOwn: boolean
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, "0")}`
}

export function VoiceMessage({ metadata, isOwn }: VoiceMessageProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [currentTime, setCurrentTime] = useState(0)
  const animRef = useRef<number | null>(null)

  const updateProgress = useCallback(() => {
    const audio = audioRef.current
    if (!audio || audio.paused) return
    const pct = audio.duration ? (audio.currentTime / audio.duration) * 100 : 0
    setProgress(pct)
    setCurrentTime(audio.currentTime)
    animRef.current = requestAnimationFrame(updateProgress)
  }, [])

  useEffect(() => {
    const audio = audioRef.current
    const anim = animRef
    return () => {
      if (anim.current) cancelAnimationFrame(anim.current)
      audio?.pause()
    }
  }, [])

  const togglePlay = useCallback(() => {
    const audio = audioRef.current
    if (!audio) return

    if (audio.paused) {
      audio.play()
      setIsPlaying(true)
      animRef.current = requestAnimationFrame(updateProgress)
    } else {
      audio.pause()
      setIsPlaying(false)
      if (animRef.current) cancelAnimationFrame(animRef.current)
    }
  }, [updateProgress])

  const handleEnded = useCallback(() => {
    setIsPlaying(false)
    setProgress(0)
    setCurrentTime(0)
    if (animRef.current) cancelAnimationFrame(animRef.current)
  }, [])

  const handleProgressClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const audio = audioRef.current
    if (!audio || !audio.duration) return
    const rect = e.currentTarget.getBoundingClientRect()
    const pct = (e.clientX - rect.left) / rect.width
    audio.currentTime = pct * audio.duration
    setProgress(pct * 100)
    setCurrentTime(audio.currentTime)
  }, [])

  const displayDuration = isPlaying || currentTime > 0
    ? formatDuration(currentTime)
    : formatDuration(metadata.duration)

  return (
    <div className="flex items-center gap-3 py-1">
      {/* Hidden audio element */}
      <audio ref={audioRef} src={metadata.url} preload="metadata" onEnded={handleEnded} />

      {/* Play/Pause button */}
      <Button variant="ghost" size="auto"
        type="button"
        onClick={togglePlay}
        className={cn(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-full transition-all duration-200",
          isOwn
            ? "bg-white/20 hover:bg-white/30 active:scale-[0.95]"
            : "bg-rose-100 hover:bg-rose-200 dark:bg-rose-500/20 dark:hover:bg-rose-500/30 active:scale-[0.95]",
        )}
        aria-label={isPlaying ? "Pause" : "Play"}
      >
        {isPlaying ? (
          <Pause
            className={cn("h-4 w-4", isOwn ? "text-white" : "text-rose-600 dark:text-rose-400")}
            strokeWidth={2}
          />
        ) : (
          <Play
            className={cn("h-4 w-4", isOwn ? "text-white" : "text-rose-600 dark:text-rose-400")}
            strokeWidth={2}
          />
        )}
      </Button>

      {/* Progress bar + duration */}
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div
          className={cn(
            "h-1.5 w-full cursor-pointer rounded-full",
            isOwn ? "bg-white/20" : "bg-gray-200 dark:bg-gray-700",
          )}
          onClick={handleProgressClick}
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className={cn(
              "h-full rounded-full transition-[width] duration-100",
              isOwn ? "bg-white" : "bg-rose-500 dark:bg-rose-400",
            )}
            style={{ width: `${progress}%` }}
          />
        </div>
        <div className="flex items-center gap-1">
          <Mic
            className={cn(
              "h-3 w-3",
              isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
            )}
            strokeWidth={1.5}
          />
          <span
            className={cn(
              "font-mono text-[10px]",
              isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
            )}
          >
            {displayDuration}
          </span>
        </div>
      </div>
    </div>
  )
}
