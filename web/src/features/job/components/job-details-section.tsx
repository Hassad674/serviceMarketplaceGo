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
type JobDetailsSectionProps = {
  formData: JobFormData
  updateField: <K extends keyof JobFormData>(field: K, value: JobFormData[K]) => void
  hideApplicantType?: boolean
}

const TITLE_MAX_LENGTH = 100
const MAX_TAGS = 5
const DESC_OPTIONS: DescriptionType[] = ["text", "video", "both"]

const VIDEO_MAX_SIZE = 100 * 1024 * 1024 // 100 MB

export function JobDetailsSection({ formData, updateField, hideApplicantType = false }: JobDetailsSectionProps) {
  const t = useTranslations("job")
  const tUpload = useTranslations("upload")
  const { data: user } = useUser()
  const [videoModalOpen, setVideoModalOpen] = useState(false)

  const showTextarea = formData.descriptionType === "text" || formData.descriptionType === "both"
  const showVideo = formData.descriptionType === "video" || formData.descriptionType === "both"

  const descLabelMap: Record<DescriptionType, { label: string; icon: React.ReactNode }> = {
    text: { label: t("descText"), icon: <FileText className="h-4 w-4" /> },
    video: { label: t("descVideo"), icon: <Video className="h-4 w-4" /> },
    both: { label: t("descBoth"), icon: <span className="flex items-center gap-1"><FileText className="h-4 w-4" /><Video className="h-4 w-4" /></span> },
  }

  return (
    <div className="space-y-5">
      {/* Job title */}
      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("jobTitle")}</label>
          <span className={cn("text-xs tabular-nums", formData.title.length > TITLE_MAX_LENGTH ? "text-red-500" : "text-gray-400 dark:text-gray-500")}>
            {formData.title.length}/{TITLE_MAX_LENGTH}
          </span>
        </div>
        <Input
          type="text"
          value={formData.title}
          onChange={(e) => updateField("title", e.target.value)}
          maxLength={TITLE_MAX_LENGTH}
          placeholder={t("jobTitlePlaceholder")}
          className={cn("h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700", "bg-gray-50 dark:bg-gray-800 px-4 text-sm", "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500", "transition-all duration-200", "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10")}
        />
      </div>

      {/* Description type selector */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">{t("descriptionFormat")}</label>
        <div className="space-y-2">
          {DESC_OPTIONS.map((option) => (
            <label key={option} className={cn("flex cursor-pointer items-center gap-3 rounded-xl border px-4 py-3", "transition-all duration-200", formData.descriptionType === option ? "border-rose-500 bg-rose-50 dark:bg-rose-500/10 dark:border-rose-400" : "border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-600")}>
              <Input type="radio" name="descriptionType" value={option} checked={formData.descriptionType === option} onChange={() => updateField("descriptionType", option)} className="h-4 w-4 border-gray-300 dark:border-gray-600 text-rose-500 focus:ring-rose-500/20" />
              <span className="flex items-center gap-2">
                {descLabelMap[option].icon}
                <span className={cn("text-sm font-medium", formData.descriptionType === option ? "text-rose-700 dark:text-rose-300" : "text-gray-700 dark:text-gray-300")}>{descLabelMap[option].label}</span>
              </span>
            </label>
          ))}
        </div>
      </div>

      {/* Text description */}
      {showTextarea && (
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">{t("jobDescription")}</label>
          <textarea value={formData.description} onChange={(e) => updateField("description", e.target.value)} rows={5} className={cn("w-full rounded-xl border border-gray-200 dark:border-gray-700", "bg-gray-50 dark:bg-gray-800 px-4 py-3 text-sm", "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500", "resize-none transition-all duration-200", "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10")} />
        </div>
      )}

      {/* Video upload zone */}
      {showVideo && (
        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">{t("videoDescription")}</label>
          {formData.videoUrl ? (
            <div className="space-y-3">
              <div className="aspect-video max-h-[300px] overflow-hidden rounded-xl bg-black">
                <video src={formData.videoUrl} controls className="h-full w-full object-contain" aria-label={t("videoDescription")}>
                  <track kind="captions" />
                </video>
              </div>
              <Button variant="ghost" size="auto" type="button" onClick={() => { updateField("videoUrl", ""); updateField("videoFile", null) }} className={cn("w-full rounded-xl border border-gray-200 dark:border-gray-700 px-4 py-2.5 text-sm font-medium", "text-gray-600 dark:text-gray-400 transition-all duration-200", "hover:border-rose-300 dark:hover:border-rose-500 hover:text-rose-600 dark:hover:text-rose-400")}>{t("removeVideo")}</Button>
            </div>
          ) : (
            <div
              onClick={() => setVideoModalOpen(true)}
              className={cn("flex flex-col items-center justify-center gap-3 rounded-xl border-2 border-dashed cursor-pointer", "border-gray-300 dark:border-gray-600 p-8", "transition-all duration-200", "hover:border-rose-400 dark:hover:border-rose-500")}
            >
              <Video className="h-8 w-8 text-gray-400 dark:text-gray-500" />
              <p className="text-sm text-gray-500 dark:text-gray-400">{t("videoUploadHint")}</p>
              <span className={cn("rounded-xl px-5 py-2.5 text-sm font-medium", "bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300", "transition-all duration-200", "hover:bg-rose-50 dark:hover:bg-rose-500/10 hover:text-rose-600 dark:hover:text-rose-400")}>
                {t("videoUploadButton")}
              </span>
              <p className="text-xs text-gray-400 dark:text-gray-500">MP4, WebM, MOV — 100 MB max</p>
            </div>
          )}
          <UploadModal
            open={videoModalOpen}
            onClose={() => setVideoModalOpen(false)}
            onUpload={async (file) => { updateField("videoFile", file); updateField("videoUrl", URL.createObjectURL(file)); setVideoModalOpen(false) }}
            accept="video/mp4,video/webm,video/quicktime"
            maxSize={VIDEO_MAX_SIZE}
            title={tUpload("addVideo")}
            description={tUpload("videoFormats")}
            uploading={false}
          />
        </div>
      )}

      {/* Skills */}
      <TagInput label={t("skills")} placeholder={t("skillsPlaceholder")} tags={formData.skills} max={MAX_TAGS} onChange={(tags) => updateField("skills", tags)} />

      {/* Applicant type (hidden for agencies) */}
      {!hideApplicantType && (
        <ApplicantTypeSelector value={formData.applicantType} onChange={(v) => updateField("applicantType", v)} />
      )}

      {/* About company card */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">{t("aboutCompany")}</label>
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
              {((user?.first_name?.charAt(0) ?? "") + (user?.last_name?.charAt(0) ?? "")).toUpperCase() || "?"}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">{user?.display_name ?? "---"}</p>
              <div className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
                <Building2 className="h-3 w-3" strokeWidth={1.5} />
                <span className="capitalize">{user?.role ?? "---"}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

type TagInputProps = { label: string; placeholder: string; tags: string[]; max: number; onChange: (tags: string[]) => void }

function TagInput({ label, placeholder, tags, max, onChange }: TagInputProps) {
  const [inputValue, setInputValue] = useState("")
  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") { e.preventDefault(); addTag() }
    if (e.key === "Backspace" && inputValue === "" && tags.length > 0) onChange(tags.slice(0, -1))
  }
  function addTag() {
    const trimmed = inputValue.trim()
    if (!trimmed || tags.length >= max || tags.some((t) => t.toLowerCase() === trimmed.toLowerCase())) return
    onChange([...tags, trimmed])
    setInputValue("")
  }
  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</label>
        <span className="text-xs tabular-nums text-gray-400 dark:text-gray-500">{tags.length}/{max}</span>
      </div>
      <div className={cn("flex flex-wrap items-center gap-2 rounded-xl border border-gray-200 dark:border-gray-700", "bg-gray-50 dark:bg-gray-800 px-3 py-2.5 min-h-[48px]", "transition-all duration-200", "focus-within:border-rose-500 focus-within:bg-white dark:focus-within:bg-gray-900 focus-within:ring-4 focus-within:ring-rose-500/10")}>
        {tags.map((tag, index) => (
          <span key={tag} className={cn("inline-flex items-center gap-1 rounded-lg px-2.5 py-1", "bg-rose-100 dark:bg-rose-500/20 text-sm font-medium", "text-rose-700 dark:text-rose-300", "animate-scale-in")}>
            {tag}
            <Button variant="ghost" size="auto" type="button" onClick={() => onChange(tags.filter((_, i) => i !== index))} className="rounded p-0.5 transition-colors hover:bg-rose-200 dark:hover:bg-rose-500/30" aria-label={`Remove ${tag}`}>
              <X className="h-3 w-3" strokeWidth={2.5} />
            </Button>
          </span>
        ))}
        {tags.length < max && (
          <Input type="text" value={inputValue} onChange={(e) => setInputValue(e.target.value)} onKeyDown={handleKeyDown} onBlur={addTag} placeholder={tags.length === 0 ? placeholder : ""} className={cn("min-w-[120px] flex-1 bg-transparent text-sm", "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500", "focus:outline-none")} />
        )}
      </div>
    </div>
  )
}
