"use client"

import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react"
import { Loader2, Plus, Search } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useSkillAutocomplete } from "../hooks/use-skill-autocomplete"
import { useCreateUserSkill } from "../hooks/use-create-user-skill"
import type { SkillResponse } from "../types"

interface SkillSearchBarProps {
  alreadySelected: Set<string>
  onAdd: (skill: SkillResponse) => void
  disabled?: boolean
}

// Controlled autocomplete input with a suggestion dropdown. The
// caller controls whether a skill is addable via `alreadySelected`
// (we filter out duplicates) and via `disabled` (global cap reached).
//
// Keyboard contract:
//   - ArrowDown / ArrowUp cycle through rows
//   - Enter picks the highlighted row (or triggers "Create" when no
//     curated match is found)
//   - Escape clears focus on the dropdown
export function SkillSearchBar({
  alreadySelected,
  onAdd,
  disabled = false,
}: SkillSearchBarProps) {
  const t = useTranslations("profile.skills")
  const [input, setInput] = useState("")
  const [activeIndex, setActiveIndex] = useState(0)
  const [isOpen, setIsOpen] = useState(false)
  const inputId = useId()
  const listboxId = useId()
  const containerRef = useRef<HTMLDivElement>(null)
  const autocomplete = useSkillAutocomplete(input)
  const createMutation = useCreateUserSkill()

  const suggestions = useMemo<SkillResponse[]>(
    () =>
      (autocomplete.data ?? []).filter(
        (skill) => !alreadySelected.has(skill.skill_text),
      ),
    [autocomplete.data, alreadySelected],
  )

  const trimmed = input.trim()
  const hasExactMatch = suggestions.some(
    (skill) => skill.display_text.toLowerCase() === trimmed.toLowerCase(),
  )
  const canCreate = trimmed.length > 0 && !hasExactMatch
  const rows: Array<{ type: "suggestion" | "create"; skill?: SkillResponse }> =
    useMemo(() => {
      const out: Array<{
        type: "suggestion" | "create"
        skill?: SkillResponse
      }> = suggestions.map((skill) => ({ type: "suggestion", skill }))
      if (canCreate) out.push({ type: "create" })
      return out
    }, [suggestions, canCreate])

  useEffect(() => {
    setActiveIndex(0)
  }, [input])

  // Close the dropdown when the user clicks outside the input/menu.
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (!containerRef.current) return
      if (!containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [])

  const pickSuggestion = useCallback(
    (skill: SkillResponse) => {
      onAdd(skill)
      setInput("")
      setActiveIndex(0)
    },
    [onAdd],
  )

  const handleCreate = useCallback(async () => {
    if (!trimmed) return
    try {
      const created = await createMutation.mutateAsync(trimmed)
      if (!alreadySelected.has(created.skill_text)) {
        onAdd(created)
      }
      setInput("")
      setActiveIndex(0)
    } catch {
      // Swallow — UI level error surfacing happens in the parent
      // modal via its own error boundary. The mutation stays in
      // error state so the dropdown can still highlight it.
    }
  }, [alreadySelected, createMutation, onAdd, trimmed])

  const activateRow = useCallback(
    (index: number) => {
      const row = rows[index]
      if (!row) return
      if (row.type === "create") {
        void handleCreate()
        return
      }
      if (row.skill) pickSuggestion(row.skill)
    },
    [rows, pickSuggestion, handleCreate],
  )

  function handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (!isOpen) setIsOpen(true)
    if (event.key === "ArrowDown") {
      event.preventDefault()
      setActiveIndex((i) => (rows.length === 0 ? 0 : (i + 1) % rows.length))
      return
    }
    if (event.key === "ArrowUp") {
      event.preventDefault()
      setActiveIndex((i) =>
        rows.length === 0 ? 0 : (i - 1 + rows.length) % rows.length,
      )
      return
    }
    if (event.key === "Enter") {
      event.preventDefault()
      activateRow(activeIndex)
      return
    }
    if (event.key === "Escape") {
      setIsOpen(false)
    }
  }

  const showDropdown = isOpen && trimmed.length > 0

  return (
    <div className="relative" ref={containerRef}>
      <label htmlFor={inputId} className="sr-only">
        {t("searchPlaceholder")}
      </label>
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <input
          id={inputId}
          type="text"
          value={input}
          onChange={(event) => {
            setInput(event.target.value)
            setIsOpen(true)
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={t("searchPlaceholder")}
          disabled={disabled}
          autoComplete="off"
          role="combobox"
          aria-expanded={showDropdown}
          aria-controls={listboxId}
          aria-activedescendant={
            showDropdown && rows.length > 0
              ? `${listboxId}-option-${activeIndex}`
              : undefined
          }
          className={cn(
            "h-10 w-full rounded-lg border border-border bg-background pl-9 pr-3 text-sm",
            "placeholder:text-muted-foreground",
            "focus-visible:border-primary focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-0",
            disabled && "opacity-60 cursor-not-allowed",
          )}
        />
      </div>
      {showDropdown ? (
        <SkillSearchDropdown
          listboxId={listboxId}
          rows={rows}
          activeIndex={activeIndex}
          isLoading={autocomplete.isFetching}
          createMutationPending={createMutation.isPending}
          query={trimmed}
          onActivate={activateRow}
          onHover={setActiveIndex}
        />
      ) : null}
    </div>
  )
}

type DropdownRow = { type: "suggestion" | "create"; skill?: SkillResponse }

type SkillSearchDropdownProps = {
  listboxId: string
  rows: DropdownRow[]
  activeIndex: number
  isLoading: boolean
  createMutationPending: boolean
  query: string
  onActivate: (index: number) => void
  onHover: (index: number) => void
}

function SkillSearchDropdown(props: SkillSearchDropdownProps) {
  const {
    listboxId,
    rows,
    activeIndex,
    isLoading,
    createMutationPending,
    query,
    onActivate,
    onHover,
  } = props
  const t = useTranslations("profile.skills")

  return (
    <div
      id={listboxId}
      role="listbox"
      className={cn(
        "absolute z-20 mt-1 max-h-72 w-full overflow-y-auto rounded-lg border",
        "border-border bg-background shadow-lg",
      )}
    >
      {isLoading && rows.length === 0 ? (
        <p className="flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
          <span>{t("searching")}</span>
        </p>
      ) : null}
      {rows.length === 0 && !isLoading ? (
        <p className="px-3 py-2 text-sm text-muted-foreground">
          {t("noResults", { query })}
        </p>
      ) : null}
      {rows.map((row, index) => (
        <SkillSearchRow
          key={row.type === "suggestion" ? row.skill?.skill_text : "__create__"}
          row={row}
          index={index}
          listboxId={listboxId}
          isActive={activeIndex === index}
          createMutationPending={createMutationPending}
          query={query}
          onActivate={onActivate}
          onHover={onHover}
        />
      ))}
    </div>
  )
}

type SkillSearchRowProps = {
  row: DropdownRow
  index: number
  listboxId: string
  isActive: boolean
  createMutationPending: boolean
  query: string
  onActivate: (index: number) => void
  onHover: (index: number) => void
}

function SkillSearchRow(props: SkillSearchRowProps) {
  const {
    row,
    index,
    listboxId,
    isActive,
    createMutationPending,
    query,
    onActivate,
    onHover,
  } = props
  const t = useTranslations("profile.skills")
  const baseClass = cn(
    "flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm",
    "hover:bg-muted",
    isActive && "bg-muted",
  )

  if (row.type === "create") {
    return (
      <button
        id={`${listboxId}-option-${index}`}
        type="button"
        role="option"
        aria-selected={isActive}
        onMouseEnter={() => onHover(index)}
        onClick={() => onActivate(index)}
        disabled={createMutationPending}
        className={baseClass}
      >
        <span className="flex items-center gap-2 text-primary">
          <Plus className="h-4 w-4" aria-hidden="true" />
          {t("createNew", { query })}
        </span>
        {createMutationPending ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
        ) : null}
      </button>
    )
  }

  const skill = row.skill
  if (!skill) return null
  return (
    <button
      id={`${listboxId}-option-${index}`}
      type="button"
      role="option"
      aria-selected={isActive}
      onMouseEnter={() => onHover(index)}
      onClick={() => onActivate(index)}
      className={baseClass}
    >
      <span className="truncate font-medium text-foreground">
        {skill.display_text}
      </span>
      <span className="text-xs text-muted-foreground">
        {t("usageCount", { count: skill.usage_count })}
      </span>
    </button>
  )
}
