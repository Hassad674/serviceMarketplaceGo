"use client"

import Image from "next/image"
import { CheckCircle2, Clock } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { formatCurrency, formatDate } from "@/shared/lib/utils"
import type { ClientProjectHistoryEntry } from "../api/client-profile-api"

interface ClientProjectHistoryProps {
  entries: ClientProjectHistoryEntry[]
}

// ClientProjectHistory renders the list of missions this client has
// paid for, in reverse chronological order. Each row links to the
// hired provider's public profile (agency vs freelance routing is the
// caller's problem — here we always link to /freelancers because the
// backend currently only exposes freelance provider orgs as
// completed-project counterparties; agency links will be added when
// the mission system starts returning agency provider types).
export function ClientProjectHistory(props: ClientProjectHistoryProps) {
  const { entries } = props
  const t = useTranslations("clientProfile")

  return (
    <section
      className="bg-card border border-border rounded-2xl p-6 shadow-sm"
      aria-labelledby="client-project-history-heading"
    >
      <div className="flex items-center gap-3">
        <h2
          id="client-project-history-heading"
          className="text-lg font-semibold text-foreground"
        >
          {t("projectHistoryTitle")}
        </h2>
        {entries.length > 0 ? (
          <span className="rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium">
            {entries.length}
          </span>
        ) : null}
      </div>

      {entries.length === 0 ? (
        <EmptyState />
      ) : (
        <ul className="mt-4 space-y-3">
          {entries.map((entry) => (
            <li key={entry.proposal_id}>
              <HistoryRow entry={entry} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

function EmptyState() {
  const t = useTranslations("clientProfile")
  return (
    <div className="mt-4 flex flex-col items-center gap-2 rounded-xl border border-dashed border-border bg-muted/30 px-6 py-8 text-center">
      <Clock className="h-8 w-8 text-muted-foreground" aria-hidden="true" />
      <p className="text-sm text-muted-foreground">{t("projectHistoryEmpty")}</p>
    </div>
  )
}

interface HistoryRowProps {
  entry: ClientProjectHistoryEntry
}

function HistoryRow({ entry }: HistoryRowProps) {
  const t = useTranslations("clientProfile")
  return (
    <article className="flex items-start gap-4 rounded-xl border border-border/60 bg-background p-4 transition-colors duration-150 hover:border-border">
      <CheckCircle2
        className="mt-0.5 h-5 w-5 shrink-0 text-emerald-500"
        aria-hidden="true"
      />
      <div className="min-w-0 flex-1">
        <h3 className="truncate text-sm font-medium text-foreground">
          {entry.title}
        </h3>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("completedOn", { date: formatDate(entry.completed_at) })}
        </p>
        <div className="mt-3 flex items-center gap-2">
          <Link
            href={`/freelancers/${entry.provider.organization_id}`}
            className="inline-flex items-center gap-2 rounded-lg bg-muted/50 px-2 py-1 text-xs font-medium text-foreground hover:bg-muted focus:outline-none focus:ring-2 focus:ring-rose-500 transition-colors"
          >
            <ProviderAvatar provider={entry.provider} />
            <span className="truncate max-w-[160px]">
              {entry.provider.display_name}
            </span>
          </Link>
        </div>
      </div>
      <div className="shrink-0 text-right">
        <p className="text-sm font-mono font-semibold tabular-nums text-foreground">
          {formatCurrency(entry.amount / 100)}
        </p>
      </div>
    </article>
  )
}

function ProviderAvatar({
  provider,
}: {
  provider: ClientProjectHistoryEntry["provider"]
}) {
  if (provider.avatar_url) {
    return (
      <Image
        src={provider.avatar_url}
        alt={provider.display_name}
        width={20}
        height={20}
        className="h-5 w-5 rounded-full object-cover"
      />
    )
  }
  return (
    <span
      aria-hidden="true"
      className="flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-[10px] font-semibold text-white"
    >
      {provider.display_name.charAt(0).toUpperCase() || "?"}
    </span>
  )
}
