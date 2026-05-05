"use client"

import { useState } from "react"
import Image from "next/image"
import { BadgeCheck, Camera, Globe, MapPin, Star } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { UploadModal } from "@/shared/components/upload-modal"
import { AvailabilityPill } from "@/shared/components/ui/availability-pill"
import { Portrait } from "@/shared/components/ui/portrait"
import { cn } from "@/shared/lib/utils"
import {
  formatPricing,
  type PricingLocale,
} from "@/shared/lib/profile/pricing-format"
import type { FreelanceProfile } from "../api/freelance-profile-api"

const PHOTO_MAX_SIZE = 5 * 1024 * 1024 // 5 MB
const PHOTO_SIZE_PX = 130

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

// FreelanceProfileHeader is the Soleil v2 hero of W-16: a warm cover
// band overlapped by a white card containing the portrait, identity
// (name + italic title + meta), availability pill, and the "À partir
// de" pricing block on the right rail. The component owns the photo
// upload modal and the inline title editor so the page composing it
// stays declarative; all data still flows from the upstream feature
// hooks via the parent.
export function FreelanceProfileHeader(props: FreelanceProfileHeaderProps) {
  const { profile, displayName, rating, editable } = props
  const t = useTranslations("profile")
  const tFreelance = useTranslations("profile.freelance")
  const tUpload = useTranslations("upload")
  const locale = (useLocale() === "fr" ? "fr" : "en") satisfies PricingLocale
  const [photoModalOpen, setPhotoModalOpen] = useState(false)

  const isEditable = Boolean(editable?.onUploadPhoto)
  const portraitId = portraitSeed(profile.organization_id)
  const meta = buildMeta(profile)

  async function handlePhotoUpload(file: File) {
    if (!editable?.onUploadPhoto) return
    await editable.onUploadPhoto(file)
    setPhotoModalOpen(false)
  }

  return (
    <>
      <section
        aria-labelledby="freelance-profile-header-title"
        className="relative isolate"
      >
        <CoverBand />

        <div className="relative -mt-16 mx-4 sm:mx-6 rounded-2xl border border-border bg-card px-6 py-7 shadow-[0_4px_24px_rgba(42,31,21,0.04)] sm:px-8 sm:py-8">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8">
            <PortraitFrame
              photoUrl={profile.photo_url}
              photoAlt={t("imageAlt", {
                imageType: t("photo"),
                name: displayName,
              })}
              portraitId={portraitId}
              editable={isEditable}
              onEdit={
                editable?.onUploadPhoto
                  ? () => setPhotoModalOpen(true)
                  : undefined
              }
              editLabel={t("editPhoto", {
                imageType: t("photo").toLowerCase(),
              })}
            />

            <div className="min-w-0 flex-1 space-y-3">
              <NameRow
                displayName={displayName}
                availability={profile.availability_status}
              />
              <TitleRow
                title={profile.title}
                onSaveTitle={editable?.onSaveTitle}
              />
              <MetaRow
                meta={meta}
                rating={rating}
                noReviewsLabel={t("noReviews")}
              />
            </div>

            <PricingRail
              pricing={profile.pricing}
              locale={locale}
              fromLabel={tFreelance("priceFromLabel")}
            />
          </div>
        </div>
      </section>

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

// ---------- Subcomponents ----------

function CoverBand() {
  return (
    <div
      aria-hidden="true"
      className="gradient-warm relative h-40 rounded-2xl"
      style={{
        backgroundImage:
          "radial-gradient(60% 80% at 18% 30%, rgba(232,93,74,0.28), transparent 60%), radial-gradient(50% 70% at 82% 70%, rgba(240,138,168,0.35), transparent 60%), linear-gradient(135deg, var(--primary-soft), var(--pink-soft), var(--amber-soft))",
      }}
    />
  )
}

interface PortraitFrameProps {
  photoUrl: string
  photoAlt: string
  portraitId: number
  editable: boolean
  onEdit?: () => void
  editLabel: string
}

function PortraitFrame({
  photoUrl,
  photoAlt,
  portraitId,
  editable,
  onEdit,
  editLabel,
}: PortraitFrameProps) {
  const frameClass =
    "relative shrink-0 rounded-2xl bg-card p-1 shadow-[0_2px_12px_rgba(42,31,21,0.06)]"

  const inner = photoUrl ? (
    <Image
      src={photoUrl}
      alt={photoAlt}
      width={PHOTO_SIZE_PX}
      height={PHOTO_SIZE_PX}
      className="h-[130px] w-[130px] rounded-xl object-cover"
    />
  ) : (
    <Portrait
      id={portraitId}
      size={PHOTO_SIZE_PX}
      rounded="xl"
      alt={photoAlt}
    />
  )

  if (editable && onEdit) {
    return (
      <div className={frameClass}>
        <button
          type="button"
          onClick={onEdit}
          aria-label={editLabel}
          className="block rounded-xl outline-none transition-opacity hover:opacity-90 focus-visible:ring-4 focus-visible:ring-primary/20"
        >
          {inner}
        </button>
        <span className="absolute bottom-1 right-1 inline-flex h-7 w-7 items-center justify-center rounded-full bg-foreground text-background shadow-[0_2px_6px_rgba(0,0,0,0.15)]">
          <Camera className="h-3.5 w-3.5" aria-hidden="true" />
        </span>
      </div>
    )
  }

  return (
    <div className={frameClass}>
      {inner}
      <span
        aria-hidden="true"
        className="absolute -bottom-1 -right-1 inline-flex h-7 w-7 items-center justify-center rounded-full bg-card text-success shadow-[0_2px_6px_rgba(0,0,0,0.15)]"
      >
        <BadgeCheck className="h-4 w-4" aria-hidden="true" />
      </span>
    </div>
  )
}

interface NameRowProps {
  displayName: string
  availability: FreelanceProfile["availability_status"]
}

function NameRow({ displayName, availability }: NameRowProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <h1
        id="freelance-profile-header-title"
        className="font-serif text-3xl font-medium tracking-[-0.025em] text-foreground sm:text-[38px] sm:leading-tight"
      >
        {displayName}
      </h1>
      {availability ? <AvailabilityPill status={availability} /> : null}
    </div>
  )
}

interface TitleRowProps {
  title: string
  onSaveTitle?: (next: string) => void
}

function TitleRow({ title, onSaveTitle }: TitleRowProps) {
  const t = useTranslations("profile")
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(title)
  const editable = Boolean(onSaveTitle)

  if (!editable) {
    return title ? (
      <p className="font-serif text-base italic text-muted-foreground sm:text-[17px]">
        {title}
      </p>
    ) : null
  }

  if (editing) {
    return (
      <input
        type="text"
        autoFocus
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => commitTitle(draft, title, setEditing, onSaveTitle)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            commitTitle(draft, title, setEditing, onSaveTitle)
          }
          if (e.key === "Escape") {
            setEditing(false)
            setDraft(title)
          }
        }}
        placeholder={t("yourProfessionalTitle")}
        aria-label={t("professionalTitle")}
        className="w-full max-w-md rounded-md border border-border-strong bg-background px-3 py-1.5 font-serif text-base italic text-foreground outline-none focus:border-primary focus:ring-4 focus:ring-primary/10"
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
      className="text-left font-serif text-base italic text-muted-foreground transition-colors hover:text-foreground sm:text-[17px]"
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

interface MetaItem {
  key: string
  icon: typeof MapPin
  label: string
}

interface MetaRowProps {
  meta: MetaItem[]
  rating?: { average: number; count: number }
  noReviewsLabel: string
}

function MetaRow({ meta, rating, noReviewsLabel }: MetaRowProps) {
  const tFreelance = useTranslations("profile.freelance")
  const hasRating = Boolean(rating && rating.count > 0)

  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-2 pt-1 text-[13px] text-muted-foreground">
      {meta.map((item) => (
        <span
          key={item.key}
          className="inline-flex items-center gap-1.5"
        >
          <item.icon
            className="h-3.5 w-3.5 text-subtle-foreground"
            aria-hidden="true"
          />
          <span>{item.label}</span>
        </span>
      ))}
      <span className="inline-flex items-center gap-1.5">
        <Star
          className={cn(
            "h-3.5 w-3.5",
            hasRating
              ? "fill-primary text-primary"
              : "text-subtle-foreground",
          )}
          strokeWidth={1.5}
          aria-hidden="true"
        />
        {hasRating && rating ? (
          <span>
            <strong className="font-semibold text-foreground">
              {rating.average.toFixed(1)}
            </strong>
            <span className="text-muted-foreground">
              {" "}
              · {tFreelance("reviewsCount", { count: rating.count })}
            </span>
          </span>
        ) : (
          <span>{noReviewsLabel}</span>
        )}
      </span>
    </div>
  )
}

interface PricingRailProps {
  pricing: FreelanceProfile["pricing"]
  locale: PricingLocale
  fromLabel: string
}

function PricingRail({ pricing, locale, fromLabel }: PricingRailProps) {
  if (!pricing) return null
  const formatted = formatPricing(pricing, locale)
  return (
    <div className="shrink-0 self-start text-left lg:text-right">
      <div className="mb-1 font-mono text-[10.5px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
        {fromLabel}
      </div>
      <div className="font-serif text-3xl font-medium leading-none tracking-[-0.025em] text-foreground sm:text-[32px]">
        {formatted}
      </div>
    </div>
  )
}

// ---------- Helpers ----------

function buildMeta(profile: FreelanceProfile): MetaItem[] {
  const items: MetaItem[] = []
  if (profile.city) {
    items.push({
      key: "location",
      icon: MapPin,
      label: profile.country_code
        ? `${profile.city} · ${profile.country_code}`
        : profile.city,
    })
  }
  const langs = profile.languages_professional ?? []
  if (langs.length > 0) {
    items.push({
      key: "languages",
      icon: Globe,
      label: langs.map(formatLanguageCode).join(" · "),
    })
  }
  return items
}

function formatLanguageCode(code: string): string {
  // Keep BCP-47 short codes capitalized — display matches the Soleil
  // header where "Français · Anglais" is shown via ISO labels. We
  // intentionally keep this simple: the language picker upstream already
  // stores ISO codes, and the listing card uses LanguagesStrip which
  // renders the short code. Centralizing into a single shared formatter
  // is a follow-up for the i18n-language-labels chantier.
  return code.toUpperCase()
}

function portraitSeed(orgId: string): number {
  let hash = 0
  for (let i = 0; i < orgId.length; i += 1) {
    hash = (hash * 31 + orgId.charCodeAt(i)) & 0xffffffff
  }
  return Math.abs(hash)
}
