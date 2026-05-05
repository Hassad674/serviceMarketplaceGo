"use client"

import { useState, useRef, useCallback } from "react"
import { Upload, X, FileIcon } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

// Soleil v2 — File drop zone.
// Dashed corail border, ivoire bg, corail-soft icon plate, Fraunces
// nothing here (file picker stays UI-y), Inter Tight body.

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
          "flex cursor-pointer flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed px-6 py-10",
          "transition-all duration-200 ease-out",
          isDragging
            ? "border-primary bg-primary-soft"
            : "border-border-strong bg-background hover:border-primary hover:bg-primary-soft/40",
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
        <div
          className={cn(
            "flex h-12 w-12 items-center justify-center rounded-full",
            "bg-primary-soft text-primary",
          )}
          aria-hidden="true"
        >
          <Upload className="h-5 w-5" strokeWidth={1.7} />
        </div>
        <p className="text-center text-[13.5px] text-muted-foreground">
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
                "flex items-center gap-3 rounded-xl border border-border bg-card px-3.5 py-2.5",
              )}
            >
              <FileIcon
                className="h-4 w-4 shrink-0 text-subtle-foreground"
                strokeWidth={1.7}
                aria-hidden="true"
              />
              <div className="min-w-0 flex-1">
                <p className="truncate text-[13.5px] font-medium text-foreground">
                  {file.name}
                </p>
                <p className="font-mono text-[11px] text-subtle-foreground">
                  {formatFileSize(file.size)}
                </p>
              </div>
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  handleRemove(index)
                }}
                className={cn(
                  "shrink-0 rounded-full p-1.5 text-subtle-foreground",
                  "transition-colors duration-150",
                  "hover:bg-primary-soft hover:text-primary",
                )}
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
