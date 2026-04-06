"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import { Check, ChevronDown, Search } from "lucide-react"
import { useTranslations } from "next-intl"

import {
  REGION_LABELS,
  STRIPE_CONNECT_COUNTRIES,
  searchCountries,
  type SupportedCountry,
} from "../lib/countries"

type CountrySelectorProps = {
  value: string | null
  onChange: (code: string) => void
  disabled?: boolean
}

export function CountrySelector({ value, onChange, disabled }: CountrySelectorProps) {
  const t = useTranslations("paymentInfo")
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [highlightIndex, setHighlightIndex] = useState(0)
  const rootRef = useRef<HTMLDivElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)

  const filtered = useMemo(() => searchCountries(query, "en"), [query])
  const grouped = useMemo(() => groupByRegion(filtered), [filtered])
  const selected = useMemo(
    () => (value ? STRIPE_CONNECT_COUNTRIES.find((c) => c.code === value) : null),
    [value],
  )

  useEffect(() => {
    if (open && searchRef.current) {
      searchRef.current.focus()
    }
  }, [open])

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener("mousedown", onClickOutside)
    return () => document.removeEventListener("mousedown", onClickOutside)
  }, [open])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setOpen(false)
      return
    }
    if (e.key === "ArrowDown") {
      e.preventDefault()
      setHighlightIndex((i) => Math.min(i + 1, filtered.length - 1))
    }
    if (e.key === "ArrowUp") {
      e.preventDefault()
      setHighlightIndex((i) => Math.max(i - 1, 0))
    }
    if (e.key === "Enter" && filtered[highlightIndex]) {
      e.preventDefault()
      onChange(filtered[highlightIndex].code)
      setOpen(false)
      setQuery("")
    }
  }

  return (
    <div ref={rootRef} className="relative w-full">
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={`flex h-14 w-full items-center justify-between rounded-xl border-2 bg-white px-4 shadow-sm transition-all ${
          disabled
            ? "cursor-not-allowed border-slate-200 opacity-60"
            : open
              ? "border-rose-500 ring-4 ring-rose-500/10"
              : "border-slate-200 hover:border-slate-300"
        }`}
      >
        {selected ? (
          <span className="flex items-center gap-3">
            <span className="text-2xl leading-none" aria-hidden>
              {selected.flag}
            </span>
            <span className="flex flex-col items-start">
              <span className="text-[13px] font-medium text-slate-500">{t("countryLabel")}</span>
              <span className="text-[15px] font-semibold text-slate-900">{selected.labelEn}</span>
            </span>
          </span>
        ) : (
          <span className="flex flex-col items-start">
            <span className="text-[13px] font-medium text-slate-500">{t("countryLabel")}</span>
            <span className="text-[15px] text-slate-400">{t("countryPlaceholder")}</span>
          </span>
        )}
        <ChevronDown
          className={`h-5 w-5 text-slate-400 transition-transform ${open ? "rotate-180" : ""}`}
          aria-hidden
        />
      </button>

      {open ? (
        <div className="absolute left-0 right-0 top-[calc(100%+8px)] z-50 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl animate-scale-in">
          <div className="border-b border-slate-100 p-3">
            <div className="flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2">
              <Search className="h-4 w-4 shrink-0 text-slate-400" aria-hidden />
              <input
                ref={searchRef}
                type="text"
                placeholder={t("searchCountry")}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  setHighlightIndex(0)
                }}
                onKeyDown={handleKeyDown}
                className="w-full bg-transparent text-sm outline-none placeholder:text-slate-400"
                aria-label={t("searchCountry")}
              />
            </div>
          </div>

          <ul
            role="listbox"
            aria-label={t("countryLabel")}
            className="max-h-[340px] overflow-y-auto py-1"
          >
            {grouped.length === 0 ? (
              <li className="px-4 py-6 text-center text-sm text-slate-400">
                {t("noCountryFound")}
              </li>
            ) : (
              grouped.map((group) => (
                <li key={group.region}>
                  <div className="sticky top-0 bg-white/95 px-4 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-400 backdrop-blur">
                    {REGION_LABELS[group.region]}
                  </div>
                  <ul>
                    {group.items.map((country) => {
                      const globalIndex = filtered.indexOf(country)
                      const isHighlighted = globalIndex === highlightIndex
                      const isSelected = value === country.code
                      return (
                        <li key={country.code}>
                          <button
                            type="button"
                            onClick={() => {
                              onChange(country.code)
                              setOpen(false)
                              setQuery("")
                            }}
                            onMouseEnter={() => setHighlightIndex(globalIndex)}
                            className={`flex w-full items-center justify-between px-4 py-2.5 text-left text-sm transition-colors ${
                              isHighlighted ? "bg-rose-50" : ""
                            }`}
                            role="option"
                            aria-selected={isSelected}
                          >
                            <span className="flex items-center gap-3">
                              <span className="text-xl leading-none" aria-hidden>
                                {country.flag}
                              </span>
                              <span className="font-medium text-slate-900">{country.labelEn}</span>
                              <span className="font-mono text-[11px] text-slate-400">
                                {country.code}
                              </span>
                            </span>
                            {isSelected ? (
                              <Check className="h-4 w-4 text-rose-500" aria-hidden />
                            ) : null}
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                </li>
              ))
            )}
          </ul>
        </div>
      ) : null}
    </div>
  )
}

function groupByRegion(countries: SupportedCountry[]) {
  const map = new Map<SupportedCountry["region"], SupportedCountry[]>()
  for (const c of countries) {
    if (!map.has(c.region)) map.set(c.region, [])
    map.get(c.region)!.push(c)
  }
  const order: SupportedCountry["region"][] = [
    "eu",
    "europe_other",
    "americas",
    "apac",
    "mena",
  ]
  return order
    .filter((r) => map.has(r))
    .map((r) => ({ region: r, items: map.get(r)! }))
}
