"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { createPortal } from "react-dom"
import { UploadCloud, X, File as FileIcon, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
interface UploadModalProps {
  open: boolean
  onClose: () => void
  onUpload: (file: File) => Promise<void>
  accept: string
  maxSize: number
  title: string
  description?: string
  uploading?: boolean
}

const BYTES_PER_MB = 1024 * 1024

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < BYTES_PER_MB) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / BYTES_PER_MB).toFixed(1)} MB`
}

export function UploadModal({
  open,
  onClose,
  onUpload,
  accept,
  maxSize,
  title,
  description,
  uploading = false,
}: UploadModalProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isDragOver, setIsDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const modalRef = useRef<HTMLDivElement>(null)
  const t = useTranslations("upload")
  const tCommon = useTranslations("common")

  const isImage = accept.startsWith("image")
  const maxSizeLabel = formatFileSize(maxSize)

  // Reset state when modal opens/closes. Render-time tracking of `open`
  // avoids the setState-in-effect cascade.
  const [lastOpen, setLastOpen] = useState(open)
  if (lastOpen !== open) {
    setLastOpen(open)
    if (!open) {
      setSelectedFile(null)
      setPreviewUrl(null)
      setError(null)
      setIsDragOver(false)
    }
  }

  // Clean up preview URL on unmount or change
  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    }
  }, [previewUrl])

  // Close on Escape key
  useEffect(() => {
    if (!open) return
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && !uploading) onClose()
    }
    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [open, uploading, onClose])

  // Focus trap: focus modal on open
  useEffect(() => {
    if (open) modalRef.current?.focus()
  }, [open])

  const validateFile = useCallback(
    (file: File): string | null => {
      if (file.size > maxSize) {
        return t("fileTooLarge", { maxSize: maxSizeLabel })
      }
      const acceptTypes = accept.split(",").map((s) => s.trim())
      const matchesType = acceptTypes.some((type) => {
        if (type.endsWith("/*")) {
          return file.type.startsWith(type.replace("/*", "/"))
        }
        return file.type === type
      })
      if (!matchesType) {
        return isImage
          ? t("invalidImageType")
          : t("invalidVideoType")
      }
      return null
    },
    [accept, maxSize, maxSizeLabel, isImage, t],
  )

  function handleFileSelect(file: File) {
    const validationError = validateFile(file)
    if (validationError) {
      setError(validationError)
      setSelectedFile(null)
      setPreviewUrl(null)
      return
    }
    setError(null)
    setSelectedFile(file)
    if (isImage) {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      setPreviewUrl(URL.createObjectURL(file))
    }
  }

  function handleInputChange(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (file) handleFileSelect(file)
    // Reset input so re-selecting the same file triggers change
    event.target.value = ""
  }

  function handleDragOver(event: React.DragEvent) {
    event.preventDefault()
    setIsDragOver(true)
  }

  function handleDragLeave(event: React.DragEvent) {
    event.preventDefault()
    setIsDragOver(false)
  }

  function handleDrop(event: React.DragEvent) {
    event.preventDefault()
    setIsDragOver(false)
    const file = event.dataTransfer.files[0]
    if (file) handleFileSelect(file)
  }

  function handleRemoveFile() {
    setSelectedFile(null)
    if (previewUrl) URL.revokeObjectURL(previewUrl)
    setPreviewUrl(null)
    setError(null)
  }

  async function handleUpload() {
    if (!selectedFile) return
    await onUpload(selectedFile)
  }

  function handleOverlayClick(event: React.MouseEvent) {
    if (event.target === event.currentTarget && !uploading) {
      onClose()
    }
  }

  if (!open) return null

  const modalContent = (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={handleOverlayClick}
      role="presentation"
    >
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        className={cn(
          "relative bg-card rounded-xl shadow-lg w-full max-w-md mx-4 p-6",
          "animate-[fadeSlideUp_150ms_ease-out]",
          "focus:outline-none",
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">{title}</h2>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onClose}
            disabled={uploading}
            className={cn(
              "rounded-md p-1.5 text-muted-foreground hover:text-foreground",
              "hover:bg-muted transition-colors",
              "focus-visible:outline-2 focus-visible:outline-ring",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
            aria-label={tCommon("close")}
          >
            <X className="w-5 h-5" aria-hidden="true" />
          </Button>
        </div>

        {description && (
          <p className="text-sm text-muted-foreground mb-4">{description}</p>
        )}

        {/* Drop zone */}
        {!selectedFile && (
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => fileInputRef.current?.click()}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            className={cn(
              "w-full border-2 border-dashed rounded-lg p-8",
              "flex flex-col items-center justify-center gap-3",
              "transition-colors cursor-pointer",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              isDragOver
                ? "border-primary bg-primary/5"
                : "border-border hover:border-primary hover:bg-primary/5",
            )}
            aria-label={isImage ? t("dropZoneImage") : t("dropZoneVideo")}
          >
            <div
              className={cn(
                "w-12 h-12 rounded-full flex items-center justify-center",
                isDragOver ? "bg-primary/10" : "bg-muted",
              )}
            >
              <UploadCloud
                className={cn(
                  "w-6 h-6",
                  isDragOver ? "text-primary" : "text-muted-foreground",
                )}
                aria-hidden="true"
              />
            </div>
            <div className="text-center">
              <p className="text-sm font-medium text-foreground">
                {t("dragFile")}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                {t("orClickBrowse")}
              </p>
            </div>
            <p className="text-xs text-muted-foreground">
              {isImage
                ? t("imagesMaxSize", { maxSize: maxSizeLabel })
                : t("videosMaxSize", { maxSize: maxSizeLabel })}
            </p>
          </Button>
        )}

        {/* File preview */}
        {selectedFile && (
          <div className="border border-border rounded-lg p-4">
            {isImage && previewUrl ? (
              <div className="relative mb-3">
                {/* eslint-disable-next-line @next/next/no-img-element -- previewUrl is a blob: URL */}
                <img
                  src={previewUrl}
                  alt={tCommon("previewFile")}
                  className="w-full h-40 object-cover rounded-md"
                />
              </div>
            ) : null}
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-md bg-muted flex items-center justify-center shrink-0">
                <FileIcon
                  className="w-5 h-5 text-muted-foreground"
                  aria-hidden="true"
                />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {selectedFile.name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatFileSize(selectedFile.size)}
                </p>
              </div>
              <Button variant="ghost" size="auto"
                type="button"
                onClick={handleRemoveFile}
                disabled={uploading}
                className={cn(
                  "rounded-md p-1.5 text-muted-foreground hover:text-foreground",
                  "hover:bg-muted transition-colors",
                  "focus-visible:outline-2 focus-visible:outline-ring",
                  "disabled:opacity-50 disabled:cursor-not-allowed",
                )}
                aria-label={tCommon("removeFile")}
              >
                <X className="w-4 h-4" aria-hidden="true" />
              </Button>
            </div>
          </div>
        )}

        {/* Error message */}
        {error && (
          <p className="text-sm text-destructive mt-3" role="alert">
            {error}
          </p>
        )}

        {/* Hidden file input */}
        <Input
          ref={fileInputRef}
          type="file"
          accept={accept}
          onChange={handleInputChange}
          className="hidden"
          aria-label={isImage ? t("selectImage") : t("selectVideo")}
        />

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 mt-6">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onClose}
            disabled={uploading}
            className={cn(
              "h-10 px-4 text-sm font-medium rounded-md",
              "text-foreground hover:bg-muted transition-colors",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {tCommon("cancel")}
          </Button>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={handleUpload}
            disabled={!selectedFile || uploading}
            className={cn(
              "h-10 px-4 text-sm font-medium rounded-md",
              "bg-primary text-primary-foreground",
              "hover:opacity-90 transition-opacity",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              "disabled:opacity-50 disabled:cursor-not-allowed",
              "inline-flex items-center gap-2",
            )}
          >
            {uploading && (
              <Loader2
                className="w-4 h-4 animate-spin"
                aria-hidden="true"
              />
            )}
            {uploading ? t("uploading") : t("send")}
          </Button>
        </div>
      </div>
    </div>
  )

  // Portal to document.body to escape stacking contexts created by
  // backdrop-filter / overflow-hidden in the dashboard shell layout
  if (typeof document !== "undefined") {
    return createPortal(modalContent, document.body)
  }

  return modalContent
}
