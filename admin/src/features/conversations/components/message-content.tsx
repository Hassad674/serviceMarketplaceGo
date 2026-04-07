import { useState, useRef, useCallback, useEffect } from "react"
import { FileText, Download, Mic, Play, Pause } from "lucide-react"

type FileMetadata = {
  url: string
  filename: string
  size: number
  mime_type: string
}

type VoiceMetadata = {
  url: string
  duration: number
  size: number
  mime_type: string
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, "0")}`
}

function parseFileMetadata(metadata: Record<string, unknown>): FileMetadata | null {
  const url = metadata.url as string | undefined
  const filename = metadata.filename as string | undefined
  const size = metadata.size as number | undefined
  const mimeType = metadata.mime_type as string | undefined
  if (!url || !filename) return null
  return { url, filename, size: size ?? 0, mime_type: mimeType ?? "" }
}

function parseVoiceMetadata(metadata: Record<string, unknown>): VoiceMetadata | null {
  const url = metadata.url as string | undefined
  const duration = metadata.duration as number | undefined
  const size = metadata.size as number | undefined
  const mimeType = metadata.mime_type as string | undefined
  if (!url) return null
  return { url, duration: duration ?? 0, size: size ?? 0, mime_type: mimeType ?? "" }
}

export function ImageContent({ metadata }: { metadata: Record<string, unknown> }) {
  const file = parseFileMetadata(metadata)
  if (!file) return null

  return (
    <div className="space-y-1.5">
      <a href={file.url} target="_blank" rel="noopener noreferrer" className="block">
        <img
          src={file.url}
          alt={file.filename}
          className="max-w-[400px] rounded-xl object-cover"
          loading="lazy"
        />
      </a>
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Download className="h-3 w-3 shrink-0" strokeWidth={1.5} />
        <span className="truncate">{file.filename}</span>
        <span>({formatFileSize(file.size)})</span>
      </div>
    </div>
  )
}

export function DocumentContent({ metadata }: { metadata: Record<string, unknown> }) {
  const file = parseFileMetadata(metadata)
  if (!file) return null

  return (
    <a
      href={file.url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center gap-3 rounded-xl border border-border bg-muted/50 p-3 transition-colors hover:bg-muted"
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-gray-200/60">
        <FileText className="h-5 w-5 text-muted-foreground" strokeWidth={1.5} />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-foreground">
          {file.filename}
        </p>
        <p className="text-[10px] text-muted-foreground">
          {formatFileSize(file.size)}
        </p>
      </div>
      <Download className="h-4 w-4 shrink-0 text-muted-foreground" strokeWidth={1.5} />
    </a>
  )
}

export function VoiceContent({ metadata }: { metadata: Record<string, unknown> }) {
  const voice = parseVoiceMetadata(metadata)
  if (!voice) return null

  return <VoicePlayer voice={voice} />
}

function VoicePlayer({ voice }: { voice: VoiceMetadata }) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const animRef = useRef<number | null>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [currentTime, setCurrentTime] = useState(0)

  const updateProgress = useCallback(() => {
    const audio = audioRef.current
    if (!audio || audio.paused) return
    const pct = audio.duration ? (audio.currentTime / audio.duration) * 100 : 0
    setProgress(pct)
    setCurrentTime(audio.currentTime)
    animRef.current = requestAnimationFrame(updateProgress)
  }, [])

  useEffect(() => {
    return () => {
      if (animRef.current) cancelAnimationFrame(animRef.current)
      audioRef.current?.pause()
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

  const handleSeek = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const audio = audioRef.current
    if (!audio || !audio.duration) return
    const rect = e.currentTarget.getBoundingClientRect()
    const pct = (e.clientX - rect.left) / rect.width
    audio.currentTime = pct * audio.duration
    setProgress(pct * 100)
    setCurrentTime(audio.currentTime)
  }, [])

  const displayTime = isPlaying || currentTime > 0
    ? formatDuration(currentTime)
    : formatDuration(voice.duration)

  return (
    <div className="flex items-center gap-3 rounded-xl border border-border bg-muted/50 p-3">
      <audio
        ref={audioRef}
        src={voice.url}
        preload="metadata"
        onEnded={handleEnded}
      />
      <button
        type="button"
        onClick={togglePlay}
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-rose-500 text-white transition-all duration-200 hover:bg-rose-600 active:scale-[0.95]"
        aria-label={isPlaying ? "Pause" : "Play"}
      >
        {isPlaying ? (
          <Pause className="h-3.5 w-3.5" strokeWidth={2.5} />
        ) : (
          <Play className="h-3.5 w-3.5 ml-0.5" strokeWidth={2.5} />
        )}
      </button>
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div
          onClick={handleSeek}
          className="h-1.5 w-full cursor-pointer rounded-full bg-gray-200"
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className="h-full rounded-full bg-rose-500 transition-[width] duration-100"
            style={{ width: `${progress}%` }}
          />
        </div>
        <div className="flex items-center gap-1">
          <Mic className="h-3 w-3 text-muted-foreground" strokeWidth={1.5} />
          <span className="font-mono text-[10px] text-muted-foreground">
            {displayTime}
          </span>
        </div>
      </div>
    </div>
  )
}

export function isImageMimeType(metadata: Record<string, unknown>): boolean {
  const mimeType = metadata.mime_type as string | undefined
  return !!mimeType && mimeType.startsWith("image/")
}
