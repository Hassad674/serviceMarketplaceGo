"use client"

import Image from "next/image"
import { useState } from "react"
import { Camera, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"
import { useOrganizationShared } from "../hooks/use-organization-shared"
import { useUploadOrganizationPhoto } from "../hooks/use-update-organization-photo"

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB

// SharedPhotoUpload renders the dedicated "Change photo" action row
// used on /profile. It reads the current photo from the org-shared
// cache so the preview stays in sync with the identity header and
// dispatches the upload mutation which invalidates every persona
// cache on success.
export function SharedPhotoUpload() {
  const t = useTranslations("profile")
  const tUpload = useTranslations("upload")
  const [modalOpen, setModalOpen] = useState(false)
  const { data: shared } = useOrganizationShared()
  const upload = useUploadOrganizationPhoto()

  const photoUrl = shared?.photo_url ?? ""

  async function handleUpload(file: File) {
    await upload.mutateAsync(file)
    setModalOpen(false)
  }

  return (
    <>
      <section
        aria-labelledby="shared-photo-section-title"
        className="bg-card border border-border rounded-xl p-6 shadow-sm"
      >
        <header className="mb-4 flex flex-col gap-1">
          <h2
            id="shared-photo-section-title"
            className="text-lg font-semibold text-foreground"
          >
            {t("photo")}
          </h2>
        </header>

        <div className="flex items-center gap-4">
          <PhotoPreview photoUrl={photoUrl} />
          <button
            type="button"
            onClick={() => setModalOpen(true)}
            disabled={upload.isPending}
            className="inline-flex items-center gap-2 rounded-md border border-border h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
          >
            {upload.isPending ? (
              <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
            ) : (
              <Camera className="w-4 h-4" aria-hidden="true" />
            )}
            {tUpload("addPhoto")}
          </button>
        </div>
      </section>

      <UploadModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onUpload={handleUpload}
        accept="image/*"
        maxSize={PHOTO_MAX_SIZE}
        title={tUpload("addPhoto")}
        description={tUpload("imageFormats", {
          imageType: t("photo").toLowerCase(),
        })}
        uploading={upload.isPending}
      />
    </>
  )
}

interface PhotoPreviewProps {
  photoUrl: string
}

function PhotoPreview({ photoUrl }: PhotoPreviewProps) {
  if (!photoUrl) {
    return (
      <div className="w-20 h-20 rounded-full bg-muted flex items-center justify-center border border-border">
        <Camera className="w-6 h-6 text-muted-foreground" aria-hidden="true" />
      </div>
    )
  }
  return (
    <div className="w-20 h-20 rounded-full bg-muted overflow-hidden border border-border">
      {/* 80×80 shared-photo preview. Hosts (MinIO + R2) live in next.config.ts. */}
      <Image
        src={photoUrl}
        alt=""
        width={80}
        height={80}
        className="w-full h-full object-cover"
      />
    </div>
  )
}
