"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"
import {
  ProfileIdentityHeader,
  type ProfileIdentityHeaderProps,
} from "@/shared/components/ui/profile-identity-header"
import type { Profile } from "../api/profile-api"

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB

interface AgencyProfileHeaderProps {
  profile: Profile
  displayName: string
  rating?: { average: number; count: number }
  editable?: {
    onSaveTitle?: (next: string) => void
    onUploadPhoto?: (file: File) => Promise<void>
    uploadingPhoto?: boolean
  }
}

// AgencyProfileHeader wraps the shared ProfileIdentityHeader with the
// legacy agency-scoped upload modal. Mirrors FreelanceProfileHeader
// one-for-one so the editable agency page and the editable freelance
// page share the same header shape, spacing and photo-upload flow.
// The legacy provider hooks (updateProfile + uploadPhoto) are passed
// in via the `editable` bag — this component stays presentational.
export function AgencyProfileHeader(props: AgencyProfileHeaderProps) {
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
          imageType: t("logo"),
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
            imageType: t("logo").toLowerCase(),
          })}
          uploading={editable.uploadingPhoto ?? false}
        />
      ) : null}
    </>
  )
}
