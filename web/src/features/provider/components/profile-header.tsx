"use client"

import Image from "next/image"
import { useState, useRef } from "react"
import { Camera, Star, Edit2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { UploadModal } from "@/shared/components/upload-modal"
import type { Profile } from "../api/profile-api"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
type RoleContext = "agency" | "provider" | "referrer"

interface ProfileHeaderProps {
  profile: Profile | undefined
  displayName: string
  roleContext: RoleContext
  onUpdateTitle?: (title: string) => void
  onUploadPhoto?: (file: File) => Promise<void>
  uploadingPhoto?: boolean
  readOnly?: boolean
  averageRating?: number
  reviewCount?: number
}

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB

export function ProfileHeader({
  profile,
  displayName,
  roleContext,
  onUpdateTitle,
  onUploadPhoto,
  uploadingPhoto = false,
  readOnly = false,
  averageRating,
  reviewCount,
}: ProfileHeaderProps) {
  const [isEditingTitle, setIsEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState(profile?.title ?? "")
  const [photoModalOpen, setPhotoModalOpen] = useState(false)
  const [photoError, setPhotoError] = useState(false)
  const titleInputRef = useRef<HTMLInputElement>(null)
  const t = useTranslations("profile")
  const tUpload = useTranslations("upload")
  const tSidebar = useTranslations("sidebar")

  // Reset error state when photo URL changes (e.g. after upload). We
  // track the URL in render-time state so the reset happens during the
  // render that observes the change, not in an effect.
  const [lastPhotoUrl, setLastPhotoUrl] = useState(profile?.photo_url)
  if (lastPhotoUrl !== profile?.photo_url) {
    setLastPhotoUrl(profile?.photo_url)
    setPhotoError(false)
  }

  const imageLabel = roleContext === "agency" ? t("logo") : t("photo")
  const badgeText = roleContext === "referrer" ? tSidebar("businessReferrer") : null
  const isRounded = roleContext === "agency"

  function handleTitleClick() {
    setTitleDraft(profile?.title ?? "")
    setIsEditingTitle(true)
    setTimeout(() => titleInputRef.current?.focus(), 0)
  }

  function handleTitleSubmit() {
    setIsEditingTitle(false)
    const trimmed = titleDraft.trim()
    if (trimmed && trimmed !== profile?.title && onUpdateTitle) {
      onUpdateTitle(trimmed)
    }
  }

  function handleTitleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter") handleTitleSubmit()
    if (event.key === "Escape") setIsEditingTitle(false)
  }

  async function handlePhotoUpload(file: File) {
    if (!onUploadPhoto) return
    await onUploadPhoto(file)
    setPhotoModalOpen(false)
  }

  return (
    <>
      <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <div className="flex flex-col sm:flex-row items-start gap-5">
          {/* Photo / Logo — 96px */}
          <div className="relative shrink-0">
            {readOnly ? (
              <div
                className={cn(
                  "w-24 h-24 bg-muted flex items-center justify-center overflow-hidden",
                  "border border-border",
                  isRounded ? "rounded-lg" : "rounded-full",
                )}
              >
                {profile?.photo_url && !photoError ? (
                  <Image
                    src={profile.photo_url}
                    alt={t("imageAlt", { imageType: imageLabel, name: displayName })}
                    width={96}
                    height={96}
                    onError={() => setPhotoError(true)}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <Camera className="w-7 h-7 text-muted-foreground" aria-hidden="true" />
                )}
              </div>
            ) : (
              <Button variant="ghost" size="auto"
                type="button"
                onClick={() => setPhotoModalOpen(true)}
                className={cn(
                  "w-24 h-24 bg-muted flex items-center justify-center overflow-hidden",
                  "border-2 border-dashed border-border hover:border-primary transition-colors",
                  "focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2",
                  isRounded ? "rounded-lg" : "rounded-full",
                )}
                aria-label={t("editPhoto", { imageType: imageLabel.toLowerCase() })}
              >
                {profile?.photo_url && !photoError ? (
                  <Image
                    src={profile.photo_url}
                    alt={t("imageAlt", { imageType: imageLabel, name: displayName })}
                    width={96}
                    height={96}
                    onError={() => setPhotoError(true)}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <Camera className="w-7 h-7 text-muted-foreground" aria-hidden="true" />
                )}
              </Button>
            )}
            {!readOnly && (
              <span className="absolute -bottom-2 left-1/2 -translate-x-1/2 text-xs text-muted-foreground bg-card px-2">
                {imageLabel}
              </span>
            )}
          </div>

          {/* Name, title, stats */}
          <div className="flex-1 min-w-0 space-y-1.5">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl font-bold text-foreground truncate">
                {displayName}
              </h1>
              {badgeText && (
                <span className="rounded-full bg-accent text-accent-foreground px-2.5 py-0.5 text-xs font-medium">
                  {badgeText}
                </span>
              )}
            </div>

            {/* Title */}
            {readOnly ? (
              profile?.title && profile.title !== displayName ? (
                <p className="text-base text-muted-foreground">{profile.title}</p>
              ) : null
            ) : isEditingTitle ? (
              <Input
                ref={titleInputRef}
                type="text"
                value={titleDraft}
                onChange={(event) => setTitleDraft(event.target.value)}
                onBlur={handleTitleSubmit}
                onKeyDown={handleTitleKeyDown}
                placeholder={t("yourProfessionalTitle")}
                className="w-full max-w-md bg-muted border border-input rounded-md px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                aria-label={t("professionalTitle")}
              />
            ) : (
              <Button variant="ghost" size="auto"
                type="button"
                onClick={handleTitleClick}
                className="group flex items-center gap-2 text-base text-muted-foreground hover:text-foreground transition-colors"
                aria-label={t("editProfessionalTitle")}
              >
                <span className={cn(!profile?.title && "italic")}>
                  {profile?.title || t("addTitle")}
                </span>
                <Edit2 className="w-3.5 h-3.5 opacity-0 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
              </Button>
            )}

            <p className="text-sm text-muted-foreground">0 {t("completedProjects")}</p>
          </div>

          {/* Average rating — shows real stars when reviews exist */}
          {reviewCount !== undefined && reviewCount > 0 ? (
            <div className="flex items-center gap-1.5 text-sm shrink-0">
              <Star
                className="w-4 h-4 fill-amber-400 text-amber-400"
                strokeWidth={1.5}
                aria-hidden="true"
              />
              <span className="font-semibold text-foreground">
                {(averageRating ?? 0).toFixed(1)}
              </span>
              <span className="text-muted-foreground">({reviewCount})</span>
            </div>
          ) : (
            <div className="flex items-center gap-1.5 text-sm text-muted-foreground shrink-0">
              <Star className="w-4 h-4" aria-hidden="true" />
              <span>{t("noReviews")}</span>
            </div>
          )}
        </div>
      </section>

      {!readOnly && (
        <UploadModal
          open={photoModalOpen}
          onClose={() => setPhotoModalOpen(false)}
          onUpload={handlePhotoUpload}
          accept="image/*"
          maxSize={PHOTO_MAX_SIZE}
          title={tUpload("addPhoto")}
          description={tUpload("imageFormats", { imageType: imageLabel.toLowerCase() })}
          uploading={uploadingPhoto}
        />
      )}
    </>
  )
}
