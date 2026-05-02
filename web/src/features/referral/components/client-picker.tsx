"use client"

import { useMemo, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Building2, Check, Loader2, MessageCircle } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import { listConversations } from "@/shared/lib/messaging/conversations-api"
import type { Conversation } from "@/shared/types/messaging"

import { PickerModal, PickerTrigger } from "./picker-modal"
import { Button } from "@/shared/components/ui/button"

export interface ClientPickerSelection {
  userId: string
  orgId: string
  name: string
}

interface ClientPickerProps {
  value: ClientPickerSelection | null
  onChange: (value: ClientPickerSelection | null) => void
  label?: string
}

// ClientPicker is the enterprise-side counterpart to ProviderPicker with a
// strict rule: the apporteur can only pick a client they already have a
// conversation with. Cold-introducing a stranger is NOT supported — it
// would put the apporteur in a weak position ("who are you?") and bypass
// the warm-relationship premise of business referrals.
//
// Renders the same trigger + modal shell as ProviderPicker but with a
// single scrollable list (no tabs) because there is only one way to pick
// a client.
export function ClientPicker({
  value,
  onChange,
  label = "Client",
}: ClientPickerProps) {
  const [open, setOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ["messaging", "conversations", "clientPicker"],
    queryFn: () => listConversations(),
    staleTime: 60 * 1000,
    enabled: open,
  })

  const enterprises = useMemo<Conversation[]>(() => {
    const all = data?.data ?? []
    return all.filter((c) => c.other_org_type === "enterprise")
  }, [data])

  function select(c: Conversation) {
    onChange({
      userId: c.other_user_id,
      orgId: c.other_org_id,
      name: c.other_org_name,
    })
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
            <Building2 className="h-4 w-4 text-rose-500" aria-hidden="true" />
            <span className="truncate text-slate-900">{value.name}</span>
            <span className="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
              Entreprise
            </span>
          </>
        ) : (
          <>
            <MessageCircle className="h-4 w-4 text-slate-400" aria-hidden="true" />
            <span className="text-slate-500">
              Choisir depuis une conversation…
            </span>
          </>
        )}
      </PickerTrigger>

      <PickerModal
        open={open}
        onClose={() => setOpen(false)}
        title="Choisir un client"
        description="Uniquement les clients avec qui vous avez déjà une conversation."
      >
        <div className="flex flex-1 flex-col">
          <div className="border-b border-slate-100 px-4 py-3 text-xs text-slate-500">
            Vous ne pouvez introduire qu&rsquo;un client avec qui vous avez
            déjà échangé. C&rsquo;est la base d&rsquo;un apport d&rsquo;affaires :
            une relation chaude, pas un contact froid.
          </div>
          <div className="flex-1 overflow-y-auto">
            {isLoading ? (
              <div className="flex items-center justify-center gap-2 p-8 text-sm text-slate-500">
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                Chargement…
              </div>
            ) : enterprises.length === 0 ? (
              <div className="p-8 text-center text-sm text-slate-500">
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
                      <Button variant="ghost" size="auto"
                        type="button"
                        onClick={() => select(c)}
                        className={cn(
                          "flex w-full items-center justify-between gap-3 px-4 py-2.5 text-left text-sm transition hover:bg-rose-50",
                          selected && "bg-rose-50",
                        )}
                      >
                        <div className="flex min-w-0 items-center gap-3">
                          <div className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-blue-100 text-xs font-semibold text-blue-700">
                            {c.other_org_name.slice(0, 1).toUpperCase()}
                          </div>
                          <div className="min-w-0">
                            <div className="truncate font-medium text-slate-900">
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
      </PickerModal>
    </>
  )
}
