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
import { Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  CITY_SEARCH_MIN_CHARS,
  searchCities,
  type CitySearchResult,
} from "../lib/city-search"

// Canonical selection shape persisted on the profile. A null value
// means "nothing selected yet" — the user has not picked anything.
export type CitySelection = {
  city: string
  countryCode: string
  latitude: number
  longitude: number
}

type Props = {
  value: CitySelection | null
  countryCode: string
  onChange: (next: CitySelection | null) => void
  disabled?: boolean
}

const DEBOUNCE_MS = 250

export function CityAutocomplete({ value, countryCode, onChange, disabled }: Props) {
  const t = useTranslations("profile.location")
  const listboxId = useId()
  const [query, setQuery] = useState(value?.city ?? "")
  const [results, setResults] = useState<CitySearchResult[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [activeIndex, setActiveIndex] = useState(-1)
  const rootRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  // Keep the visible query synced with the persisted canonical
  // selection — e.g. when TanStack Query refetches the profile or
  // when the parent clears the value after a country change.
  useEffect(() => {
    setQuery(value?.city ?? "")
  }, [value])

  // Debounced search. Cancels any in-flight request when the query
  // changes so the dropdown never shows stale matches.
  useEffect(() => {
    const trimmed = query.trim()
    if (trimmed.length < CITY_SEARCH_MIN_CHARS) {
      setResults([])
      setIsLoading(false)
      return
    }
    if (value && trimmed === value.city) {
      // Don't re-query for the city we already have selected —
      // this avoids a flicker right after a selection.
      return
    }
    const handle = window.setTimeout(() => {
      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller
      setIsLoading(true)
      searchCities(trimmed, countryCode, controller.signal)
        .then((next) => {
          if (controller.signal.aborted) return
          setResults(next)
          setActiveIndex(next.length > 0 ? 0 : -1)
        })
        .catch((error: unknown) => {
          if (controller.signal.aborted) return
          if (error instanceof DOMException && error.name === "AbortError") return
          setResults([])
        })
        .finally(() => {
          if (!controller.signal.aborted) setIsLoading(false)
        })
    }, DEBOUNCE_MS)
    return () => window.clearTimeout(handle)
  }, [query, countryCode, value])

  // Close the dropdown when focus leaves the whole component.
  useEffect(() => {
    const onDocumentMouseDown = (e: MouseEvent) => {
      const root = rootRef.current
      if (!root) return
      if (root.contains(e.target as Node)) return
      setIsOpen(false)
      // Restore the canonical selection if the user left without
      // picking anything — bare text is never savable.
      setQuery(value?.city ?? "")
    }
    document.addEventListener("mousedown", onDocumentMouseDown)
    return () => document.removeEventListener("mousedown", onDocumentMouseDown)
  }, [value])

  const commitSelection = useCallback(
    (pick: CitySearchResult) => {
      onChange({
        city: pick.city,
        countryCode: pick.countryCode,
        latitude: pick.latitude,
        longitude: pick.longitude,
      })
      setQuery(pick.city)
      setResults([])
      setIsOpen(false)
      setActiveIndex(-1)
    },
    [onChange],
  )

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (!isOpen || results.length === 0) {
      if (e.key === "ArrowDown" && results.length > 0) setIsOpen(true)
      return
    }
    if (e.key === "ArrowDown") {
      e.preventDefault()
      setActiveIndex((i) => (i + 1) % results.length)
      return
    }
    if (e.key === "ArrowUp") {
      e.preventDefault()
      setActiveIndex((i) => (i <= 0 ? results.length - 1 : i - 1))
      return
    }
    if (e.key === "Enter") {
      e.preventDefault()
      const pick = results[activeIndex] ?? results[0]
      if (pick) commitSelection(pick)
      return
    }
    if (e.key === "Escape") {
      setIsOpen(false)
      setQuery(value?.city ?? "")
    }
  }

  const handleInputChange = (next: string) => {
    setQuery(next)
    setIsOpen(true)
    // Typing invalidates the previous canonical selection — the
    // parent sees null until the user picks a row from the list.
    if (value) onChange(null)
  }

  const emptyState = useMemo(() => {
    if (isLoading) return null
    if (query.trim().length < CITY_SEARCH_MIN_CHARS) return t("cityAutocompleteHint")
    if (results.length === 0) return t("cityAutocompleteEmpty")
    return null
  }, [isLoading, query, results.length, t])

  const showDropdown = isOpen && (isLoading || results.length > 0 || emptyState !== null)

  return (
    <div ref={rootRef} className="relative">
      <div className="relative">
        <input
          type="text"
          role="combobox"
          aria-expanded={showDropdown}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            activeIndex >= 0 ? `${listboxId}-option-${activeIndex}` : undefined
          }
          value={query}
          disabled={disabled}
          onChange={(e) => handleInputChange(e.target.value)}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={t("cityAutocompletePlaceholder")}
          className="w-full h-10 rounded-lg border border-border bg-background px-3 pr-9 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
        />
        {isLoading ? (
          <Loader2
            className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 animate-spin text-muted-foreground"
            aria-hidden="true"
          />
        ) : null}
      </div>

      {showDropdown ? (
        <ul
          id={listboxId}
          role="listbox"
          className="absolute z-20 mt-1 w-full max-h-72 overflow-auto rounded-lg border border-border bg-background shadow-lg animate-fade-in"
        >
          {results.map((result, index) => (
            <CityOption
              key={`${result.city}-${result.latitude}-${result.longitude}`}
              id={`${listboxId}-option-${index}`}
              result={result}
              isActive={index === activeIndex}
              onHover={() => setActiveIndex(index)}
              onSelect={() => commitSelection(result)}
            />
          ))}
          {results.length === 0 && emptyState ? (
            <li className="px-3 py-2 text-sm text-muted-foreground">{emptyState}</li>
          ) : null}
        </ul>
      ) : null}
    </div>
  )
}

type OptionProps = {
  id: string
  result: CitySearchResult
  isActive: boolean
  onHover: () => void
  onSelect: () => void
}

function CityOption({ id, result, isActive, onHover, onSelect }: OptionProps) {
  return (
    <li
      id={id}
      role="option"
      aria-selected={isActive}
      onMouseEnter={onHover}
      onMouseDown={(e) => {
        // Prevent the input from losing focus before the click fires,
        // which would otherwise cancel the selection via blur.
        e.preventDefault()
        onSelect()
      }}
      className={cn(
        "cursor-pointer px-3 py-2 text-sm transition-colors duration-150",
        isActive ? "bg-muted text-foreground" : "text-foreground hover:bg-muted/60",
      )}
    >
      <div className="font-medium">{result.city}</div>
      {result.context ? (
        <div className="text-xs text-muted-foreground">{result.context}</div>
      ) : null}
    </li>
  )
}
