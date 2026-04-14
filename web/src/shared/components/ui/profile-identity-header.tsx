"use client"

import { Camera, Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { cn } from "@/shared/lib/utils"
import {
  AvailabilityPill,
  type AvailabilityStatus,
} from "./availability-pill"

// ProfileIdentityHeaderProps groups the identity header's inputs so
// the component stays under the 4-prop cap. Optional handlers (edit
// title, upload photo) opt into editable behaviour when present.
export interface ProfileIdentityHeaderProps {
  identity: {
    photoUrl: string
    displayName: string
    title: string
    availabilityStatus?: AvailabilityStatus
  }
  rating?: {
    average: number
    count: number
  }
  badge?: {
    label: string
  }
  editable?: {
    onEditPhoto?: () => void
    onEditTitle?: (next: string) => void
    photoAlt?: string
  }
}

// ProfileIdentityHeader renders the classic 96px-photo + name + title
// + availability + rating banner used on both /profile and the public
// freelance/referrer pages. Pure presentational shell: photo upload
// and title editing are delegated to the parent via callbacks, so the
// same component serves editable + read-only contexts without
// branching on feature-specific hooks.
export function ProfileIdentityHeader(props: ProfileIdentityHeaderProps) {
  const { identity, rating, badge, editable } = props
  const t = useTranslations("profile")
  const [photoError, setPhotoError] = useState(false)

  const photoAlt =
    editable?.photoAlt ??
    t("imageAlt", { imageType: t("photo"), name: identity.displayName })

  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
        <PhotoBlock
          photoUrl={identity.photoUrl}
          photoAlt={photoAlt}
          editable={Boolean(editable?.onEditPhoto)}
          onEdit={editable?.onEditPhoto}
          onError={() => setPhotoError(true)}
          errored={photoError}
        />

        <div className="min-w-0 flex-1 space-y-1.5">
          <HeaderTitleRow
            displayName={identity.displayName}
            badgeLabel={badge?.label}
          />

          <SubtitleRow
            title={identity.title}
            onEditTitle={editable?.onEditTitle}
          />

          {identity.availabilityStatus ? (
            <div className="pt-1">
              <AvailabilityPill status={identity.availabilityStatus} />
            </div>
          ) : null}
        </div>

        <RatingBlock rating={rating} />
      </div>
    </section>
  )
}

interface PhotoBlockProps {
  photoUrl: string
  photoAlt: string
  editable: boolean
  errored: boolean
  onError: () => void
  onEdit?: () => void
}

function PhotoBlock({
  photoUrl,
  photoAlt,
  editable,
  errored,
  onError,
  onEdit,
}: PhotoBlockProps) {
  const t = useTranslations("profile")
  const classes = cn(
    "w-24 h-24 bg-muted flex items-center justify-center overflow-hidden rounded-full",
    editable
      ? "border-2 border-dashed border-border hover:border-primary transition-colors focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
      : "border border-border",
  )

  const inner =
    photoUrl && !errored ? (
      <img
        src={photoUrl}
        alt={photoAlt}
        width={96}
        height={96}
        onError={onError}
        className="w-full h-full object-cover"
      />
    ) : (
      <Camera className="w-7 h-7 text-muted-foreground" aria-hidden="true" />
    )

  if (!editable) {
    return (
      <div className="relative shrink-0">
        <div className={classes}>{inner}</div>
      </div>
    )
  }
  return (
    <div className="relative shrink-0">
      <button
        type="button"
        onClick={onEdit}
        aria-label={t("editPhoto", { imageType: t("photo").toLowerCase() })}
        className={classes}
      >
        {inner}
      </button>
    </div>
  )
}

interface HeaderTitleRowProps {
  displayName: string
  badgeLabel: string | undefined
}

function HeaderTitleRow({ displayName, badgeLabel }: HeaderTitleRowProps) {
  return (
    <div className="flex items-center gap-3 flex-wrap">
      <h1 className="text-2xl font-bold text-foreground truncate">
        {displayName}
      </h1>
      {badgeLabel ? (
        <span className="rounded-full bg-accent text-accent-foreground px-2.5 py-0.5 text-xs font-medium">
          {badgeLabel}
        </span>
      ) : null}
    </div>
  )
}

interface SubtitleRowProps {
  title: string
  onEditTitle?: (next: string) => void
}

function SubtitleRow({ title, onEditTitle }: SubtitleRowProps) {
  const t = useTranslations("profile")
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(title)

  const editable = Boolean(onEditTitle)
  if (!editable) {
    return title ? (
      <p className="text-base text-muted-foreground">{title}</p>
    ) : null
  }

  if (editing) {
    return (
      <input
        type="text"
        autoFocus
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => commitTitle(draft, title, setEditing, onEditTitle)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            commitTitle(draft, title, setEditing, onEditTitle)
          }
          if (e.key === "Escape") {
            setEditing(false)
            setDraft(title)
          }
        }}
        placeholder={t("yourProfessionalTitle")}
        aria-label={t("professionalTitle")}
        className="w-full max-w-md bg-muted border border-input rounded-md px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
      />
    )
  }
  return (
    <button
      type="button"
      onClick={() => {
        setDraft(title)
        setEditing(true)
      }}
      className="text-left text-base text-muted-foreground hover:text-foreground transition-colors"
      aria-label={t("editProfessionalTitle")}
    >
      <span className={cn(!title && "italic")}>
        {title || t("addTitle")}
      </span>
    </button>
  )
}

function commitTitle(
  draft: string,
  current: string,
  setEditing: (next: boolean) => void,
  onSave: ((next: string) => void) | undefined,
) {
  setEditing(false)
  const trimmed = draft.trim()
  if (!onSave) return
  if (trimmed && trimmed !== current) {
    onSave(trimmed)
  }
}

interface RatingBlockProps {
  rating: { average: number; count: number } | undefined
}

function RatingBlock({ rating }: RatingBlockProps) {
  const t = useTranslations("profile")
  if (!rating || rating.count === 0) {
    return (
      <div className="flex items-center gap-1.5 text-sm text-muted-foreground shrink-0">
        <Star className="w-4 h-4" aria-hidden="true" />
        <span>{t("noReviews")}</span>
      </div>
    )
  }
  return (
    <div className="flex items-center gap-1.5 text-sm shrink-0">
      <Star
        className="w-4 h-4 fill-amber-400 text-amber-400"
        strokeWidth={1.5}
        aria-hidden="true"
      />
      <span className="font-semibold text-foreground">
        {rating.average.toFixed(1)}
      </span>
      <span className="text-muted-foreground">({rating.count})</span>
    </div>
  )
}
