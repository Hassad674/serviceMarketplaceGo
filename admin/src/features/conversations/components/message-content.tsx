import { FileText, Download, Mic } from "lucide-react"

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

  return (
    <div className="flex items-center gap-3 rounded-xl border border-border bg-muted/50 px-4 py-3">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-rose-100">
        <Mic className="h-4 w-4 text-rose-600" strokeWidth={2} />
      </div>
      <div className="flex flex-col gap-0.5">
        <div className="h-1.5 w-32 rounded-full bg-gray-200">
          <div className="h-full w-0 rounded-full bg-rose-400" />
        </div>
        <span className="font-mono text-[10px] text-muted-foreground">
          {formatDuration(voice.duration)}
        </span>
      </div>
    </div>
  )
}

export function isImageMimeType(metadata: Record<string, unknown>): boolean {
  const mimeType = metadata.mime_type as string | undefined
  return !!mimeType && mimeType.startsWith("image/")
}
