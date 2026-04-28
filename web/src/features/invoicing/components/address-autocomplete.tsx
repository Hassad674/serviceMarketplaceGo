"use client"

import { useEffect, useRef, useState } from "react"
import { Loader2, MapPin, Search } from "lucide-react"
import { cn } from "@/shared/lib/utils"

// Free, key-less French government BAN (Base Adresse Nationale)
// search endpoint. Documented at https://adresse.data.gouv.fr/api-doc/adresse.
// We keep this client-side because the request is anonymous and
// adding a backend proxy would only add latency without adding
// security. The fetch is restricted to country === "FR" so we never
// hit the FR API with a non-FR query and confuse the user with
// French-only results.
const BAN_ENDPOINT = "https://api-adresse.data.gouv.fr/search/"

type BANFeature = {
  properties: {
    label: string
    name: string
    postcode: string
    city: string
    context: string
  }
}

export type AutocompleteAddress = {
  line1: string
  postalCode: string
  city: string
}

type AddressAutocompleteProps = {
  /** ISO-3166 alpha-2 country code (uppercase). Autocomplete only
   *  fires when country === "FR" — non-FR users get the standard
   *  manual fields below this component. */
  country: string
  onSelect: (a: AutocompleteAddress) => void
  /** Optional placeholder; FR-localised default. */
  placeholder?: string
  /** When false, the input renders disabled with an explanatory
   *  message — used by the form to communicate that autocomplete
   *  isn't available for the selected country. */
  disabled?: boolean
}

/**
 * Search field that calls the French gov BAN endpoint as the user
 * types and offers a small dropdown of full address matches. Picking
 * a suggestion fires `onSelect` with the address split into
 * `line1` / `postalCode` / `city`, which the parent form copies into
 * its own controlled inputs.
 *
 * The dropdown closes on outside click, on Escape, and after a pick.
 * Empty queries clear results (no debounced phantom request).
 */
export function AddressAutocomplete({
  country,
  onSelect,
  placeholder = "Commencez à taper votre adresse…",
  disabled = false,
}: AddressAutocompleteProps) {
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<BANFeature[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement | null>(null)

  const enabled = !disabled && country === "FR"

  // Close the dropdown on outside click — kept inline so the
  // component stays self-contained, no `useClickOutside` dependency.
  useEffect(() => {
    function handleOutside(e: MouseEvent) {
      if (!containerRef.current) return
      if (!containerRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener("mousedown", handleOutside)
      return () => document.removeEventListener("mousedown", handleOutside)
    }
  }, [isOpen])

  // Debounced search. 250ms is the sweet spot for "feels live" while
  // staying respectful of the public BAN API rate limits.
  useEffect(() => {
    if (!enabled) {
      setResults([])
      return
    }
    const trimmed = query.trim()
    if (trimmed.length < 3) {
      setResults([])
      return
    }
    const ctl = new AbortController()
    const timer = setTimeout(async () => {
      setIsLoading(true)
      try {
        const url = `${BAN_ENDPOINT}?q=${encodeURIComponent(trimmed)}&limit=6&autocomplete=1`
        const res = await fetch(url, { signal: ctl.signal })
        if (!res.ok) {
          setResults([])
          return
        }
        const json = (await res.json()) as { features?: BANFeature[] }
        setResults(json.features ?? [])
      } catch (err) {
        if (!(err instanceof DOMException) || err.name !== "AbortError") {
          setResults([])
        }
      } finally {
        setIsLoading(false)
      }
    }, 250)
    return () => {
      ctl.abort()
      clearTimeout(timer)
    }
  }, [query, enabled])

  function handlePick(feature: BANFeature) {
    onSelect({
      line1: feature.properties.name,
      postalCode: feature.properties.postcode,
      city: feature.properties.city,
    })
    setQuery("")
    setResults([])
    setIsOpen(false)
  }

  if (!enabled) {
    // Non-FR or explicitly disabled — render a static hint rather
    // than a dead input so users know to fill the fields below.
    return (
      <div className="flex items-start gap-2 rounded-lg border border-dashed border-slate-200 bg-slate-50 p-3 text-xs text-slate-500 dark:border-slate-700 dark:bg-slate-800/40 dark:text-slate-400">
        <MapPin className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        <p>
          La saisie automatique d&apos;adresse n&apos;est disponible que pour
          les adresses françaises. Remplis les champs ci-dessous manuellement.
        </p>
      </div>
    )
  }

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          type="text"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value)
            setIsOpen(true)
          }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={(e) => {
            if (e.key === "Escape") setIsOpen(false)
          }}
          placeholder={placeholder}
          aria-label="Rechercher une adresse"
          autoComplete="off"
          className={cn(
            "h-10 w-full rounded-lg border border-slate-200 bg-white pl-9 pr-9 text-sm",
            "shadow-xs transition-colors duration-200 placeholder:text-slate-400",
            "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            "dark:border-slate-700 dark:bg-slate-900 dark:text-white",
          )}
        />
        {isLoading && (
          <Loader2
            className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-slate-400"
            aria-hidden="true"
          />
        )}
      </div>

      {isOpen && results.length > 0 && (
        <ul
          role="listbox"
          className={cn(
            "absolute z-20 mt-1 max-h-72 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 shadow-lg",
            "dark:border-slate-700 dark:bg-slate-800",
          )}
        >
          {results.map((feature, idx) => (
            <li
              key={`${feature.properties.label}-${idx}`}
              role="option"
              aria-selected={false}
              className="cursor-pointer px-3 py-2 hover:bg-slate-50 dark:hover:bg-slate-700"
              onMouseDown={(e) => {
                // mousedown so the click fires before the input's
                // blur tears down the dropdown via outside-click.
                e.preventDefault()
                handlePick(feature)
              }}
            >
              <p className="text-sm font-medium text-slate-900 dark:text-white">
                {feature.properties.name}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {feature.properties.postcode} {feature.properties.city}
              </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
