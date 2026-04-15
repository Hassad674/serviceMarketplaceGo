"use client"

import { useEffect, useRef, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Check, Loader2, Search, Users, X } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import {
  searchProfiles,
  type PublicProfileSummary,
} from "@/features/provider/api/search-api"

export interface ProviderPickerSelection {
  userId: string
  orgId: string
  name: string
  orgType: string
}

interface ProviderPickerProps {
  value: ProviderPickerSelection | null
  onChange: (value: ProviderPickerSelection | null) => void
  label?: string
  placeholder?: string
}

// ProviderPicker is a searchable input with a dropdown that fetches
// freelancers + agencies and filters by name client-side.
//
// The apporteur cannot type a raw UUID anymore — they MUST pick from the
// suggestions. The button shape was chosen on purpose: clicking anywhere
// on the input opens the dropdown, closing it without a pick resets
// nothing (the previous selection is preserved).
//
// Enterprises are NOT searchable here — you should only introduce clients
// you already have a conversation with. That branch uses ClientPicker.
export function ProviderPicker({
  value,
  onChange,
  label = "Prestataire",
  placeholder = "Rechercher un freelance ou une agence…",
}: ProviderPickerProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const containerRef = useRef<HTMLDivElement>(null)

  // Close on outside click.
  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [open])

  // Fetch freelancers and agencies in parallel. 100 items per type is the
  // backend's max page size; client-side filtering on name takes over from
  // there until a proper `?q=…` query param lands on the backend.
  const freelancers = useQuery({
    queryKey: ["profiles", "search", "freelancer"],
    queryFn: () => searchProfiles("freelancer"),
    staleTime: 60 * 1000,
    enabled: open,
  })
  const agencies = useQuery({
    queryKey: ["profiles", "search", "agency"],
    queryFn: () => searchProfiles("agency"),
    staleTime: 60 * 1000,
    enabled: open,
  })

  const loading = freelancers.isLoading || agencies.isLoading
  const all: PublicProfileSummary[] = [
    ...(freelancers.data?.data ?? []),
    ...(agencies.data?.data ?? []),
  ]

  const q = query.trim().toLowerCase()
  const filtered = q
    ? all.filter((p) => p.name.toLowerCase().includes(q))
    : all

  function select(p: PublicProfileSummary) {
    onChange({
      userId: p.owner_user_id,
      orgId: p.organization_id,
      name: p.name,
      orgType: p.org_type,
    })
    setOpen(false)
    setQuery("")
  }

  function clear(e: React.MouseEvent) {
    e.stopPropagation()
    onChange(null)
  }

  return (
    <div className="relative" ref={containerRef}>
      <label className="mb-1.5 block text-sm font-medium text-slate-700">
        {label}
      </label>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={cn(
          "flex w-full items-center justify-between gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-left text-sm transition",
          "focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-100",
          open && "border-rose-500 ring-2 ring-rose-100",
        )}
      >
        {value ? (
          <span className="flex items-center gap-2 text-slate-900">
            <Users className="h-4 w-4 text-rose-500" aria-hidden="true" />
            <span className="truncate">{value.name}</span>
            <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
              {orgTypeLabel(value.orgType)}
            </span>
          </span>
        ) : (
          <span className="flex items-center gap-2 text-slate-500">
            <Search className="h-4 w-4" aria-hidden="true" />
            {placeholder}
          </span>
        )}
        {value && (
          <button
            type="button"
            onClick={clear}
            className="rounded p-0.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
            aria-label="Effacer la sélection"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        )}
      </button>

      {open && (
        <div className="absolute z-20 mt-1 w-full rounded-lg border border-slate-200 bg-white shadow-lg">
          <div className="border-b border-slate-100 p-2">
            <div className="flex items-center gap-2 rounded-md bg-slate-50 px-3 py-1.5">
              <Search className="h-4 w-4 text-slate-400" aria-hidden="true" />
              <input
                type="text"
                autoFocus
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Filtrer par nom…"
                className="flex-1 bg-transparent text-sm focus:outline-none"
              />
            </div>
          </div>
          <div className="max-h-64 overflow-y-auto">
            {loading ? (
              <div className="flex items-center justify-center gap-2 p-6 text-sm text-slate-500">
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                Chargement…
              </div>
            ) : filtered.length === 0 ? (
              <div className="p-6 text-center text-sm text-slate-500">
                Aucun résultat.
              </div>
            ) : (
              <ul>
                {filtered.slice(0, 50).map((p) => {
                  const selected = value?.orgId === p.organization_id
                  return (
                    <li key={p.organization_id}>
                      <button
                        type="button"
                        onClick={() => select(p)}
                        className={cn(
                          "flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left text-sm transition hover:bg-rose-50",
                          selected && "bg-rose-50",
                        )}
                      >
                        <div className="flex items-center gap-3">
                          <div className="grid h-9 w-9 place-items-center rounded-full bg-rose-100 text-xs font-semibold text-rose-700">
                            {p.name.slice(0, 1).toUpperCase()}
                          </div>
                          <div>
                            <div className="font-medium text-slate-900">{p.name}</div>
                            <div className="text-xs text-slate-500">
                              {orgTypeLabel(p.org_type)}
                              {p.title ? ` · ${p.title}` : null}
                            </div>
                          </div>
                        </div>
                        {selected && (
                          <Check className="h-4 w-4 text-rose-500" aria-hidden="true" />
                        )}
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function orgTypeLabel(orgType: string): string {
  switch (orgType) {
    case "provider_personal":
      return "Freelance"
    case "agency":
      return "Agence"
    case "enterprise":
      return "Entreprise"
    default:
      return orgType
  }
}
