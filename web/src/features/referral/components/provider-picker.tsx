"use client"

import { useMemo, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  Check,
  Loader2,
  MessageCircle,
  Search,
  Sparkles,
  Users,
} from "lucide-react"

import { cn } from "@/shared/lib/utils"
import { listConversations } from "@/shared/lib/messaging/conversations-api"
import type { Conversation } from "@/shared/types/messaging"
import {
  searchProfiles,
  type PublicProfileSummary,
} from "@/shared/lib/search/search-api"

import { PickerModal, PickerTrigger } from "./picker-modal"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
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
}

type Tab = "search" | "conversations"

// ProviderPicker is the trigger + modal that lets the apporteur select the
// provider party of a business referral. Two tabs:
//
//   1. Search — freelances + agencies from /api/v1/profiles/search, filtered
//      client-side by name. Good for the apporteur who already knows the
//      marketplace catalogue and wants to recommend someone they don't yet
//      have a conversation with.
//
//   2. From a conversation — the apporteur's existing conversations filtered
//      to other_org_type IN (provider_personal, agency). Matches the real
//      workflow: the apporteur starts with a warm contact and wants to
//      introduce them without re-typing anything.
//
// Enterprises are not exposed in the search tab on purpose — they go
// through the ClientPicker. The "conversations" tab only surfaces
// freelances + agencies for the same reason.
export function ProviderPicker({
  value,
  onChange,
  label = "Prestataire",
}: ProviderPickerProps) {
  const [open, setOpen] = useState(false)
  const [tab, setTab] = useState<Tab>("search")

  function select(next: ProviderPickerSelection) {
    onChange(next)
    setOpen(false)
  }

  return (
    <>
      <PickerTrigger
        label={label}
        open={open}
        onOpen={() => setOpen(true)}
        onClear={value ? () => onChange(null) : null}
      >
        {value ? (
          <>
            <Users className="h-4 w-4 text-rose-500" aria-hidden="true" />
            <span className="truncate text-slate-900">{value.name}</span>
            <span className="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
              {orgTypeLabel(value.orgType)}
            </span>
          </>
        ) : (
          <>
            <Search className="h-4 w-4 text-slate-400" aria-hidden="true" />
            <span className="text-slate-500">
              Rechercher un freelance ou une agence…
            </span>
          </>
        )}
      </PickerTrigger>

      <PickerModal
        open={open}
        onClose={() => setOpen(false)}
        title="Choisir un prestataire"
        description="Recherchez dans le catalogue ou choisissez depuis une conversation existante."
      >
        <TabBar tab={tab} onChange={setTab} />
        {tab === "search" ? (
          <SearchTab currentValue={value} onSelect={select} />
        ) : (
          <ConversationsTab currentValue={value} onSelect={select} />
        )}
      </PickerModal>
    </>
  )
}

// ─── Tab bar ───────────────────────────────────────────────────────────────

interface TabBarProps {
  tab: Tab
  onChange: (tab: Tab) => void
}

function TabBar({ tab, onChange }: TabBarProps) {
  return (
    <div role="tablist" className="flex border-b border-slate-100 px-2">
      <TabButton
        active={tab === "search"}
        onClick={() => onChange("search")}
        icon={<Search className="h-4 w-4" aria-hidden="true" />}
      >
        Rechercher
      </TabButton>
      <TabButton
        active={tab === "conversations"}
        onClick={() => onChange("conversations")}
        icon={<MessageCircle className="h-4 w-4" aria-hidden="true" />}
      >
        Depuis une conversation
      </TabButton>
    </div>
  )
}

interface TabButtonProps {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  children: React.ReactNode
}

function TabButton({ active, onClick, icon, children }: TabButtonProps) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={cn(
        "relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition",
        active ? "text-rose-600" : "text-slate-500 hover:text-slate-700",
      )}
    >
      {icon}
      {children}
      {active && (
        <span
          aria-hidden="true"
          className="absolute inset-x-2 bottom-0 h-0.5 rounded-full bg-rose-500"
        />
      )}
    </Button>
  )
}

// ─── Search tab ────────────────────────────────────────────────────────────

interface SearchTabProps {
  currentValue: ProviderPickerSelection | null
  onSelect: (value: ProviderPickerSelection) => void
}

function SearchTab({ currentValue, onSelect }: SearchTabProps) {
  const [query, setQuery] = useState("")

  const freelancers = useQuery({
    queryKey: ["profiles", "search", "freelancer"],
    queryFn: () => searchProfiles("freelancer"),
    staleTime: 60 * 1000,
  })
  const agencies = useQuery({
    queryKey: ["profiles", "search", "agency"],
    queryFn: () => searchProfiles("agency"),
    staleTime: 60 * 1000,
  })

  const loading = freelancers.isLoading || agencies.isLoading
  const all = useMemo<PublicProfileSummary[]>(
    () => [
      ...(freelancers.data?.data ?? []),
      ...(agencies.data?.data ?? []),
    ],
    [freelancers.data, agencies.data],
  )

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return all
    return all.filter((p) => p.name.toLowerCase().includes(q))
  }, [all, query])

  return (
    <div className="flex flex-1 flex-col">
      <div className="border-b border-slate-100 p-3">
        <div className="flex items-center gap-2 rounded-md bg-slate-50 px-3 py-2">
          <Search className="h-4 w-4 text-slate-400" aria-hidden="true" />
          <Input
            type="text"
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filtrer par nom…"
            className="flex-1 bg-transparent text-sm focus:outline-none"
          />
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <LoadingRow />
        ) : filtered.length === 0 ? (
          <EmptyRow message="Aucun résultat pour cette recherche." />
        ) : (
          <ul>
            {filtered.slice(0, 50).map((p) => {
              const selected = currentValue?.orgId === p.organization_id
              return (
                <li key={p.organization_id}>
                  <Button variant="ghost" size="auto"
                    type="button"
                    onClick={() =>
                      onSelect({
                        userId: p.owner_user_id,
                        orgId: p.organization_id,
                        name: p.name,
                        orgType: p.org_type,
                      })
                    }
                    className={cn(
                      "flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left text-sm transition hover:bg-rose-50",
                      selected && "bg-rose-50",
                    )}
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <Avatar name={p.name} />
                      <div className="min-w-0">
                        <div className="truncate font-medium text-slate-900">
                          {p.name}
                        </div>
                        <div className="truncate text-xs text-slate-500">
                          {orgTypeLabel(p.org_type)}
                          {p.title ? ` · ${p.title}` : null}
                        </div>
                      </div>
                    </div>
                    {selected && (
                      <Check
                        className="h-4 w-4 shrink-0 text-rose-500"
                        aria-hidden="true"
                      />
                    )}
                  </Button>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

// ─── Conversations tab ────────────────────────────────────────────────────

interface ConversationsTabProps {
  currentValue: ProviderPickerSelection | null
  onSelect: (value: ProviderPickerSelection) => void
}

function ConversationsTab({ currentValue, onSelect }: ConversationsTabProps) {
  const { data, isLoading } = useQuery({
    queryKey: ["messaging", "conversations", "providerPicker"],
    queryFn: () => listConversations(),
    staleTime: 60 * 1000,
  })

  const providers = useMemo<Conversation[]>(() => {
    const all = data?.data ?? []
    return all.filter(
      (c) =>
        c.other_org_type === "provider_personal" ||
        c.other_org_type === "agency",
    )
  }, [data])

  return (
    <div className="flex flex-1 flex-col">
      <div className="border-b border-slate-100 px-4 py-3 text-xs text-slate-500">
        <Sparkles
          className="mr-1 inline h-3.5 w-3.5 text-rose-400"
          aria-hidden="true"
        />
        Prestataires avec qui vous avez déjà une conversation.
      </div>
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <LoadingRow />
        ) : providers.length === 0 ? (
          <EmptyRow message="Aucune conversation avec un freelance ou une agence. Passez par l'onglet Rechercher pour choisir un prestataire du catalogue." />
        ) : (
          <ul>
            {providers.map((c) => {
              const selected = currentValue?.orgId === c.other_org_id
              return (
                <li key={c.id}>
                  <Button variant="ghost" size="auto"
                    type="button"
                    onClick={() =>
                      onSelect({
                        userId: c.other_user_id,
                        orgId: c.other_org_id,
                        name: c.other_org_name,
                        orgType: c.other_org_type,
                      })
                    }
                    className={cn(
                      "flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left text-sm transition hover:bg-rose-50",
                      selected && "bg-rose-50",
                    )}
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <Avatar name={c.other_org_name} />
                      <div className="min-w-0">
                        <div className="truncate font-medium text-slate-900">
                          {c.other_org_name}
                        </div>
                        <div className="truncate text-xs text-slate-500">
                          {orgTypeLabel(c.other_org_type)}
                          {c.last_message ? ` · ${c.last_message}` : null}
                        </div>
                      </div>
                    </div>
                    {selected && (
                      <Check
                        className="h-4 w-4 shrink-0 text-rose-500"
                        aria-hidden="true"
                      />
                    )}
                  </Button>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

// ─── Small helpers ────────────────────────────────────────────────────────

function Avatar({ name }: { name: string }) {
  return (
    <div className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-rose-100 text-xs font-semibold text-rose-700">
      {name.slice(0, 1).toUpperCase()}
    </div>
  )
}

function LoadingRow() {
  return (
    <div className="flex items-center justify-center gap-2 p-8 text-sm text-slate-500">
      <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
      Chargement…
    </div>
  )
}

function EmptyRow({ message }: { message: string }) {
  return <div className="p-8 text-center text-sm text-slate-500">{message}</div>
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
