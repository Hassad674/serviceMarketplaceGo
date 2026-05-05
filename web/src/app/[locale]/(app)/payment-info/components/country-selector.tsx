"use client"

import { useEffect, useMemo, useRef, useState } from "react"
import { Check, ChevronDown, Search } from "lucide-react"
import { useTranslations } from "next-intl"

import {
  REGION_LABELS,
  STRIPE_CONNECT_COUNTRIES,
  searchCountries,
  type SupportedCountry,
} from "@/shared/lib/stripe-countries"

import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
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
      <Button
        variant="ghost"
        size="auto"
        type="button"
        disabled={disabled}
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={cn(
          "flex h-14 w-full items-center justify-between rounded-2xl border bg-card px-4 transition-all",
          disabled
            ? "cursor-not-allowed border-border opacity-60"
            : open
              ? "border-primary ring-4 ring-primary/15 shadow-card"
              : "border-border-strong hover:border-primary/60",
        )}
      >
        {selected ? (
          <span className="flex items-center gap-3">
            <span className="text-2xl leading-none" aria-hidden>
              {selected.flag}
            </span>
            <span className="flex flex-col items-start">
              <span className="text-[12px] font-medium text-muted-foreground">
                {t("countryLabel")}
              </span>
              <span className="text-[15px] font-semibold text-foreground">
                {selected.labelEn}
              </span>
            </span>
          </span>
        ) : (
          <span className="flex flex-col items-start">
            <span className="text-[12px] font-medium text-muted-foreground">
              {t("countryLabel")}
            </span>
            <span className="text-[15px] text-subtle-foreground">
              {t("countryPlaceholder")}
            </span>
          </span>
        )}
        <ChevronDown
          className={cn(
            "h-5 w-5 text-subtle-foreground transition-transform",
            open && "rotate-180 text-primary",
          )}
          aria-hidden
        />
      </Button>

      {open ? (
        <div className="absolute left-0 right-0 top-[calc(100%+8px)] z-50 overflow-hidden rounded-2xl border border-border bg-card shadow-card-strong animate-scale-in">
          <div className="border-b border-border p-3">
            <div className="flex items-center gap-2 rounded-xl bg-background px-3 py-2 ring-1 ring-border">
              <Search className="h-4 w-4 shrink-0 text-subtle-foreground" aria-hidden />
              <Input
                ref={searchRef}
                type="text"
                placeholder={t("searchCountry")}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  setHighlightIndex(0)
                }}
                onKeyDown={handleKeyDown}
                className="w-full border-0 bg-transparent p-0 text-sm text-foreground outline-none placeholder:text-subtle-foreground focus-visible:ring-0"
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
              <li className="px-4 py-6 text-center text-sm text-muted-foreground">
                {t("noCountryFound")}
              </li>
            ) : (
              grouped.map((group) => (
                <li key={group.region}>
                  <div className="sticky top-0 bg-card/95 px-4 py-1.5 font-mono text-[10px] font-semibold uppercase tracking-[0.12em] text-subtle-foreground backdrop-blur">
                    {REGION_LABELS[group.region]}
                  </div>
                  <ul>
                    {group.items.map((country) => {
                      const globalIndex = filtered.indexOf(country)
                      const isHighlighted = globalIndex === highlightIndex
                      const isSelected = value === country.code
                      return (
                        <li key={country.code}>
                          <Button
                            variant="ghost"
                            size="auto"
                            type="button"
                            onClick={() => {
                              onChange(country.code)
                              setOpen(false)
                              setQuery("")
                            }}
                            onMouseEnter={() => setHighlightIndex(globalIndex)}
                            className={cn(
                              "flex w-full items-center justify-between px-4 py-2.5 text-left text-sm transition-colors",
                              isHighlighted && "bg-primary-soft",
                            )}
                            role="option"
                            aria-selected={isSelected}
                          >
                            <span className="flex items-center gap-3">
                              <span className="text-xl leading-none" aria-hidden>
                                {country.flag}
                              </span>
                              <span className="font-medium text-foreground">
                                {country.labelEn}
                              </span>
                              <span className="font-mono text-[11px] text-subtle-foreground">
                                {country.code}
                              </span>
                            </span>
                            {isSelected ? (
                              <Check className="h-4 w-4 text-primary" aria-hidden />
                            ) : null}
                          </Button>
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
