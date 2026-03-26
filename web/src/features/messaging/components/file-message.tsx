"use client"

import { FileText, Download } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import type { FileMetadata } from "../types"

interface FileMessageProps {
  metadata: FileMetadata
  isOwn: boolean
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function FileMessage({ metadata, isOwn }: FileMessageProps) {
  const isImage = metadata.mime_type.startsWith("image/")

  if (isImage) {
    return (
      <div className="block overflow-hidden rounded-lg">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={metadata.url}
          alt={metadata.filename}
          className="max-h-64 max-w-full rounded-lg object-cover"
        />
        <a
          href={metadata.url}
          download={metadata.filename}
          className={cn(
            "mt-1 flex items-center gap-1 text-[10px] hover:underline",
            isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
          )}
        >
          <Download className="h-3 w-3 shrink-0" strokeWidth={1.5} />
          {metadata.filename} ({formatFileSize(metadata.size)})
        </a>
      </div>
    )
  }

  return (
    <a
      href={metadata.url}
      download={metadata.filename}
      className={cn(
        "flex items-center gap-3 rounded-lg p-3 transition-colors",
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
    </a>
  )
}
