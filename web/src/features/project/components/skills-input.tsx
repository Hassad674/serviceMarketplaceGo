"use client"

import { useState } from "react"
import { X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

type SkillsInputProps = {
  skills: string[]
  onChange: (skills: string[]) => void
}

export function SkillsInput({ skills, onChange }: SkillsInputProps) {
  const t = useTranslations("projects")
  const [inputValue, setInputValue] = useState("")

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault()
      addSkill()
    }
    if (e.key === "Backspace" && inputValue === "" && skills.length > 0) {
      onChange(skills.slice(0, -1))
    }
  }

  function addSkill() {
    const trimmed = inputValue.trim()
    if (trimmed === "") return
    if (skills.some((s) => s.toLowerCase() === trimmed.toLowerCase())) return
    onChange([...skills, trimmed])
    setInputValue("")
  }

  function removeSkill(index: number) {
    onChange(skills.filter((_, i) => i !== index))
  }

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {t("requiredSkills")}
      </label>
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 rounded-xl border border-gray-200 dark:border-gray-700",
          "bg-gray-50 dark:bg-gray-800 px-3 py-2.5 min-h-[48px]",
          "transition-all duration-200",
          "focus-within:border-rose-500 focus-within:bg-white dark:focus-within:bg-gray-900 focus-within:ring-4 focus-within:ring-rose-500/10",
        )}
      >
        {skills.map((skill, index) => (
          <span
            key={skill}
            className={cn(
              "inline-flex items-center gap-1 rounded-lg px-2.5 py-1",
              "bg-rose-100 dark:bg-rose-500/20 text-sm font-medium",
              "text-rose-700 dark:text-rose-300",
              "animate-scale-in",
            )}
          >
            {skill}
            <button
              type="button"
              onClick={() => removeSkill(index)}
              className="rounded p-0.5 transition-colors hover:bg-rose-200 dark:hover:bg-rose-500/30"
              aria-label={`Remove ${skill}`}
            >
              <X className="h-3 w-3" strokeWidth={2.5} />
            </button>
          </span>
        ))}
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={addSkill}
          placeholder={skills.length === 0 ? t("skillsPlaceholder") : ""}
          className={cn(
            "min-w-[120px] flex-1 bg-transparent text-sm",
            "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
            "focus:outline-none",
          )}
        />
      </div>
    </div>
  )
}
