"use client"

import { useEffect, useState } from "react"
import Image from "next/image"
import { Briefcase, Camera, Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"
import { cn, formatCurrency } from "@/shared/lib/utils"

export interface ClientProfileHeaderStats {
  totalSpent: number
  reviewCount: number
  averageRating: number
  projectsCompleted: number
}

export interface ClientProfileHeaderEditable {
  onUploadPhoto: (file: File) => Promise<void>
  uploadingPhoto: boolean
}

interface ClientProfileHeaderProps {
  companyName: string
  avatarUrl: string | null
  stats: ClientProfileHeaderStats
  // `editable` is optional — when absent the header renders in its
  // original read-only shape (public /clients/[id] page and non-admin
  // private viewers). When provided the avatar turns into an editable
  // widget that reuses the provider feature's upload mutation, so the
  // logo stays in sync across the two facets for agencies.
  editable?: ClientProfileHeaderEditable
}

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB — matches provider ProfileHeader

// ClientProfileHeader renders the identity + metrics strip for both
// the private `/client-profile` page and the public `/clients/[id]`
// page. When `editable` is passed it also mounts the shared
// UploadModal so owners with the `org_client_profile.edit` permission
// can swap the avatar without hopping to the provider profile.
export function ClientProfileHeader(props: ClientProfileHeaderProps) {
  const { companyName, avatarUrl, stats, editable } = props
  const t = useTranslations("clientProfile")

  const initials = getInitials(companyName)
  // Amount comes in as integer cents — formatCurrency expects whole
  // euros, so divide by 100. Keeping the cents/unit conversion here
  // (rather than in the hook) mirrors the pattern used by the
  // billing FeePreview component.
  const totalSpent = formatCurrency(stats.totalSpent / 100)

  return (
    <section
      className="bg-card border border-border rounded-2xl p-6 shadow-sm"
      aria-label={t("pageTitle")}
    >
      <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
        <Avatar
          avatarUrl={avatarUrl}
          initials={initials}
          name={companyName}
          editable={editable}
        />
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-semibold text-foreground truncate">
            {companyName}
          </h1>
          <p className="mt-1 inline-flex items-center gap-2 rounded-md bg-purple-50 px-2 py-1 text-xs font-medium text-purple-700 dark:bg-purple-500/15 dark:text-purple-300">
            <Briefcase className="h-3.5 w-3.5" aria-hidden="true" />
            {t("roleLabel")}
          </p>
        </div>
      </div>

      <dl className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label={t("totalSpentLabel")} value={totalSpent} />
        <Stat
          label={t("projectsCompleted")}
          value={String(stats.projectsCompleted)}
        />
        <Stat
          label={t("reviewsReceived")}
          value={String(stats.reviewCount)}
        />
        <Stat
          label={t("averageRating")}
          value={
            stats.reviewCount === 0
              ? "—"
              : `${stats.averageRating.toFixed(1)} / 5`
          }
          icon={stats.reviewCount > 0 ? <Star className="h-4 w-4 text-amber-500" aria-hidden="true" /> : null}
        />
      </dl>
    </section>
  )
}

function getInitials(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return "?"
  const parts = trimmed.split(/\s+/).slice(0, 2)
  return parts.map((p) => p.charAt(0).toUpperCase()).join("")
}

interface AvatarProps {
  avatarUrl: string | null
  initials: string
  name: string
  editable?: ClientProfileHeaderEditable
}

function Avatar({ avatarUrl, initials, name, editable }: AvatarProps) {
  const t = useTranslations("clientProfile")
  const tUpload = useTranslations("upload")
  const [open, setOpen] = useState(false)
  const [photoError, setPhotoError] = useState(false)

  // Reset the broken-image flag when the URL changes — typical on a
  // fresh upload returning a new signed URL.
  useEffect(() => {
    setPhotoError(false)
  }, [avatarUrl])

  async function handleUpload(file: File) {
    if (!editable) return
    await editable.onUploadPhoto(file)
    setOpen(false)
  }

  const hasImage = Boolean(avatarUrl) && !photoError

  const pictureNode = hasImage ? (
    <Image
      src={avatarUrl!}
      alt={name}
      width={80}
      height={80}
      onError={() => setPhotoError(true)}
      className="h-full w-full rounded-full object-cover"
    />
  ) : editable ? (
    <Camera
      className="h-6 w-6 text-muted-foreground"
      aria-hidden="true"
    />
  ) : (
    <span
      aria-hidden="true"
      className="flex h-full w-full items-center justify-center rounded-full bg-gradient-to-br from-purple-500 to-indigo-600 text-xl font-semibold text-white"
    >
      {initials}
    </span>
  )

  if (!editable) {
    return (
      <div className="h-20 w-20 shrink-0 overflow-hidden rounded-full">
        {pictureNode}
      </div>
    )
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={t("editLogo")}
        className={cn(
          "relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full",
          "border-2 border-dashed border-border bg-muted transition-colors",
          "hover:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2",
        )}
      >
        {pictureNode}
      </button>
      <UploadModal
        open={open}
        onClose={() => setOpen(false)}
        onUpload={handleUpload}
        accept="image/*"
        maxSize={PHOTO_MAX_SIZE}
        title={tUpload("addPhoto")}
        description={tUpload("imageFormats", { imageType: t("logo") })}
        uploading={editable.uploadingPhoto}
      />
    </>
  )
}

interface StatProps {
  label: string
  value: string
  icon?: React.ReactNode
}

function Stat({ label, value, icon }: StatProps) {
  return (
    <div className="rounded-xl bg-muted/40 px-4 py-3">
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd className="mt-1 flex items-center gap-1.5 text-lg font-semibold text-foreground">
        {icon}
        <span className="font-mono tabular-nums">{value}</span>
      </dd>
    </div>
  )
}
