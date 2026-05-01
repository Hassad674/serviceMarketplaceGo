"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { createPortal } from "react-dom"
import {
  UploadCloud,
  X,
  File as FileIcon,
  FileText,
  FileImage,
  Loader2,
  Trash2,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
const MAX_FILE_SIZE = 10 * 1024 * 1024 // 10MB
const MAX_FILES = 5
const BYTES_PER_MB = 1024 * 1024

const ALLOWED_TYPES = [
  // Images
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
  "image/svg+xml",
  // PDF
  "application/pdf",
  // Microsoft Office
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.ms-excel",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.ms-powerpoint",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  // LibreOffice / OpenDocument
  "application/vnd.oasis.opendocument.text",
  "application/vnd.oasis.opendocument.spreadsheet",
  "application/vnd.oasis.opendocument.presentation",
  // Text formats
  "text/plain",
  "text/csv",
  "text/markdown",
  "text/html",
  "text/xml",
  "application/rtf",
  // Data formats
  "application/json",
  "application/xml",
  // Archives
  "application/zip",
  "application/x-rar-compressed",
]

const ALLOWED_EXTENSIONS = [
  ".txt", ".csv", ".md", ".html", ".htm", ".xml", ".json", ".rtf",
  ".odt", ".ods", ".odp",
  ".pdf",
  ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
  ".zip", ".rar",
  ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg",
]

const BLOCKED_EXTENSIONS = [
  ".exe", ".sh", ".bat", ".cmd", ".ps1", ".php", ".jsp",
]

type SelectedFile = {
  file: File
  id: string
  previewUrl: string | null
}

interface FileUploadModalProps {
  open: boolean
  onClose: () => void
  onUploadFiles: (files: File[]) => Promise<void>
  uploading: boolean
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < BYTES_PER_MB) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / BYTES_PER_MB).toFixed(1)} MB`
}

function getFileIcon(mimeType: string) {
  if (mimeType.startsWith("image/")) return FileImage
  if (mimeType === "application/pdf") return FileText
  return FileIcon
}

function isAllowedFile(file: File): boolean {
  const ext = "." + file.name.split(".").pop()?.toLowerCase()
  if (BLOCKED_EXTENSIONS.includes(ext)) return false
  if (ALLOWED_TYPES.includes(file.type)) return true
  if (file.type.startsWith("image/")) return true
  // Fallback: some browsers report empty or generic MIME types for certain
  // file formats (e.g. .odt, .md, .json). Check the extension as well.
  if (ALLOWED_EXTENSIONS.includes(ext)) return true
  return false
}

export function FileUploadModal({
  open,
  onClose,
  onUploadFiles,
  uploading,
}: FileUploadModalProps) {
  const [selectedFiles, setSelectedFiles] = useState<SelectedFile[]>([])
  const [error, setError] = useState<string | null>(null)
  const [isDragOver, setIsDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const modalRef = useRef<HTMLDivElement>(null)
  const t = useTranslations("messaging")
  const tCommon = useTranslations("common")

  // Reset on open/close. Tracking `open` in render-time state lets us
  // tear down preview URLs and reset local state during the render that
  // observes the close, without setState-in-effect.
  const [lastOpen, setLastOpen] = useState(open)
  if (lastOpen !== open) {
    setLastOpen(open)
    if (!open) {
      selectedFiles.forEach((sf) => {
        if (sf.previewUrl) URL.revokeObjectURL(sf.previewUrl)
      })
      setSelectedFiles([])
      setError(null)
      setIsDragOver(false)
    }
  }

  // Close on Escape + focus trap (Tab cycles within the modal)
  useEffect(() => {
    if (!open) return

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && !uploading) {
        onClose()
        return
      }

      // Focus trap: cycle Tab within the modal
      if (event.key === "Tab" && modalRef.current) {
        const focusable = modalRef.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
        )
        if (focusable.length === 0) return

        const first = focusable[0]
        const last = focusable[focusable.length - 1]

        if (event.shiftKey) {
          if (document.activeElement === first) {
            event.preventDefault()
            last.focus()
          }
        } else {
          if (document.activeElement === last) {
            event.preventDefault()
            first.focus()
          }
        }
      }
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [open, uploading, onClose])

  // Focus modal on open
  useEffect(() => {
    if (open) modalRef.current?.focus()
  }, [open])

  const validateFiles = useCallback(
    (files: File[]): { valid: File[]; errorMsg: string | null } => {
      const totalCount = selectedFiles.length + files.length
      if (totalCount > MAX_FILES) {
        return {
          valid: [],
          errorMsg: t("maxFilesExceeded", { max: MAX_FILES }),
        }
      }

      const valid: File[] = []
      for (const file of files) {
        if (file.size > MAX_FILE_SIZE) {
          return {
            valid: [],
            errorMsg: t("fileTooLarge", {
              filename: file.name,
              maxSize: "10 MB",
            }),
          }
        }
        if (!isAllowedFile(file)) {
          return {
            valid: [],
            errorMsg: t("invalidFileType", { filename: file.name }),
          }
        }
        valid.push(file)
      }
      return { valid, errorMsg: null }
    },
    [selectedFiles.length, t],
  )

  function addFiles(files: File[]) {
    const { valid, errorMsg } = validateFiles(files)
    if (errorMsg) {
      setError(errorMsg)
      return
    }
    setError(null)

    const newSelected: SelectedFile[] = valid.map((file) => ({
      file,
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      previewUrl: file.type.startsWith("image/")
        ? URL.createObjectURL(file)
        : null,
    }))

    setSelectedFiles((prev) => [...prev, ...newSelected])
  }

  function removeFile(id: string) {
    setSelectedFiles((prev) => {
      const removed = prev.find((f) => f.id === id)
      if (removed?.previewUrl) URL.revokeObjectURL(removed.previewUrl)
      return prev.filter((f) => f.id !== id)
    })
    setError(null)
  }

  function handleInputChange(event: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(event.target.files ?? [])
    if (files.length > 0) addFiles(files)
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
    const files = Array.from(event.dataTransfer.files)
    if (files.length > 0) addFiles(files)
  }

  async function handleUpload() {
    if (selectedFiles.length === 0) return
    const files = selectedFiles.map((sf) => sf.file)
    await onUploadFiles(files)
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
        aria-label={t("fileUploadTitle")}
        tabIndex={-1}
        className={cn(
          "relative bg-card rounded-xl shadow-lg w-full max-w-lg mx-4 p-6",
          "animate-[fadeSlideUp_150ms_ease-out]",
          "focus:outline-none",
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">
            {t("fileUploadTitle")}
          </h2>
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

        <p className="text-sm text-muted-foreground mb-4">
          {t("fileUploadDesc")}
        </p>

        {/* Drop zone — show when less than max files */}
        {selectedFiles.length < MAX_FILES && (
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => fileInputRef.current?.click()}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            disabled={uploading}
            className={cn(
              "w-full border-2 border-dashed rounded-lg p-6",
              "flex flex-col items-center justify-center gap-2",
              "transition-colors cursor-pointer",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              "disabled:opacity-50 disabled:cursor-not-allowed",
              isDragOver
                ? "border-primary bg-primary/5"
                : "border-border hover:border-primary hover:bg-primary/5",
            )}
            aria-label={t("dropZoneFiles")}
          >
            <div
              className={cn(
                "w-10 h-10 rounded-full flex items-center justify-center",
                isDragOver ? "bg-primary/10" : "bg-muted",
              )}
            >
              <UploadCloud
                className={cn(
                  "w-5 h-5",
                  isDragOver ? "text-primary" : "text-muted-foreground",
                )}
                aria-hidden="true"
              />
            </div>
            <p className="text-sm font-medium text-foreground">
              {t("dragFiles")}
            </p>
            <p className="text-xs text-muted-foreground">
              {t("filesMaxInfo")}
            </p>
          </Button>
        )}

        {/* Selected files list */}
        {selectedFiles.length > 0 && (
          <div className="mt-4 space-y-2 max-h-48 overflow-y-auto">
            {selectedFiles.map((sf) => {
              const Icon = getFileIcon(sf.file.type)
              return (
                <div
                  key={sf.id}
                  className="flex items-center gap-3 rounded-lg border border-border p-3"
                >
                  {sf.previewUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element -- previewUrl is a blob: URL, not a remote asset
                    <img
                      src={sf.previewUrl}
                      alt={sf.file.name}
                      className="w-10 h-10 rounded-md object-cover shrink-0"
                    />
                  ) : (
                    <div className="w-10 h-10 rounded-md bg-muted flex items-center justify-center shrink-0">
                      <Icon
                        className="w-5 h-5 text-muted-foreground"
                        aria-hidden="true"
                      />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">
                      {sf.file.name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {formatFileSize(sf.file.size)}
                    </p>
                  </div>
                  <Button variant="ghost" size="auto"
                    type="button"
                    onClick={() => removeFile(sf.id)}
                    disabled={uploading}
                    className={cn(
                      "rounded-md p-1.5 text-muted-foreground hover:text-destructive",
                      "hover:bg-destructive/10 transition-colors",
                      "focus-visible:outline-2 focus-visible:outline-ring",
                      "disabled:opacity-50 disabled:cursor-not-allowed",
                    )}
                    aria-label={tCommon("removeFile")}
                  >
                    <Trash2 className="w-4 h-4" aria-hidden="true" />
                  </Button>
                </div>
              )
            })}
          </div>
        )}

        {/* Error */}
        {error && (
          <p className="text-sm text-destructive mt-3" role="alert">
            {error}
          </p>
        )}

        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          multiple
          onChange={handleInputChange}
          className="hidden"
          aria-label={t("selectFiles")}
        />

        {/* Actions */}
        <div className="flex items-center justify-between mt-6">
          <p className="text-xs text-muted-foreground">
            {t("filesCount", { count: selectedFiles.length, max: MAX_FILES })}
          </p>
          <div className="flex items-center gap-3">
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
              disabled={selectedFiles.length === 0 || uploading}
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
              {uploading ? t("uploadingFiles") : t("sendFiles")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )

  if (typeof document !== "undefined") {
    return createPortal(modalContent, document.body)
  }

  return modalContent
}
