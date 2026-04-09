"use client"

import { useState, useEffect, useCallback } from "react"
import { X, ChevronLeft, ChevronRight, ExternalLink, Image as ImageIcon, Film } from "lucide-react"
import { useTranslations } from "next-intl"
import type { PortfolioItem } from "../api/portfolio-api"

interface PortfolioDetailModalProps {
  item: PortfolioItem | null
  open: boolean
  onClose: () => void
}

export function PortfolioDetailModal({
  item,
  open,
  onClose,
}: PortfolioDetailModalProps) {
  const [currentIndex, setCurrentIndex] = useState(0)
  const t = useTranslations("portfolio")

  // Reset index when item changes
  useEffect(() => {
    setCurrentIndex(0)
  }, [item?.id])

  const media = item ? [...item.media].sort((a, b) => a.position - b.position) : []
  const current = media[currentIndex]
  const hasPrev = currentIndex > 0
  const hasNext = currentIndex < media.length - 1

  const goPrev = useCallback(() => {
    setCurrentIndex((i) => Math.max(0, i - 1))
  }, [])

  const goNext = useCallback(() => {
    setCurrentIndex((i) => Math.min(media.length - 1, i + 1))
  }, [media.length])

  // Keyboard navigation
  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose()
      else if (e.key === "ArrowLeft") goPrev()
      else if (e.key === "ArrowRight") goNext()
    }
    window.addEventListener("keydown", handleKey)
    return () => window.removeEventListener("keydown", handleKey)
  }, [open, onClose, goPrev, goNext])

  if (!open || !item) return null

  const imageCount = media.filter((m) => m.media_type === "image").length
  const videoCount = media.filter((m) => m.media_type === "video").length

  return (
    <div
      className="fixed inset-0 z-50 flex items-stretch justify-center bg-black/85 backdrop-blur-md animate-fade-in md:items-center md:p-4"
      onClick={onClose}
    >
      <div
        className="relative flex h-full w-full flex-col overflow-hidden bg-card shadow-2xl animate-scale-in md:h-auto md:max-h-[92vh] md:max-w-6xl md:flex-row md:rounded-3xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button — always visible top right */}
        <button
          onClick={onClose}
          className="absolute right-3 top-3 z-10 flex h-10 w-10 items-center justify-center rounded-full bg-black/60 text-white backdrop-blur-md transition-all hover:scale-110 hover:bg-black/80 md:right-4 md:top-4"
          aria-label={t("close")}
        >
          <X className="h-5 w-5" />
        </button>

        {/* Gallery — top on mobile, left on desktop */}
        <div className="relative flex aspect-video w-full shrink-0 items-center justify-center bg-slate-950 md:aspect-auto md:h-[600px] md:w-3/5">
          {media.length > 0 ? (
            <>
              {current?.media_type === "video" ? (
                <video
                  key={current.id}
                  src={current.media_url}
                  controls
                  autoPlay
                  playsInline
                  className="h-full w-full object-contain"
                />
              ) : (
                <img
                  src={current?.media_url}
                  alt={`${item.title} — ${currentIndex + 1}`}
                  className="h-full w-full object-contain animate-fade-in"
                />
              )}

              {/* Prev arrow */}
              {hasPrev && (
                <button
                  onClick={goPrev}
                  className="absolute left-3 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white backdrop-blur-md transition-all hover:scale-110 hover:bg-white/20 md:left-4 md:h-12 md:w-12"
                  aria-label={t("previous")}
                >
                  <ChevronLeft className="h-6 w-6" />
                </button>
              )}

              {/* Next arrow */}
              {hasNext && (
                <button
                  onClick={goNext}
                  className="absolute right-3 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white backdrop-blur-md transition-all hover:scale-110 hover:bg-white/20 md:right-4 md:h-12 md:w-12"
                  aria-label={t("next")}
                >
                  <ChevronRight className="h-6 w-6" />
                </button>
              )}

              {/* Counter */}
              {media.length > 1 && (
                <div className="absolute bottom-3 left-1/2 -translate-x-1/2 rounded-full bg-black/60 px-3 py-1.5 text-xs font-medium text-white backdrop-blur-md md:bottom-4">
                  {currentIndex + 1} / {media.length}
                </div>
              )}
            </>
          ) : (
            <div className="flex flex-col items-center text-white/40">
              <ImageIcon className="h-16 w-16" strokeWidth={1.5} />
              <p className="mt-2 text-sm">{t("noMedia")}</p>
            </div>
          )}
        </div>

        {/* Info panel — bottom on mobile, right on desktop */}
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden md:w-2/5">
          {/* Header */}
          <div className="shrink-0 border-b border-border px-5 py-4 pr-14 sm:px-6 sm:py-5">
            <h2 className="break-words text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              {item.title}
            </h2>
            {(imageCount > 0 || videoCount > 0) && (
              <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
                {imageCount > 0 && (
                  <span className="flex items-center gap-1.5">
                    <ImageIcon className="h-3.5 w-3.5" />
                    {imageCount} {imageCount > 1 ? t("photos") : t("photo")}
                  </span>
                )}
                {videoCount > 0 && (
                  <span className="flex items-center gap-1.5">
                    <Film className="h-3.5 w-3.5" />
                    {videoCount} {videoCount > 1 ? t("videos") : t("video")}
                  </span>
                )}
              </div>
            )}
          </div>

          {/* Body — scrollable */}
          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-6">
            {item.description ? (
              <p className="whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground/80">
                {item.description}
              </p>
            ) : (
              <p className="text-sm italic text-muted-foreground">{t("noDescription")}</p>
            )}

            {/* Thumbnails strip */}
            {media.length > 1 && (
              <div className="mt-6">
                <p className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  {t("gallery")}
                </p>
                <div className="grid grid-cols-4 gap-2">
                  {media.map((m, i) => (
                    <button
                      key={m.id}
                      onClick={() => setCurrentIndex(i)}
                      className={`relative aspect-square overflow-hidden rounded-lg border-2 transition-all ${
                        i === currentIndex
                          ? "border-rose-500 shadow-md"
                          : "border-transparent opacity-60 hover:opacity-100"
                      }`}
                    >
                      {m.media_type === "video" ? (
                        <div className="relative h-full w-full bg-slate-900">
                          {m.thumbnail_url ? (
                            <img
                              src={m.thumbnail_url}
                              alt={`Thumb ${i + 1}`}
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
                            <Film className="h-3.5 w-3.5 text-white drop-shadow-md" />
                          </div>
                        </div>
                      ) : (
                        <img
                          src={m.media_url}
                          alt={`Thumb ${i + 1}`}
                          className="h-full w-full object-cover"
                        />
                      )}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Footer link */}
          {item.link_url && (
            <div className="shrink-0 border-t border-border bg-muted/20 px-5 py-4 sm:px-6">
              <a
                href={item.link_url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-rose-500 to-rose-600 text-sm font-semibold text-white shadow-md transition-all hover:shadow-lg hover:shadow-rose-500/30 active:scale-[0.98]"
              >
                <ExternalLink className="h-4 w-4" />
                {t("viewProject")}
              </a>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
