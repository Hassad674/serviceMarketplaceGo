"use client"

import { useState } from "react"
import { X, Building2, FileText, Video } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { UploadModal } from "@/shared/components/upload-modal"
import type { JobFormData, DescriptionType } from "../types"
import { ApplicantTypeSelector } from "./applicant-type-selector"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"

// Job details — title + description (text/video/both) + skills + applicant
// type + about-org card. Soleil v2 chrome (ivoire bg, corail focus,
// rounded-full pills, mono labels). Form behaviour and the underlying
// `formData` shape are unchanged — both create- and edit-job-form
// consume this widget identically.

type JobDetailsSectionProps = {
  formData: JobFormData
  updateField: <K extends keyof JobFormData>(
    field: K,
    value: JobFormData[K],
  ) => void
  hideApplicantType?: boolean
}

const TITLE_MAX_LENGTH = 100
const MAX_TAGS = 5
const DESC_OPTIONS: DescriptionType[] = ["text", "video", "both"]

const VIDEO_MAX_SIZE = 100 * 1024 * 1024 // 100 MB

const FIELD_INPUT_CLASSES = cn(
  "h-12 w-full rounded-xl border border-border bg-background px-4 text-[14px]",
  "text-foreground placeholder:text-subtle-foreground",
  "transition-colors duration-150",
  "focus:border-primary focus:bg-card focus:outline-none focus:ring-4 focus:ring-primary/15",
)

export function JobDetailsSection({
  formData,
  updateField,
  hideApplicantType = false,
}: JobDetailsSectionProps) {
  const t = useTranslations("job")
  const tUpload = useTranslations("upload")
  const { data: user } = useUser()
  const [videoModalOpen, setVideoModalOpen] = useState(false)

  const showTextarea =
    formData.descriptionType === "text" || formData.descriptionType === "both"
  const showVideo =
    formData.descriptionType === "video" || formData.descriptionType === "both"

  const descLabelMap: Record<
    DescriptionType,
    { label: string; icon: React.ReactNode }
  > = {
    text: {
      label: t("descText"),
      icon: <FileText className="h-4 w-4" strokeWidth={1.7} />,
    },
    video: {
      label: t("descVideo"),
      icon: <Video className="h-4 w-4" strokeWidth={1.7} />,
    },
    both: {
      label: t("descBoth"),
      icon: (
        <span className="flex items-center gap-1">
          <FileText className="h-4 w-4" strokeWidth={1.7} />
          <Video className="h-4 w-4" strokeWidth={1.7} />
        </span>
      ),
    },
  }

  return (
    <div className="space-y-5">
      {/* Job title */}
      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <label className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {t("jobTitle")}
          </label>
          <span
            className={cn(
              "font-mono text-[11px] tabular-nums",
              formData.title.length > TITLE_MAX_LENGTH
                ? "text-primary-deep"
                : "text-subtle-foreground",
            )}
          >
            {formData.title.length}/{TITLE_MAX_LENGTH}
          </span>
        </div>
        <Input
          type="text"
          value={formData.title}
          onChange={(e) => updateField("title", e.target.value)}
          maxLength={TITLE_MAX_LENGTH}
          placeholder={t("jobTitlePlaceholder")}
          className={FIELD_INPUT_CLASSES}
        />
      </div>

      {/* Description type selector */}
      <div>
        <label className="mb-1.5 block font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
          {t("descriptionFormat")}
        </label>
        <div className="space-y-2">
          {DESC_OPTIONS.map((option) => (
            <label
              key={option}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-2xl border px-4 py-3",
                "transition-colors duration-150",
                formData.descriptionType === option
                  ? "border-primary bg-primary-soft/60"
                  : "border-border bg-card hover:border-border-strong",
              )}
            >
              <Input
                type="radio"
                name="descriptionType"
                value={option}
                checked={formData.descriptionType === option}
                onChange={() => updateField("descriptionType", option)}
                className="h-4 w-4 border-border text-primary focus:ring-primary/15"
              />
              <span className="flex items-center gap-2">
                {descLabelMap[option].icon}
                <span
                  className={cn(
                    "text-[13.5px] font-medium",
                    formData.descriptionType === option
                      ? "text-primary-deep"
                      : "text-foreground",
                  )}
                >
                  {descLabelMap[option].label}
                </span>
              </span>
            </label>
          ))}
        </div>
      </div>

      {/* Text description */}
      {showTextarea && (
        <div>
          <label className="mb-1.5 block font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {t("jobDescription")}
          </label>
          <textarea
            value={formData.description}
            onChange={(e) => updateField("description", e.target.value)}
            rows={5}
            className={cn(
              "w-full resize-none rounded-xl border border-border bg-background px-4 py-3 text-[14px]",
              "text-foreground placeholder:text-subtle-foreground",
              "transition-colors duration-150",
              "focus:border-primary focus:bg-card focus:outline-none focus:ring-4 focus:ring-primary/15",
            )}
          />
        </div>
      )}

      {/* Video upload zone */}
      {showVideo && (
        <div>
          <label className="mb-1.5 block font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
            {t("videoDescription")}
          </label>
          {formData.videoUrl ? (
            <div className="space-y-3">
              <div className="aspect-video max-h-[300px] overflow-hidden rounded-xl bg-foreground">
                <video
                  src={formData.videoUrl}
                  controls
                  className="h-full w-full object-contain"
                  aria-label={t("videoDescription")}
                >
                  <track kind="captions" />
                </video>
              </div>
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={() => {
                  updateField("videoUrl", "")
                  updateField("videoFile", null)
                }}
                className={cn(
                  "w-full rounded-full border border-border-strong px-4 py-2.5 text-[13px] font-medium",
                  "text-foreground transition-colors duration-150",
                  "hover:bg-primary-soft hover:text-primary-deep",
                )}
              >
                {t("removeVideo")}
              </Button>
            </div>
          ) : (
            <div
              onClick={() => setVideoModalOpen(true)}
              className={cn(
                "flex cursor-pointer flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed border-border-strong p-8",
                "transition-colors duration-150 hover:border-primary",
              )}
            >
              <Video
                className="h-8 w-8 text-subtle-foreground"
                strokeWidth={1.6}
              />
              <p className="text-[13.5px] text-muted-foreground">
                {t("videoUploadHint")}
              </p>
              <span
                className={cn(
                  "rounded-full bg-card px-5 py-2 text-[13px] font-medium text-foreground",
                  "border border-border-strong transition-colors duration-150",
                  "hover:bg-primary-soft hover:text-primary-deep",
                )}
              >
                {t("videoUploadButton")}
              </span>
              <p className="font-mono text-[11px] text-subtle-foreground">
                MP4, WebM, MOV — 100 MB max
              </p>
            </div>
          )}
          <UploadModal
            open={videoModalOpen}
            onClose={() => setVideoModalOpen(false)}
            onUpload={async (file) => {
              updateField("videoFile", file)
              updateField("videoUrl", URL.createObjectURL(file))
              setVideoModalOpen(false)
            }}
            accept="video/mp4,video/webm,video/quicktime"
            maxSize={VIDEO_MAX_SIZE}
            title={tUpload("addVideo")}
            description={tUpload("videoFormats")}
            uploading={false}
          />
        </div>
      )}

      {/* Skills */}
      <TagInput
        label={t("skills")}
        placeholder={t("skillsPlaceholder")}
        tags={formData.skills}
        max={MAX_TAGS}
        onChange={(tags) => updateField("skills", tags)}
      />

      {/* Applicant type (hidden for agencies) */}
      {!hideApplicantType && (
        <ApplicantTypeSelector
          value={formData.applicantType}
          onChange={(v) => updateField("applicantType", v)}
        />
      )}

      {/* About company card */}
      <div>
        <label className="mb-1.5 block font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
          {t("aboutCompany")}
        </label>
        <div className="rounded-2xl border border-border bg-card p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-soft text-[13px] font-semibold text-primary-deep">
              {(
                (user?.first_name?.charAt(0) ?? "") +
                (user?.last_name?.charAt(0) ?? "")
              ).toUpperCase() || "?"}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-[14px] font-semibold text-foreground">
                {user?.display_name ?? "---"}
              </p>
              <div className="flex items-center gap-1.5 text-[12px] text-muted-foreground">
                <Building2 className="h-3 w-3" strokeWidth={1.6} />
                <span className="capitalize">{user?.role ?? "---"}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

type TagInputProps = {
  label: string
  placeholder: string
  tags: string[]
  max: number
  onChange: (tags: string[]) => void
}

function TagInput({
  label,
  placeholder,
  tags,
  max,
  onChange,
}: TagInputProps) {
  const [inputValue, setInputValue] = useState("")
  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault()
      addTag()
    }
    if (e.key === "Backspace" && inputValue === "" && tags.length > 0)
      onChange(tags.slice(0, -1))
  }
  function addTag() {
    const trimmed = inputValue.trim()
    if (
      !trimmed ||
      tags.length >= max ||
      tags.some((t) => t.toLowerCase() === trimmed.toLowerCase())
    )
      return
    onChange([...tags, trimmed])
    setInputValue("")
  }
  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <label className="font-mono text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
          {label}
        </label>
        <span className="font-mono text-[11px] tabular-nums text-subtle-foreground">
          {tags.length}/{max}
        </span>
      </div>
      <div
        className={cn(
          "flex min-h-[48px] flex-wrap items-center gap-2 rounded-xl border border-border bg-background px-3 py-2.5",
          "transition-colors duration-150",
          "focus-within:border-primary focus-within:bg-card focus-within:ring-4 focus-within:ring-primary/15",
        )}
      >
        {tags.map((tag, index) => (
          <span
            key={tag}
            className={cn(
              "inline-flex items-center gap-1 rounded-full bg-primary-soft px-2.5 py-1",
              "text-[12.5px] font-semibold text-primary-deep",
            )}
          >
            {tag}
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => onChange(tags.filter((_, i) => i !== index))}
              className="rounded-full p-0.5 transition-colors hover:bg-primary/15"
              aria-label={`Remove ${tag}`}
            >
              <X className="h-3 w-3" strokeWidth={2.2} />
            </Button>
          </span>
        ))}
        {tags.length < max && (
          <Input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            onBlur={addTag}
            placeholder={tags.length === 0 ? placeholder : ""}
            className={cn(
              "min-w-[120px] flex-1 bg-transparent text-[13.5px]",
              "text-foreground placeholder:text-subtle-foreground",
              "focus:outline-none",
            )}
          />
        )}
      </div>
    </div>
  )
}
