"use client"

import { useState } from "react"
import { X, Building2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import type { JobFormData } from "../types"
import { ContractorCount } from "./contractor-count"
import { ApplicantTypeSelector } from "./applicant-type-selector"

type JobDetailsSectionProps = {
  formData: JobFormData
  updateField: <K extends keyof JobFormData>(field: K, value: JobFormData[K]) => void
}

const TITLE_MAX_LENGTH = 100
const MAX_TAGS = 5

export function JobDetailsSection({ formData, updateField }: JobDetailsSectionProps) {
  const t = useTranslations("job")
  const { data: user } = useUser()

  return (
    <div className="space-y-5">
      {/* Job title */}
      <div>
        <div className="mb-1.5 flex items-center justify-between">
          <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("jobTitle")}
          </label>
          <span
            className={cn(
              "text-xs tabular-nums",
              formData.title.length > TITLE_MAX_LENGTH
                ? "text-red-500"
                : "text-gray-400 dark:text-gray-500",
            )}
          >
            {formData.title.length}/{TITLE_MAX_LENGTH}
          </span>
        </div>
        <input
          type="text"
          value={formData.title}
          onChange={(e) => updateField("title", e.target.value)}
          maxLength={TITLE_MAX_LENGTH}
          placeholder={t("jobTitlePlaceholder")}
          className={cn(
            "h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700",
            "bg-gray-50 dark:bg-gray-800 px-4 text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          )}
        />
      </div>

      {/* Job description */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("jobDescription")}
        </label>
        <textarea
          value={formData.description}
          onChange={(e) => updateField("description", e.target.value)}
          rows={5}
          className={cn(
            "w-full rounded-xl border border-gray-200 dark:border-gray-700",
            "bg-gray-50 dark:bg-gray-800 px-4 py-3 text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "resize-none transition-all duration-200",
            "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
          )}
        />
      </div>

      {/* Skills */}
      <TagInput
        label={t("skills")}
        placeholder={t("skillsPlaceholder")}
        tags={formData.skills}
        max={MAX_TAGS}
        onChange={(tags) => updateField("skills", tags)}
      />

      {/* Tools */}
      <TagInput
        label={t("tools")}
        placeholder={t("toolsPlaceholder")}
        tags={formData.tools}
        max={MAX_TAGS}
        onChange={(tags) => updateField("tools", tags)}
      />

      {/* Contractor count */}
      <ContractorCount
        label={t("contractorCount")}
        value={formData.contractorCount}
        onChange={(v) => updateField("contractorCount", v)}
      />

      {/* Applicant type */}
      <ApplicantTypeSelector
        value={formData.applicantType}
        onChange={(v) => updateField("applicantType", v)}
      />

      {/* About company card */}
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("aboutCompany")}
        </label>
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
              {user
                ? `${user.first_name.charAt(0)}${user.last_name.charAt(0)}`
                : "?"}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">
                {user?.display_name ?? "---"}
              </p>
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

/* -------------------------------------------------- */
/* Tag input (reusable within this feature)           */
/* -------------------------------------------------- */

type TagInputProps = {
  label: string
  placeholder: string
  tags: string[]
  max: number
  onChange: (tags: string[]) => void
}

function TagInput({ label, placeholder, tags, max, onChange }: TagInputProps) {
  const [inputValue, setInputValue] = useState("")

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault()
      addTag()
    }
    if (e.key === "Backspace" && inputValue === "" && tags.length > 0) {
      onChange(tags.slice(0, -1))
    }
  }

  function addTag() {
    const trimmed = inputValue.trim()
    if (trimmed === "") return
    if (tags.length >= max) return
    if (tags.some((t) => t.toLowerCase() === trimmed.toLowerCase())) return
    onChange([...tags, trimmed])
    setInputValue("")
  }

  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </label>
        <span className="text-xs tabular-nums text-gray-400 dark:text-gray-500">
          {tags.length}/{max}
        </span>
      </div>
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 rounded-xl border border-gray-200 dark:border-gray-700",
          "bg-gray-50 dark:bg-gray-800 px-3 py-2.5 min-h-[48px]",
          "transition-all duration-200",
          "focus-within:border-rose-500 focus-within:bg-white dark:focus-within:bg-gray-900 focus-within:ring-4 focus-within:ring-rose-500/10",
        )}
      >
        {tags.map((tag, index) => (
          <span
            key={tag}
            className={cn(
              "inline-flex items-center gap-1 rounded-lg px-2.5 py-1",
              "bg-rose-100 dark:bg-rose-500/20 text-sm font-medium",
              "text-rose-700 dark:text-rose-300",
              "animate-scale-in",
            )}
          >
            {tag}
            <button
              type="button"
              onClick={() => onChange(tags.filter((_, i) => i !== index))}
              className="rounded p-0.5 transition-colors hover:bg-rose-200 dark:hover:bg-rose-500/30"
              aria-label={`Remove ${tag}`}
            >
              <X className="h-3 w-3" strokeWidth={2.5} />
            </button>
          </span>
        ))}
        {tags.length < max && (
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            onBlur={addTag}
            placeholder={tags.length === 0 ? placeholder : ""}
            className={cn(
              "min-w-[120px] flex-1 bg-transparent text-sm",
              "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
              "focus:outline-none",
            )}
          />
        )}
      </div>
    </div>
  )
}
