"use client"

import Image from "next/image"
import { FileText, Download } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import type { FileMetadata } from "../types"
import { Button } from "@/shared/components/ui/button"

interface FileMessageProps {
  metadata: FileMetadata
  isOwn: boolean
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

async function downloadFile(url: string, filename: string) {
  try {
    const response = await fetch(url)
    const blob = await response.blob()
    const blobUrl = URL.createObjectURL(blob)
    const link = document.createElement("a")
    link.href = blobUrl
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(blobUrl)
  } catch {
    window.open(url, "_blank")
  }
}

export function FileMessage({ metadata, isOwn }: FileMessageProps) {
  const isImage = metadata.mime_type.startsWith("image/")

  if (isImage) {
    return (
      <div className="block overflow-hidden rounded-lg">
        <Image
          src={metadata.url}
          alt={metadata.filename}
          width={320}
          height={256}
          className="max-h-64 max-w-full rounded-lg object-cover"
          unoptimized
        />
        <Button variant="ghost" size="auto"
          type="button"
          onClick={() => downloadFile(metadata.url, metadata.filename)}
          className={cn(
            "mt-1 flex items-center gap-1 text-[10px] hover:underline",
            isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
          )}
        >
          <Download className="h-3 w-3 shrink-0" strokeWidth={1.5} />
          {metadata.filename} ({formatFileSize(metadata.size)})
        </Button>
      </div>
    )
  }

  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={() => downloadFile(metadata.url, metadata.filename)}
      className={cn(
        "flex w-full items-center gap-3 rounded-lg p-3 text-left transition-colors",
        isOwn
          ? "bg-rose-600/30 hover:bg-rose-600/40"
          : "bg-gray-200/50 hover:bg-gray-200/70 dark:bg-gray-700/50 dark:hover:bg-gray-700/70",
      )}
    >
      <div
        className={cn(
          "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg",
          isOwn ? "bg-rose-400/30" : "bg-gray-300/50 dark:bg-gray-600/50",
        )}
      >
        <FileText
          className={cn(
            "h-5 w-5",
            isOwn ? "text-white" : "text-gray-600 dark:text-gray-300",
          )}
          strokeWidth={1.5}
        />
      </div>
      <div className="min-w-0 flex-1">
        <p
          className={cn(
            "truncate text-sm font-medium",
            isOwn ? "text-white" : "text-gray-900 dark:text-gray-100",
          )}
        >
          {metadata.filename}
        </p>
        <p
          className={cn(
            "text-[10px]",
            isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
          )}
        >
          {formatFileSize(metadata.size)}
        </p>
      </div>
      <Download
        className={cn(
          "h-4 w-4 shrink-0",
          isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
        )}
        strokeWidth={1.5}
      />
    </Button>
  )
}
