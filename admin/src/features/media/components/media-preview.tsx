import { FileText, Film, Music } from "lucide-react"
import { cn } from "@/shared/lib/utils"

type MediaPreviewProps = {
  fileUrl: string
  fileType: string
  fileName: string
  className?: string
  size?: "sm" | "md" | "lg"
}

const sizeClasses = {
  sm: "h-12 w-12",
  md: "h-32 w-32",
  lg: "h-64 w-full max-w-md",
} as const

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
      <img
        src={fileUrl}
        alt={fileName}
        className={cn("rounded-lg object-cover", sizeClass, className)}
        loading="lazy"
      />
    )
  }

  if (fileType.startsWith("video/")) {
    if (size === "sm") {
      return (
        <div className={cn("flex items-center justify-center rounded-lg bg-muted", sizeClass, className)}>
          <Film className="h-5 w-5 text-muted-foreground" />
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

  if (fileType.startsWith("audio/")) {
    return (
      <div className={cn("flex items-center justify-center rounded-lg bg-muted", sizeClass, className)}>
        <Music className="h-5 w-5 text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className={cn("flex items-center justify-center rounded-lg bg-muted", sizeClass, className)}>
      <FileText className="h-5 w-5 text-muted-foreground" />
    </div>
  )
}
