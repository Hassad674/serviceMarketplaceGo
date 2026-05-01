"use client"

import { Pencil, Trash2, ImageIcon, Film, Play, ExternalLink } from "lucide-react"
import { useTranslations } from "next-intl"
import type { PortfolioItem } from "../api/portfolio-api"

interface PortfolioItemCardProps {
  item: PortfolioItem
  readOnly?: boolean
  onView?: () => void
  onEdit?: () => void
  onDelete?: () => void
}

export function PortfolioItemCard({
  item,
  readOnly = false,
  onView,
  onEdit,
  onDelete,
}: PortfolioItemCardProps) {
  const t = useTranslations("portfolio")
  const sortedMedia = [...item.media].sort((a, b) => a.position - b.position)
  const cover = sortedMedia[0]
  const imageCount = sortedMedia.filter((m) => m.media_type === "image").length
  const videoCount = sortedMedia.filter((m) => m.media_type === "video").length
  const totalMedia = sortedMedia.length
  const coverIsVideo = cover?.media_type === "video"

  let hostname = ""
  if (item.link_url) {
    try {
      hostname = new URL(item.link_url).hostname
    } catch {
      hostname = item.link_url
    }
  }

  return (
    <div
      className="group relative aspect-[4/5] cursor-pointer overflow-hidden rounded-2xl bg-slate-900 shadow-sm transition-all duration-300 ease-out hover:shadow-xl hover:-translate-y-1"
      onClick={onView}
    >
      {/* Cover — custom thumbnail (videos) > image > video first frame > placeholder.
          Plain <img> for both branches: portfolio media is a MinIO URL,
          see profile-header.tsx for the rationale. */}
      {coverIsVideo && cover?.thumbnail_url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={cover.thumbnail_url}
          alt={item.title}
          className="absolute inset-0 h-full w-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.04]"
        />
      ) : coverIsVideo && cover?.media_url ? (
        <video
          src={`${cover.media_url}#t=0.1`}
          preload="metadata"
          muted
          playsInline
          className="absolute inset-0 h-full w-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.04]"
        />
      ) : cover?.media_url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={cover.media_url}
          alt={item.title}
          className="absolute inset-0 h-full w-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.04]"
        />
      ) : (
        <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-slate-200 to-slate-300">
          <ImageIcon className="h-12 w-12 text-slate-400" strokeWidth={1.5} />
        </div>
      )}

      {/* Play icon overlay if cover is a video */}
      {coverIsVideo && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-black/50 backdrop-blur-sm transition-transform duration-300 group-hover:scale-110">
            <Play className="h-6 w-6 fill-white text-white" />
          </div>
        </div>
      )}

      {/* Media count badge */}
      {totalMedia > 1 && (
        <div className="absolute left-2.5 top-2.5 flex items-center gap-1.5 rounded-full bg-black/60 px-2.5 py-1 text-xs font-medium text-white backdrop-blur-sm">
          {imageCount > 0 && (
            <span className="flex items-center gap-1">
              <ImageIcon className="h-3 w-3" strokeWidth={2.5} />
              {imageCount}
            </span>
          )}
          {videoCount > 0 && (
            <span className="flex items-center gap-1">
              <Film className="h-3 w-3" strokeWidth={2.5} />
              {videoCount}
            </span>
          )}
        </div>
      )}

      {/* Edit/Delete actions (edit mode only) — always visible on mobile, hover on desktop */}
      {!readOnly && (
        <div className="absolute right-2 top-2 flex gap-1.5 opacity-100 transition-all duration-200 sm:right-2.5 sm:top-2.5 sm:translate-y-1 sm:opacity-0 sm:group-hover:translate-y-0 sm:group-hover:opacity-100">
          <button
            onClick={(e) => {
              e.stopPropagation()
              onEdit?.()
            }}
            className="flex h-8 w-8 items-center justify-center rounded-full bg-white/95 text-slate-700 shadow-md backdrop-blur-sm transition-all hover:scale-110 hover:bg-white hover:text-rose-600 sm:h-9 sm:w-9"
            aria-label={t("edit")}
          >
            <Pencil className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation()
              onDelete?.()
            }}
            className="flex h-8 w-8 items-center justify-center rounded-full bg-white/95 text-slate-700 shadow-md backdrop-blur-sm transition-all hover:scale-110 hover:bg-white hover:text-red-600 sm:h-9 sm:w-9"
            aria-label={t("delete")}
          >
            <Trash2 className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
          </button>
        </div>
      )}

      {/* Bottom gradient + title */}
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/95 via-black/60 to-transparent p-2.5 pt-10 sm:p-4 sm:pt-12">
        <h3 className="line-clamp-2 break-words text-sm font-semibold text-white sm:text-base">
          {item.title}
        </h3>
        {hostname && (
          <div className="mt-1 hidden items-center gap-1 text-xs text-white/80 opacity-0 transition-opacity duration-300 group-hover:opacity-100 sm:flex">
            <ExternalLink className="h-3 w-3 shrink-0" />
            <span className="truncate">{hostname}</span>
          </div>
        )}
      </div>
    </div>
  )
}
