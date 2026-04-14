"use client"

import { useState } from "react"
import { UploadModal } from "@/shared/components/upload-modal"
import {
  ProfileIdentityHeader,
  type ProfileIdentityHeaderProps,
} from "@/shared/components/ui/profile-identity-header"
import { useTranslations } from "next-intl"
import type { FreelanceProfile } from "../api/freelance-profile-api"

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB

interface FreelanceProfileHeaderProps {
  profile: FreelanceProfile
  displayName: string
  rating?: { average: number; count: number }
  editable?: {
    onSaveTitle?: (next: string) => void
    onUploadPhoto?: (file: File) => Promise<void>
    uploadingPhoto?: boolean
  }
}

// FreelanceProfileHeader wires the shared identity header to the
// freelance persona: pulls photo + title + availability straight from
// the profile aggregate and, when editable, opens the upload modal on
// photo-click. No business logic beyond the cache read — the
// mutations live in their dedicated hooks.
export function FreelanceProfileHeader(props: FreelanceProfileHeaderProps) {
  const { profile, displayName, rating, editable } = props
  const t = useTranslations("profile")
  const tUpload = useTranslations("upload")
  const [photoModalOpen, setPhotoModalOpen] = useState(false)

  const identity: ProfileIdentityHeaderProps["identity"] = {
    photoUrl: profile.photo_url,
    displayName,
    title: profile.title,
    availabilityStatus: profile.availability_status,
  }

  const editableProps: ProfileIdentityHeaderProps["editable"] = editable
    ? {
        onEditPhoto: editable.onUploadPhoto
          ? () => setPhotoModalOpen(true)
          : undefined,
        onEditTitle: editable.onSaveTitle,
        photoAlt: t("imageAlt", {
          imageType: t("photo"),
          name: displayName,
        }),
      }
    : undefined

  async function handlePhotoUpload(file: File) {
    if (!editable?.onUploadPhoto) return
    await editable.onUploadPhoto(file)
    setPhotoModalOpen(false)
  }

  return (
    <>
      <ProfileIdentityHeader
        identity={identity}
        rating={rating}
        editable={editableProps}
      />

      {editable?.onUploadPhoto ? (
        <UploadModal
          open={photoModalOpen}
          onClose={() => setPhotoModalOpen(false)}
          onUpload={handlePhotoUpload}
          accept="image/*"
          maxSize={PHOTO_MAX_SIZE}
          title={tUpload("addPhoto")}
          description={tUpload("imageFormats", {
            imageType: t("photo").toLowerCase(),
          })}
          uploading={editable.uploadingPhoto ?? false}
        />
      ) : null}
    </>
  )
}
