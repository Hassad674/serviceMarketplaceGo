"use client"

import { FileText } from "lucide-react"
import { useTranslations } from "next-intl"
import { ProjectHistoryCard } from "@/shared/components/profile/project-history-card"
import type { ClientProjectHistoryEntry } from "../api/client-profile-api"

interface ClientProjectHistorySectionProps {
  entries: ClientProjectHistoryEntry[]
  // On the public surface we hide the section entirely when there is
  // nothing to show (clean profile for a first-time client). On the
  // private surface the section stays visible with an empty-state
  // coach mark so the owner understands where future history will
  // appear.
  readOnly?: boolean
}

// ClientProjectHistorySection renders the client-facing counterpart
// of shared/components/profile/ProjectHistorySection. Data comes
// pre-fetched from the /api/v1/clients/{orgId} payload — we do NOT
// hit the generic /profiles/{orgId}/project-history endpoint here
// because that one returns the PROVIDER side of the org's history
// (missions delivered), whereas the client profile needs the
// symmetric list of missions the org paid for. Same visual as the
// provider section via the shared ProjectHistoryCard, so reviews
// show up identically on both surfaces for the same proposal.
export function ClientProjectHistorySection(
  props: ClientProjectHistorySectionProps,
) {
  const { entries, readOnly = false } = props
  const t = useTranslations("profile")
  const count = entries.length

  // Public surface: hide the card entirely on a zero-history client
  // so the page does not surface a conspicuous empty block to
  // strangers browsing the profile.
  if (readOnly && count === 0) {
    return null
  }

  return (
    <section className="bg-card border border-border rounded-xl p-4 shadow-sm sm:p-6">
      <Header count={count} />
      {count === 0 ? (
        <EmptyState />
      ) : (
        <ul className="space-y-4">
          {entries.map((entry) => (
            <li key={entry.proposal_id}>
              <ProjectHistoryCard entry={entry} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

function Header({ count }: { count: number }) {
  const t = useTranslations("profile")
  return (
    <div className="flex items-center gap-3 mb-4">
      <h2 className="text-base font-semibold text-foreground sm:text-lg">
        {t("projectHistory")}
      </h2>
      {count > 0 ? (
        <span className="rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium">
          {t("completedCount", { count })}
        </span>
      ) : null}
    </div>
  )
}

function EmptyState() {
  const t = useTranslations("profile")
  return (
    <div className="flex flex-col items-center justify-center py-10 text-center">
      <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3">
        <FileText
          className="w-6 h-6 text-muted-foreground"
          aria-hidden="true"
        />
      </div>
      <p className="text-base font-medium text-foreground mb-1">
        {t("noProjects")}
      </p>
      <p className="text-sm text-muted-foreground italic">
        {t("projectsAppearHere")}
      </p>
    </div>
  )
}
