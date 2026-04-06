import { useEffect, useState } from "react"
import { FileText, Film, Music, Play, FileQuestion } from "lucide-react"
import { cn } from "@/shared/lib/utils"

type MediaPreviewProps = {
  fileUrl: string
  fileType: string
  fileName: string
  className?: string
  size?: "sm" | "md" | "lg"
}

const sizeClasses = {
  sm: "h-20 w-20",
  md: "h-32 w-32",
  lg: "h-64 w-full max-w-md",
} as const

const TEXT_MIME_TYPES = new Set([
  "application/json",
  "application/xml",
  "text/csv",
  "text/plain",
  "text/markdown",
])

const MAX_TEXT_LENGTH = 10000

function isTextFile(fileType: string): boolean {
  return fileType.startsWith("text/") || TEXT_MIME_TYPES.has(fileType)
}

export function MediaPreview({
  fileUrl,
  fileType,
  fileName,
  className,
  size = "sm",
}: MediaPreviewProps) {
  const sizeClass = sizeClasses[size]

  if (fileType.startsWith("image/")) {
    return (
      <ImagePreview
        fileUrl={fileUrl}
        fileName={fileName}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  if (fileType.startsWith("video/")) {
    return (
      <VideoPreview
        fileUrl={fileUrl}
        size={size}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  if (fileType.startsWith("audio/")) {
    return (
      <IconPlaceholder
        icon={Music}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  if (fileType === "application/pdf") {
    return (
      <PdfPreview
        fileUrl={fileUrl}
        fileName={fileName}
        size={size}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  if (isTextFile(fileType)) {
    return (
      <TextPreview
        fileUrl={fileUrl}
        size={size}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  return (
    <UnknownPreview
      fileName={fileName}
      fileType={fileType}
      size={size}
      sizeClass={sizeClass}
      className={className}
    />
  )
}

type ImagePreviewProps = {
  fileUrl: string
  fileName: string
  sizeClass: string
  className?: string
}

function ImagePreview({ fileUrl, fileName, sizeClass, className }: ImagePreviewProps) {
  return (
    <img
      src={fileUrl}
      alt={fileName}
      className={cn("rounded-lg object-cover", sizeClass, className)}
      loading="lazy"
    />
  )
}

type VideoPreviewProps = {
  fileUrl: string
  size: "sm" | "md" | "lg"
  sizeClass: string
  className?: string
}

function VideoPreview({ fileUrl, size, sizeClass, className }: VideoPreviewProps) {
  if (size === "sm") {
    return (
      <div className={cn("relative flex items-center justify-center rounded-lg bg-muted", sizeClass, className)}>
        <Film className="h-6 w-6 text-muted-foreground" />
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-foreground/60">
            <Play className="h-3.5 w-3.5 fill-white text-white" />
          </div>
        </div>
      </div>
    )
  }
  return (
    <video
      src={fileUrl}
      controls
      className={cn("rounded-lg", sizeClass, className)}
    >
      <track kind="captions" />
    </video>
  )
}

type IconPlaceholderProps = {
  icon: typeof FileText
  sizeClass: string
  className?: string
}

function IconPlaceholder({ icon: Icon, sizeClass, className }: IconPlaceholderProps) {
  return (
    <div className={cn("flex items-center justify-center rounded-lg bg-muted", sizeClass, className)}>
      <Icon className="h-5 w-5 text-muted-foreground" />
    </div>
  )
}

type PdfPreviewProps = {
  fileUrl: string
  fileName: string
  size: "sm" | "md" | "lg"
  sizeClass: string
  className?: string
}

function PdfPreview({ fileUrl, fileName, size, sizeClass, className }: PdfPreviewProps) {
  if (size !== "lg") {
    return (
      <IconPlaceholder
        icon={FileText}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  return (
    <iframe
      src={fileUrl}
      title={fileName}
      className={cn("w-full h-[600px] rounded-lg border border-gray-200", className)}
    />
  )
}

type TextPreviewProps = {
  fileUrl: string
  size: "sm" | "md" | "lg"
  sizeClass: string
  className?: string
}

function TextPreview({ fileUrl, size, sizeClass, className }: TextPreviewProps) {
  if (size !== "lg") {
    return (
      <IconPlaceholder
        icon={FileText}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  return <TextFileContent fileUrl={fileUrl} className={className} />
}

type TextFileContentProps = {
  fileUrl: string
  className?: string
}

function TextFileContent({ fileUrl, className }: TextFileContentProps) {
  const [textContent, setTextContent] = useState<string | null>(null)
  const [isTruncated, setIsTruncated] = useState(false)

  useEffect(() => {
    fetch(fileUrl)
      .then((res) => res.text())
      .then((text) => {
        if (text.length > MAX_TEXT_LENGTH) {
          setTextContent(text.slice(0, MAX_TEXT_LENGTH))
          setIsTruncated(true)
        } else {
          setTextContent(text)
        }
      })
      .catch(() => setTextContent("Impossible de charger le fichier"))
  }, [fileUrl])

  if (textContent === null) {
    return (
      <div className={cn("w-full h-[600px] rounded-lg border border-gray-200 bg-gray-50 animate-pulse", className)} />
    )
  }

  return (
    <div className={cn("w-full", className)}>
      <pre className="w-full max-h-[600px] overflow-auto rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm font-mono whitespace-pre-wrap">
        {textContent}
      </pre>
      {isTruncated && (
        <p className="mt-2 text-xs text-muted-foreground text-center">
          Fichier tronqu{"é"} (trop volumineux)
        </p>
      )}
    </div>
  )
}

type UnknownPreviewProps = {
  fileName: string
  fileType: string
  size: "sm" | "md" | "lg"
  sizeClass: string
  className?: string
}

function UnknownPreview({ fileName, fileType, size, sizeClass, className }: UnknownPreviewProps) {
  if (size !== "lg") {
    return (
      <IconPlaceholder
        icon={FileText}
        sizeClass={sizeClass}
        className={className}
      />
    )
  }

  return (
    <div className={cn("flex w-full flex-col items-center justify-center gap-3 rounded-lg border border-gray-200 bg-muted/50 p-8", className)}>
      <FileQuestion className="h-12 w-12 text-muted-foreground" />
      <p className="max-w-xs truncate text-sm font-medium text-foreground">{fileName}</p>
      <p className="text-xs text-muted-foreground">{fileType}</p>
      <p className="text-sm text-muted-foreground">
        Aper{"ç"}u non disponible pour ce type de fichier
      </p>
    </div>
  )
}
