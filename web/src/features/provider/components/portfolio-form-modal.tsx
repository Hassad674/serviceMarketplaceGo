"use client"

import { useState, useCallback, useEffect, useRef } from "react"
import {
  X,
  Upload,
  Trash2,
  Loader2,
  Film,
  Star,
  Link2,
  ImagePlus,
  Camera,
  RotateCcw,
} from "lucide-react"
import { useTranslations } from "next-intl"
import type { PortfolioItem } from "../api/portfolio-api"
import {
  useCreatePortfolioItem,
  useUpdatePortfolioItem,
  useUploadPortfolioImage,
  useUploadPortfolioVideo,
} from "../hooks/use-portfolio"

interface PortfolioFormModalProps {
  item?: PortfolioItem
  open: boolean
  onClose: () => void
  nextPosition: number
}

type LocalMedia = {
  media_url: string
  media_type: "image" | "video"
  thumbnail_url: string
  position: number
}

const TITLE_MAX = 200
const DESC_MAX = 2000
const MEDIA_MAX = 8

function normalizeUrl(raw: string): string {
  const trimmed = raw.trim()
  if (!trimmed) return ""
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  return `https://${trimmed}`
}

export function PortfolioFormModal({
  item,
  open,
  onClose,
  nextPosition,
}: PortfolioFormModalProps) {
  const t = useTranslations("portfolio")
  const isEdit = !!item

  const [title, setTitle] = useState(item?.title ?? "")
  const [description, setDescription] = useState(item?.description ?? "")
  const [linkUrl, setLinkUrl] = useState(item?.link_url ?? "")
  const [media, setMedia] = useState<LocalMedia[]>(
    item?.media?.map((m) => ({
      media_url: m.media_url,
      media_type: m.media_type,
      thumbnail_url: m.thumbnail_url ?? "",
      position: m.position,
    })) ?? [],
  )
  const [uploading, setUploading] = useState(false)
  const [dragOverZone, setDragOverZone] = useState(false)
  const [draggedMediaIdx, setDraggedMediaIdx] = useState<number | null>(null)
  const [customThumbnailFor, setCustomThumbnailFor] = useState<number | null>(null)
  const customThumbnailInputRef = useRef<HTMLInputElement>(null)

  const createItem = useCreatePortfolioItem()
  const updateItem = useUpdatePortfolioItem()
  const uploadImage = useUploadPortfolioImage()
  const uploadVideo = useUploadPortfolioVideo()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const saving = createItem.isPending || updateItem.isPending

  // Reset form when item changes (edit different items)
  useEffect(() => {
    if (open) {
      setTitle(item?.title ?? "")
      setDescription(item?.description ?? "")
      setLinkUrl(item?.link_url ?? "")
      setMedia(
        item?.media?.map((m) => ({
          media_url: m.media_url,
          media_type: m.media_type,
          thumbnail_url: m.thumbnail_url ?? "",
          position: m.position,
        })) ?? [],
      )
    }
  }, [item, open])

  // Escape key closes modal
  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose()
    }
    window.addEventListener("keydown", handleKey)
    return () => window.removeEventListener("keydown", handleKey)
  }, [open, onClose])

  const uploadFiles = useCallback(
    async (files: FileList | File[]) => {
      const fileArr = Array.from(files)
      if (fileArr.length === 0) return

      setUploading(true)
      try {
        const results: LocalMedia[] = []
        for (let i = 0; i < fileArr.length && media.length + results.length < MEDIA_MAX; i++) {
          const file = fileArr[i]
          const isVideo = file.type.startsWith("video/")
          const result = isVideo
            ? await uploadVideo.mutateAsync(file)
            : await uploadImage.mutateAsync(file)

          results.push({
            media_url: result.url,
            media_type: isVideo ? "video" : "image",
            thumbnail_url: "",
            position: media.length + results.length,
          })
        }
        setMedia((prev) => [...prev, ...results])
      } finally {
        setUploading(false)
      }
    },
    [media.length, uploadImage, uploadVideo],
  )

  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      if (e.target.files) await uploadFiles(e.target.files)
      if (e.target) e.target.value = ""
    },
    [uploadFiles],
  )

  const handleZoneDrop = useCallback(
    async (e: React.DragEvent) => {
      e.preventDefault()
      setDragOverZone(false)
      if (e.dataTransfer.files) await uploadFiles(e.dataTransfer.files)
    },
    [uploadFiles],
  )

  const removeMedia = useCallback((index: number) => {
    setMedia((prev) =>
      prev.filter((_, i) => i !== index).map((m, i) => ({ ...m, position: i })),
    )
  }, [])

  // Custom thumbnail handlers (videos only)
  const openCustomThumbnailPicker = useCallback((idx: number) => {
    setCustomThumbnailFor(idx)
    setTimeout(() => customThumbnailInputRef.current?.click(), 0)
  }, [])

  const handleCustomThumbnailSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      const targetIdx = customThumbnailFor
      e.target.value = ""
      if (!file || targetIdx === null) {
        setCustomThumbnailFor(null)
        return
      }

      setUploading(true)
      try {
        const result = await uploadImage.mutateAsync(file)
        setMedia((prev) =>
          prev.map((m, i) =>
            i === targetIdx ? { ...m, thumbnail_url: result.url } : m,
          ),
        )
      } finally {
        setUploading(false)
        setCustomThumbnailFor(null)
      }
    },
    [customThumbnailFor, uploadImage],
  )

  const removeCustomThumbnail = useCallback((idx: number) => {
    setMedia((prev) =>
      prev.map((m, i) => (i === idx ? { ...m, thumbnail_url: "" } : m)),
    )
  }, [])

  // Drag-to-reorder media
  const handleMediaDragStart = (idx: number) => setDraggedMediaIdx(idx)
  const handleMediaDragOver = (e: React.DragEvent) => e.preventDefault()
  const handleMediaDrop = (targetIdx: number) => {
    if (draggedMediaIdx === null || draggedMediaIdx === targetIdx) return
    setMedia((prev) => {
      const next = [...prev]
      const [moved] = next.splice(draggedMediaIdx, 1)
      next.splice(targetIdx, 0, moved)
      return next.map((m, i) => ({ ...m, position: i }))
    })
    setDraggedMediaIdx(null)
  }

  const handleSubmit = async () => {
    if (!title.trim()) return

    const payload = {
      title: title.trim(),
      description: description.trim(),
      link_url: normalizeUrl(linkUrl),
      media: media.map((m, i) => ({
        media_url: m.media_url,
        media_type: m.media_type,
        thumbnail_url: m.thumbnail_url || undefined,
        position: i,
      })),
    }

    if (isEdit && item) {
      await updateItem.mutateAsync({ id: item.id, ...payload })
    } else {
      await createItem.mutateAsync({ ...payload, position: nextPosition })
    }
    onClose()
  }

  if (!open) return null

  const titleColor =
    title.length > TITLE_MAX ? "text-red-500" : title.length > TITLE_MAX * 0.85 ? "text-amber-500" : "text-muted-foreground"
  const descColor =
    description.length > DESC_MAX ? "text-red-500" : description.length > DESC_MAX * 0.85 ? "text-amber-500" : "text-muted-foreground"

  return (
    <div
      className="fixed inset-0 z-50 flex items-stretch justify-center bg-black/70 backdrop-blur-md animate-fade-in sm:items-center sm:p-4"
      onClick={onClose}
    >
      <div
        className="relative flex h-full w-full flex-col overflow-hidden bg-card shadow-2xl animate-scale-in sm:h-auto sm:max-h-[92vh] sm:max-w-2xl sm:rounded-3xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex shrink-0 items-center justify-between border-b border-border px-5 py-4 sm:px-7 sm:py-5">
          <div className="min-w-0 flex-1 pr-3">
            <h2 className="truncate text-lg font-semibold tracking-tight text-foreground sm:text-xl">
              {isEdit ? t("editProject") : t("addProject")}
            </h2>
            <p className="mt-0.5 hidden text-xs text-muted-foreground sm:block">
              {isEdit ? t("editProjectSubtitle") : t("addProjectSubtitle")}
            </p>
          </div>
          <button
            onClick={onClose}
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label={t("close")}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Form body */}
        <div className="flex-1 space-y-5 overflow-y-auto px-5 py-5 sm:space-y-6 sm:px-7 sm:py-6">
          {/* Title */}
          <div>
            <label className="mb-2 flex items-center justify-between text-sm font-medium text-foreground">
              <span>
                {t("title")} <span className="text-rose-500">*</span>
              </span>
              <span className={`text-xs font-normal ${titleColor}`}>
                {title.length}/{TITLE_MAX}
              </span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("titlePlaceholder")}
              maxLength={TITLE_MAX}
              autoFocus
              className="h-11 w-full rounded-xl border border-border bg-background px-4 text-sm shadow-xs outline-none transition-all focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
            />
          </div>

          {/* Media — drop zone or grid */}
          <div>
            <label className="mb-2 flex items-center justify-between text-sm font-medium text-foreground">
              <span>{t("media")}</span>
              <span className="text-xs font-normal text-muted-foreground">
                {media.length}/{MEDIA_MAX}
              </span>
            </label>

            {media.length === 0 ? (
              // Empty state — large drop zone
              <label
                onDragOver={(e) => {
                  e.preventDefault()
                  setDragOverZone(true)
                }}
                onDragLeave={() => setDragOverZone(false)}
                onDrop={handleZoneDrop}
                className={`flex cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed py-12 px-6 text-center transition-all ${
                  dragOverZone
                    ? "border-rose-500 bg-rose-50/80 scale-[1.01]"
                    : "border-border bg-muted/30 hover:border-rose-300 hover:bg-rose-50/40"
                }`}
              >
                <div className="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-rose-100 to-rose-50">
                  <ImagePlus className="h-6 w-6 text-rose-600" />
                </div>
                <p className="text-sm font-medium text-foreground">
                  {t("dropZoneTitle")}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("dropZoneSubtitle")}
                </p>
                <input
                  type="file"
                  accept="image/*,video/*"
                  multiple
                  onChange={handleFileSelect}
                  className="hidden"
                  disabled={uploading}
                />
              </label>
            ) : (
              // Media grid with drag-to-reorder
              <div className="grid grid-cols-3 gap-3 sm:grid-cols-4">
                {media.map((m, i) => (
                  <div
                    key={`${m.media_url}-${i}`}
                    draggable
                    onDragStart={() => handleMediaDragStart(i)}
                    onDragOver={handleMediaDragOver}
                    onDrop={() => handleMediaDrop(i)}
                    className={`group/thumb relative aspect-square overflow-hidden rounded-xl border-2 bg-muted shadow-sm transition-all ${
                      draggedMediaIdx === i
                        ? "border-rose-500 opacity-50"
                        : "border-transparent hover:border-rose-300"
                    } ${i === 0 ? "ring-2 ring-rose-500/30" : ""}`}
                  >
                    {m.media_type === "video" ? (
                      <div className="relative h-full w-full bg-slate-900">
                        {m.thumbnail_url ? (
                          <img
                            src={m.thumbnail_url}
                            alt={`Media ${i + 1}`}
                            className="h-full w-full object-cover"
                          />
                        ) : (
                          <video
                            src={`${m.media_url}#t=0.1`}
                            preload="metadata"
                            muted
                            playsInline
                            className="h-full w-full object-cover"
                          />
                        )}
                        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-black/50 backdrop-blur-sm">
                            <Film className="h-4 w-4 text-white" />
                          </div>
                        </div>
                        {/* Custom thumbnail bar — always visible on video thumbs */}
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            if (m.thumbnail_url) {
                              removeCustomThumbnail(i)
                            } else {
                              openCustomThumbnailPicker(i)
                            }
                          }}
                          className={`absolute inset-x-0 bottom-0 flex items-center justify-center gap-1 px-2 py-1.5 text-[10px] font-semibold text-white backdrop-blur-md transition-colors ${
                            m.thumbnail_url
                              ? "bg-rose-600/90 hover:bg-rose-600"
                              : "bg-black/70 hover:bg-black/85"
                          }`}
                          title={m.thumbnail_url ? t("revertToAuto") : t("setCustomThumbnail")}
                        >
                          {m.thumbnail_url ? (
                            <>
                              <RotateCcw className="h-3 w-3" />
                              <span>{t("custom")}</span>
                            </>
                          ) : (
                            <>
                              <Camera className="h-3 w-3" />
                              <span>{t("customCover")}</span>
                            </>
                          )}
                        </button>
                      </div>
                    ) : (
                      <img
                        src={m.media_url}
                        alt={`Media ${i + 1}`}
                        className="h-full w-full object-cover"
                      />
                    )}

                    {/* Cover badge on first */}
                    {i === 0 && (
                      <div className="absolute left-1.5 top-1.5 flex items-center gap-1 rounded-full bg-gradient-to-r from-rose-500 to-rose-600 px-2 py-0.5 text-[10px] font-semibold text-white shadow-sm">
                        <Star className="h-2.5 w-2.5 fill-white" />
                        Cover
                      </div>
                    )}

                    {/* Delete button — top right */}
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        removeMedia(i)
                      }}
                      className="absolute right-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-red-500/90 text-white opacity-0 shadow-md backdrop-blur-sm transition-opacity hover:bg-red-600 group-hover/thumb:opacity-100"
                      title={t("removeMedia")}
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>

                  </div>
                ))}

                {/* Add more button */}
                {media.length < MEDIA_MAX && (
                  <label className="flex aspect-square cursor-pointer flex-col items-center justify-center gap-1 rounded-xl border-2 border-dashed border-border bg-muted/30 text-muted-foreground transition-all hover:border-rose-300 hover:bg-rose-50/50 hover:text-rose-600">
                    {uploading ? (
                      <Loader2 className="h-5 w-5 animate-spin" />
                    ) : (
                      <>
                        <Upload className="h-5 w-5" />
                        <span className="text-[10px] font-medium">{t("addMore")}</span>
                      </>
                    )}
                    <input
                      ref={fileInputRef}
                      type="file"
                      accept="image/*,video/*"
                      multiple
                      onChange={handleFileSelect}
                      className="hidden"
                      disabled={uploading}
                    />
                  </label>
                )}
              </div>
            )}

            {media.length > 0 && (
              <p className="mt-2 text-xs text-muted-foreground">
                {t("mediaHint")}
              </p>
            )}
          </div>

          {/* Description */}
          <div>
            <label className="mb-2 flex items-center justify-between text-sm font-medium text-foreground">
              <span>{t("description")}</span>
              <span className={`text-xs font-normal ${descColor}`}>
                {description.length}/{DESC_MAX}
              </span>
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("descriptionPlaceholder")}
              maxLength={DESC_MAX}
              rows={4}
              className="w-full resize-none rounded-xl border border-border bg-background px-4 py-3 text-sm shadow-xs outline-none transition-all focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
            />
          </div>

          {/* Link URL */}
          <div>
            <label className="mb-2 block text-sm font-medium text-foreground">
              {t("projectLink")}
            </label>
            <div className="relative">
              <Link2 className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                value={linkUrl}
                onChange={(e) => setLinkUrl(e.target.value)}
                placeholder="example.com"
                className="h-11 w-full rounded-xl border border-border bg-background pl-10 pr-4 text-sm shadow-xs outline-none transition-all focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
              />
            </div>
            <p className="mt-1.5 text-xs text-muted-foreground">{t("linkHint")}</p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex shrink-0 items-center justify-end gap-3 border-t border-border bg-muted/20 px-5 py-3 sm:px-7 sm:py-4">
          <button
            onClick={onClose}
            className="h-10 rounded-xl px-5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            {t("cancel")}
          </button>
          <button
            onClick={handleSubmit}
            disabled={!title.trim() || saving}
            className="flex h-10 items-center gap-2 rounded-xl bg-gradient-to-r from-rose-500 to-rose-600 px-6 text-sm font-semibold text-white shadow-md transition-all hover:shadow-lg hover:shadow-rose-500/30 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            {saving ? t("saving") : isEdit ? t("save") : t("create")}
          </button>
        </div>

        {/* Hidden input for custom video thumbnail uploads */}
        <input
          ref={customThumbnailInputRef}
          type="file"
          accept="image/*"
          onChange={handleCustomThumbnailSelect}
          className="hidden"
        />
      </div>
    </div>
  )
}
