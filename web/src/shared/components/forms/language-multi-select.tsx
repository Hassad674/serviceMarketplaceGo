"use client"

import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react"
import { Globe, X } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  LANGUAGE_OPTIONS,
  getLanguageLabel,
} from "@/shared/lib/profile/language-options"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

/**
 * LanguageMultiSelect — Soleil v2 multi-select combobox over the
 * curated ISO 639-1 catalog. Adds three guarantees the previous
 * pill list did not have:
 *
 *   1. Type to search (filters by localized label, accent-insensitive).
 *   2. Multiple selections, surfaced as removable badges above the input.
 *   3. Listbox-keyboard navigation: Up/Down to traverse, Enter/Tab
 *      to commit, Backspace on an empty input to remove the last pick.
 *
 * The catalog is small (~30 entries) and statically imported, so no
 * heavy combobox library is needed and the component renders synchronously.
 *
 * Used by both the search filter sidebar and (eventually) the
 * profile language section, replacing the toggle-pill row that
 * forced the user to scan all options visually.
 */
export interface LanguageMultiSelectProps {
  /** Active language codes (ISO 639-1, lowercase). */
  selected: string[]
  /** Commit handler. Receives the next full list of codes. */
  onChange: (next: string[]) => void
  /** Optional placeholder shown inside the search input. */
  placeholder?: string
  /** Optional aria-label when the parent did not render a visible label. */
  ariaLabel?: string
  className?: string
  disabled?: boolean
}

export function LanguageMultiSelect({
  selected,
  onChange,
  placeholder,
  ariaLabel,
  className,
  disabled,
}: LanguageMultiSelectProps) {
  const t = useTranslations("forms.languageMultiSelect")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const inputId = useId()
  const listboxId = useId()
  const containerRef = useRef<HTMLDivElement>(null)
  const [draft, setDraft] = useState("")
  const [isOpen, setIsOpen] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)

  const selectedSet = useMemo(() => new Set(selected), [selected])
  const matches = useMemo(
    () => filterLanguages(draft, selectedSet, locale),
    [draft, selectedSet, locale],
  )

  // Reset the active highlight on draft / selection change so the
  // keyboard cursor never points past the dropdown's last row.
  const matchKey = `${draft}|${selected.join(",")}`
  const [lastMatchKey, setLastMatchKey] = useState(matchKey)
  if (matchKey !== lastMatchKey) {
    setLastMatchKey(matchKey)
    setActiveIndex(0)
  }

  // Close the dropdown when focus leaves the whole component. We
  // do not clear the draft so the user can re-open without retyping.
  useEffect(() => {
    const onDocumentMouseDown = (event: MouseEvent) => {
      const root = containerRef.current
      if (!root) return
      if (root.contains(event.target as Node)) return
      setIsOpen(false)
    }
    document.addEventListener("mousedown", onDocumentMouseDown)
    return () => document.removeEventListener("mousedown", onDocumentMouseDown)
  }, [])

  const pick = useCallback(
    (code: string) => {
      if (selectedSet.has(code)) return
      onChange([...selected, code])
      setDraft("")
    },
    [onChange, selected, selectedSet],
  )

  const remove = useCallback(
    (code: string) => {
      onChange(selected.filter((c) => c !== code))
    },
    [onChange, selected],
  )

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (!isOpen && event.key !== "Tab") setIsOpen(true)
    if (event.key === "ArrowDown") {
      event.preventDefault()
      if (matches.length > 0) {
        setActiveIndex((i) => (i + 1) % matches.length)
      }
      return
    }
    if (event.key === "ArrowUp") {
      event.preventDefault()
      if (matches.length > 0) {
        setActiveIndex((i) => (i <= 0 ? matches.length - 1 : i - 1))
      }
      return
    }
    if (event.key === "Enter" && matches.length > 0) {
      event.preventDefault()
      const row = matches[activeIndex]
      if (row) pick(row.code)
      return
    }
    if (
      event.key === "Backspace" &&
      draft.length === 0 &&
      selected.length > 0
    ) {
      remove(selected[selected.length - 1])
      return
    }
    if (event.key === "Escape") {
      event.preventDefault()
      setIsOpen(false)
    }
  }

  const activeId =
    isOpen && matches.length > 0
      ? `${listboxId}-option-${activeIndex}`
      : undefined

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      {selected.length > 0 ? (
        <SelectedBadgesRow
          codes={selected}
          locale={locale}
          onRemove={remove}
        />
      ) : null}
      <div className="relative">
        <Globe
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          id={inputId}
          type="text"
          value={draft}
          disabled={disabled}
          onChange={(event) => {
            setDraft(event.target.value)
            setIsOpen(true)
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder ?? t("placeholder")}
          aria-label={ariaLabel ?? t("placeholder")}
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
          )}
        />
      </div>
      {isOpen ? (
        <Dropdown
          listboxId={listboxId}
          matches={matches}
          activeIndex={activeIndex}
          onPick={pick}
          onHover={setActiveIndex}
          emptyLabel={t("noResults")}
        />
      ) : null}
    </div>
  )
}

interface LanguageMatch {
  code: string
  label: string
}

function filterLanguages(
  query: string,
  selectedSet: Set<string>,
  locale: "fr" | "en",
): LanguageMatch[] {
  const normalized = query.trim().toLowerCase()
  const matches: LanguageMatch[] = []
  for (const option of LANGUAGE_OPTIONS) {
    if (selectedSet.has(option.code)) continue
    const label =
      locale === "fr" ? option.labelFr : option.labelEn
    if (normalized && !label.toLowerCase().includes(normalized)) continue
    matches.push({ code: option.code, label })
  }
  return matches
}

interface SelectedBadgesRowProps {
  codes: string[]
  locale: "fr" | "en"
  onRemove: (code: string) => void
}

function SelectedBadgesRow({
  codes,
  locale,
  onRemove,
}: SelectedBadgesRowProps) {
  const t = useTranslations("forms.languageMultiSelect")
  return (
    <ul
      className="mb-2 flex flex-wrap gap-1.5"
      aria-label={t("selectedAriaLabel")}
    >
      {codes.map((code) => (
        <li key={code}>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => onRemove(code)}
            aria-label={t("removeLabel", {
              language: getLanguageLabel(code, locale),
            })}
            className="inline-flex items-center gap-1 rounded-full bg-primary-soft px-2.5 py-1 text-xs font-medium text-primary-deep transition-colors hover:bg-primary/30"
          >
            <span>{getLanguageLabel(code, locale)}</span>
            <X className="h-3 w-3" aria-hidden strokeWidth={2.5} />
          </Button>
        </li>
      ))}
    </ul>
  )
}

interface DropdownProps {
  listboxId: string
  matches: LanguageMatch[]
  activeIndex: number
  onPick: (code: string) => void
  onHover: (index: number) => void
  emptyLabel: string
}

function Dropdown({
  listboxId,
  matches,
  activeIndex,
  onPick,
  onHover,
  emptyLabel,
}: DropdownProps) {
  if (matches.length === 0) {
    return (
      <div
        id={listboxId}
        role="listbox"
        className="absolute z-20 mt-1 w-full rounded-lg border border-border bg-popover shadow-md animate-scale-in"
      >
        <p className="px-3 py-3 text-sm text-muted-foreground">{emptyLabel}</p>
      </div>
    )
  }
  return (
    <ul
      id={listboxId}
      role="listbox"
      className="absolute z-20 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-border bg-popover shadow-md animate-scale-in"
    >
      {matches.map((match, index) => (
        <li
          key={match.code}
          id={`${listboxId}-option-${index}`}
          role="option"
          aria-selected={index === activeIndex}
          onMouseEnter={() => onHover(index)}
          onMouseDown={(event) => {
            event.preventDefault()
            onPick(match.code)
          }}
          className={cn(
            "flex cursor-pointer items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors duration-150",
            index === activeIndex && "bg-muted",
          )}
        >
          <Globe
            className="h-3.5 w-3.5 text-muted-foreground"
            aria-hidden
            strokeWidth={2.25}
          />
          <span className="truncate">{match.label}</span>
        </li>
      ))}
    </ul>
  )
}
