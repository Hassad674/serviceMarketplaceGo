"use client"

import { useEffect, useRef, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Building2, Check, Loader2, MessageCircle, X } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import { listConversations } from "@/features/messaging/api/messaging-api"
import type { Conversation } from "@/features/messaging/types"

export interface ClientPickerSelection {
  userId: string
  orgId: string
  name: string
}

interface ClientPickerProps {
  value: ClientPickerSelection | null
  onChange: (value: ClientPickerSelection | null) => void
  label?: string
  placeholder?: string
}

// ClientPicker is the enterprise-side counterpart to ProviderPicker with a
// strict rule: the apporteur can only pick a client they already have a
// conversation with. Cold-introducing a stranger is NOT supported — it
// would put the apporteur in a weak position ("who are you?") and bypass
// the warm-relationship premise of business referrals.
//
// Implementation: list the apporteur's conversations, keep only those
// whose other_org_type = 'enterprise', and render them as pickable rows.
export function ClientPicker({
  value,
  onChange,
  label = "Client",
  placeholder = "Choisir depuis une conversation…",
}: ClientPickerProps) {
  const [open, setOpen] = useState(false)
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

  const conversations = useQuery({
    queryKey: ["messaging", "conversations", "clientPicker"],
    queryFn: () => listConversations(),
    staleTime: 60 * 1000,
    enabled: open,
  })

  const all: Conversation[] = conversations.data?.data ?? []
  // Filter to ONLY enterprises. The apporteur is a provider, their clients
  // are necessarily enterprise orgs (or agencies acting as clients — but
  // V1 keeps it simple to enterprise).
  const enterprises = all.filter((c) => c.other_org_type === "enterprise")

  function select(c: Conversation) {
    onChange({
      userId: c.other_user_id,
      orgId: c.other_org_id,
      name: c.other_org_name,
    })
    setOpen(false)
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
            <Building2 className="h-4 w-4 text-rose-500" aria-hidden="true" />
            <span className="truncate">{value.name}</span>
            <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
              Entreprise
            </span>
          </span>
        ) : (
          <span className="flex items-center gap-2 text-slate-500">
            <MessageCircle className="h-4 w-4" aria-hidden="true" />
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
          <div className="border-b border-slate-100 px-4 py-2.5 text-xs text-slate-500">
            Vous ne pouvez introduire qu&rsquo;un client avec qui vous avez déjà une conversation.
          </div>
          <div className="max-h-64 overflow-y-auto">
            {conversations.isLoading ? (
              <div className="flex items-center justify-center gap-2 p-6 text-sm text-slate-500">
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                Chargement…
              </div>
            ) : enterprises.length === 0 ? (
              <div className="p-6 text-center text-sm text-slate-500">
                Aucune conversation avec un client.
                <br />
                Commencez par échanger avec un prospect avant de le présenter.
              </div>
            ) : (
              <ul>
                {enterprises.map((c) => {
                  const selected = value?.orgId === c.other_org_id
                  return (
                    <li key={c.id}>
                      <button
                        type="button"
                        onClick={() => select(c)}
                        className={cn(
                          "flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left text-sm transition hover:bg-rose-50",
                          selected && "bg-rose-50",
                        )}
                      >
                        <div className="flex items-center gap-3">
                          <div className="grid h-9 w-9 place-items-center rounded-full bg-blue-100 text-xs font-semibold text-blue-700">
                            {c.other_org_name.slice(0, 1).toUpperCase()}
                          </div>
                          <div>
                            <div className="font-medium text-slate-900">
                              {c.other_org_name}
                            </div>
                            {c.last_message && (
                              <div className="truncate text-xs text-slate-500">
                                {c.last_message}
                              </div>
                            )}
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
