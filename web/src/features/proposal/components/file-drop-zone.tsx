"use client"

import { useState, useRef, useCallback } from "react"
import { Upload, X, FileIcon } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
interface FileDropZoneProps {
  files: File[]
  onFilesChange: (files: File[]) => void
}

export function FileDropZone({ files, onFilesChange }: FileDropZoneProps) {
  const t = useTranslations("proposal")
  const [isDragging, setIsDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      setIsDragging(false)
      const dropped = Array.from(e.dataTransfer.files)
      onFilesChange([...files, ...dropped])
    },
    [files, onFilesChange],
  )

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!e.target.files) return
      const selected = Array.from(e.target.files)
      onFilesChange([...files, ...selected])
      e.target.value = ""
    },
    [files, onFilesChange],
  )

  function handleRemove(index: number) {
    onFilesChange(files.filter((_, i) => i !== index))
  }

  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  return (
    <div className="space-y-3">
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
        className={cn(
          "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed px-6 py-8",
          "transition-all duration-200",
          isDragging
            ? "border-rose-400 bg-rose-50 dark:border-rose-500 dark:bg-rose-500/10"
            : "border-gray-200 bg-gray-50/50 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600",
        )}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault()
            inputRef.current?.click()
          }
        }}
        aria-label={t("proposalDocumentsHint")}
      >
        <Upload
          className={cn(
            "h-6 w-6",
            isDragging
              ? "text-rose-500"
              : "text-gray-400 dark:text-gray-500",
          )}
          strokeWidth={1.5}
        />
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("proposalDocumentsHint")}
        </p>
      </div>

      <Input
        ref={inputRef}
        type="file"
        multiple
        onChange={handleInputChange}
        className="hidden"
        aria-hidden="true"
      />

      {/* File list */}
      {files.length > 0 && (
        <div className="space-y-2">
          {files.map((file, index) => (
            <div
              key={`${file.name}-${index}`}
              className={cn(
                "flex items-center gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2",
                "dark:border-gray-700 dark:bg-gray-800",
              )}
            >
              <FileIcon className="h-4 w-4 shrink-0 text-gray-400" strokeWidth={1.5} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm text-gray-700 dark:text-gray-300">
                  {file.name}
                </p>
                <p className="text-xs text-gray-400">{formatFileSize(file.size)}</p>
              </div>
              <Button variant="ghost" size="auto"
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  handleRemove(index)
                }}
                className="shrink-0 rounded-md p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                aria-label={`Remove ${file.name}`}
              >
                <X className="h-3.5 w-3.5" strokeWidth={2} />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
