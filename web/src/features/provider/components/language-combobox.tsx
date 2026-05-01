"use client"

import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react"
import { Globe, Search } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { LANGUAGE_OPTIONS } from "../lib/language-options"

export type LanguageComboboxLocale = "fr" | "en"

interface LanguageComboboxProps {
  selectedCodes: string[]
  locale: LanguageComboboxLocale
  onPick: (code: string) => void
}

// Accessible combobox for the project's curated language catalog.
// Filters as the user types, surfaces matches in a listbox, and
// commits on Enter/click. Already-selected codes are hidden from
// the dropdown so the user cannot add the same language twice.
export function LanguageCombobox({
  selectedCodes,
  locale,
  onPick,
}: LanguageComboboxProps) {
  const t = useTranslations("profile.languages")
  const [query, setQuery] = useState("")
  const [isOpen, setIsOpen] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)
  const inputId = useId()
  const listboxId = useId()
  const containerRef = useRef<HTMLDivElement>(null)

  const selectedSet = useMemo(() => new Set(selectedCodes), [selectedCodes])
  const matches = useMemo(
    () => filterLanguages(query, selectedSet, locale),
    [query, selectedSet, locale],
  )

  // Reset the active highlight when the search input or selected
  // language list changes. Tracking these inputs in render state lets
  // us reset without setState-in-effect.
  const queryAndSelection = `${query}|${selectedCodes.join(",")}`
  const [lastQueryAndSelection, setLastQueryAndSelection] = useState(
    queryAndSelection,
  )
  if (queryAndSelection !== lastQueryAndSelection) {
    setLastQueryAndSelection(queryAndSelection)
    setActiveIndex(0)
  }

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

  const pick = useCallback(
    (code: string) => {
      onPick(code)
      setQuery("")
      setActiveIndex(0)
    },
    [onPick],
  )

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLInputElement>) => {
      const count = matches.length
      if (!isOpen && event.key !== "Tab") setIsOpen(true)
      const key = event.key
      if (key === "ArrowDown" || key === "ArrowUp") {
        event.preventDefault()
        if (count === 0) return
        const delta = key === "ArrowDown" ? 1 : -1
        setActiveIndex((i) => (i + delta + count) % count)
        return
      }
      if (key === "Home" || key === "End") {
        event.preventDefault()
        if (count > 0) setActiveIndex(key === "Home" ? 0 : count - 1)
        return
      }
      if (key === "Enter") {
        event.preventDefault()
        const row = matches[activeIndex]
        if (row) pick(row.code)
        return
      }
      if (key === "Escape") {
        event.preventDefault()
        event.stopPropagation()
        setIsOpen(false)
      }
    },
    [activeIndex, isOpen, matches, pick],
  )

  const activeId =
    isOpen && matches.length > 0 ? `${listboxId}-option-${activeIndex}` : undefined

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
          value={query}
          onChange={(event) => {
            setQuery(event.target.value)
            setIsOpen(true)
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={t("searchPlaceholder")}
          autoComplete="off"
          role="combobox"
          aria-expanded={isOpen}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={activeId}
          className={cn(
            "h-10 w-full rounded-lg border border-border bg-background pl-9 pr-3 text-sm shadow-xs",
            "placeholder:text-muted-foreground",
            "focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none",
            "transition-colors duration-150",
          )}
        />
      </div>
      {isOpen ? (
        <LanguageComboboxDropdown
          dropdownId={listboxId}
          state={{ matches, query, activeIndex }}
          onPick={pick}
          onHover={setActiveIndex}
        />
      ) : null}
    </div>
  )
}

interface LanguageMatch {
  code: string
  label: string
  matchStart: number
  matchEnd: number
}

function filterLanguages(
  query: string,
  selectedSet: Set<string>,
  locale: LanguageComboboxLocale,
): LanguageMatch[] {
  const normalized = query.trim().toLowerCase()
  const out: LanguageMatch[] = []
  for (const option of LANGUAGE_OPTIONS) {
    if (selectedSet.has(option.code)) continue
    const label = locale === "fr" ? option.labelFr : option.labelEn
    const lower = label.toLowerCase()
    const idx = normalized ? lower.indexOf(normalized) : 0
    if (normalized && idx === -1) continue
    out.push({
      code: option.code,
      label,
      matchStart: normalized ? idx : 0,
      matchEnd: normalized ? idx + normalized.length : 0,
    })
  }
  return out
}

interface DropdownState {
  matches: LanguageMatch[]
  query: string
  activeIndex: number
}

interface LanguageComboboxDropdownProps {
  dropdownId: string
  state: DropdownState
  onPick: (code: string) => void
  onHover: (index: number) => void
}

function LanguageComboboxDropdown({
  dropdownId,
  state,
  onPick,
  onHover,
}: LanguageComboboxDropdownProps) {
  const t = useTranslations("profile.languages")
  const { matches, query, activeIndex } = state

  if (matches.length === 0) {
    return (
      <div
        id={dropdownId}
        role="listbox"
        className={cn(
          "absolute z-20 mt-1 w-full rounded-lg border border-border bg-popover",
          "shadow-md animate-scale-in",
        )}
      >
        <p className="px-3 py-3 text-sm text-muted-foreground">
          {t("noResults")}
        </p>
      </div>
    )
  }

  return (
    <ul
      id={dropdownId}
      role="listbox"
      className={cn(
        "absolute z-20 mt-1 max-h-64 w-full overflow-y-auto rounded-lg",
        "border border-border bg-popover shadow-md animate-scale-in",
      )}
    >
      {matches.map((match, index) => (
        <LanguageOptionRow
          key={match.code}
          rowId={`${dropdownId}-option-${index}`}
          match={match}
          highlight={{ query, isActive: index === activeIndex }}
          actions={{ pick: () => onPick(match.code), hover: () => onHover(index) }}
        />
      ))}
    </ul>
  )
}

interface LanguageOptionRowProps {
  rowId: string
  match: LanguageMatch
  highlight: { query: string; isActive: boolean }
  actions: { pick: () => void; hover: () => void }
}

function LanguageOptionRow({
  rowId,
  match,
  highlight,
  actions,
}: LanguageOptionRowProps) {
  return (
    <li
      id={rowId}
      role="option"
      aria-selected={highlight.isActive}
      onMouseEnter={actions.hover}
      onMouseDown={(event) => {
        // onMouseDown so the pick commits before the input's blur
        // tears down the dropdown on click.
        event.preventDefault()
        actions.pick()
      }}
      className={cn(
        "flex cursor-pointer items-center gap-2 px-3 py-2 text-sm text-foreground",
        "transition-colors duration-150",
        highlight.isActive && "bg-muted",
      )}
    >
      <Globe
        className="h-3.5 w-3.5 text-muted-foreground"
        aria-hidden="true"
        strokeWidth={2.25}
      />
      <HighlightedLabel
        label={match.label}
        query={highlight.query}
        range={{ start: match.matchStart, end: match.matchEnd }}
      />
    </li>
  )
}

interface HighlightedLabelProps {
  label: string
  query: string
  range: { start: number; end: number }
}

function HighlightedLabel({ label, query, range }: HighlightedLabelProps) {
  const { start, end } = range
  if (!query.trim() || start < 0 || end <= start) {
    return <span className="truncate">{label}</span>
  }
  return (
    <span className="truncate">
      {label.slice(0, start)}
      <mark className="bg-primary/15 text-primary font-semibold rounded px-0.5">
        {label.slice(start, end)}
      </mark>
      {label.slice(end)}
    </span>
  )
}
