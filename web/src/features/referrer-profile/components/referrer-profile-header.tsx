"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"
import {
  ProfileIdentityHeader,
  type ProfileIdentityHeaderProps,
} from "@/shared/components/ui/profile-identity-header"
import type { ReferrerProfile } from "../api/referrer-profile-api"

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB

interface ReferrerProfileHeaderProps {
  profile: ReferrerProfile
  displayName: string
  rating?: { average: number; count: number }
  editable?: {
    onSaveTitle?: (next: string) => void
    onUploadPhoto?: (file: File) => Promise<void>
    uploadingPhoto?: boolean
  }
}

// ReferrerProfileHeader wires the shared identity header with the
// referrer-specific badge ("Apporteur d'affaire") so the public
// viewer immediately understands which persona they are looking at.
export function ReferrerProfileHeader(props: ReferrerProfileHeaderProps) {
  const { profile, displayName, rating, editable } = props
  const t = useTranslations("profile")
  const tUpload = useTranslations("upload")
  const tSidebar = useTranslations("sidebar")
  const [photoModalOpen, setPhotoModalOpen] = useState(false)

  const identity: ProfileIdentityHeaderProps["identity"] = {
    photoUrl: profile.photo_url,
    displayName,
    title: profile.title,
    availabilityStatus: profile.availability_status,
  }

  const badge: ProfileIdentityHeaderProps["badge"] = {
    label: tSidebar("businessReferrer"),
  }

  const editableProps: ProfileIdentityHeaderProps["editable"] = editable
    ? {
        onEditPhoto: editable.onUploadPhoto
          ? () => setPhotoModalOpen(true)
          : undefined,
        onEditTitle: editable.onSaveTitle,
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
        badge={badge}
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
