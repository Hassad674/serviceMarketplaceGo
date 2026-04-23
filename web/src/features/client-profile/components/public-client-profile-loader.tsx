"use client"

import { useTranslations } from "next-intl"
import { usePublicClientProfile } from "../hooks/use-public-client-profile"
import { ClientProfileHeader } from "./client-profile-header"
import { ClientProfileDescription } from "./client-profile-description"
import { ClientProjectHistorySection } from "./client-project-history-section"

interface PublicClientProfileLoaderProps {
  orgId: string
}

// PublicClientProfileLoader is the client-side renderer for the
// public /clients/[id] page. It owns the loading / 404 / error
// states and delegates actual presentation to the stateless
// client-profile components so they stay reusable between the
// private editor and this read-only surface.
export function PublicClientProfileLoader(
  props: PublicClientProfileLoaderProps,
) {
  const { orgId } = props
  const t = useTranslations("clientProfile")
  const { data, isLoading, isError } = usePublicClientProfile(orgId)

  if (isLoading) return <Skeleton label={t("loading")} />
  if (isError || !data) return <NotFoundState />

  return (
    <div className="mx-auto max-w-5xl space-y-6 px-4 py-8">
      <ClientProfileHeader
        companyName={data.company_name}
        avatarUrl={data.avatar_url}
        stats={{
          totalSpent: data.total_spent,
          reviewCount: data.review_count,
          averageRating: data.average_rating,
          projectsCompleted: data.projects_completed_as_client,
        }}
      />
      <ClientProfileDescription description={data.client_description} />
      {/* Unified project history — one card per completed mission
          with the provider→client review embedded, or an "awaiting
          review" placeholder when none was submitted yet. Entries
          come from the /api/v1/clients/{orgId} response directly
          (the generic /profiles/{orgId}/project-history is the
          PROVIDER side and would be the wrong data here). */}
      <ClientProjectHistorySection
        entries={data.project_history}
        readOnly
      />
    </div>
  )
}

function NotFoundState() {
  const t = useTranslations("clientProfile")
  return (
    <main className="mx-auto flex max-w-xl flex-col items-center gap-2 px-4 py-16 text-center">
      <h1 className="text-2xl font-semibold text-foreground">
        {t("notFoundTitle")}
      </h1>
      <p className="text-sm text-muted-foreground">{t("notFoundBody")}</p>
    </main>
  )
}

interface SkeletonProps {
  label: string
}

function Skeleton({ label }: SkeletonProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={label}
      className="mx-auto max-w-5xl space-y-6 px-4 py-8"
    >
      <div className="h-32 rounded-2xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-40 rounded-2xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-64 rounded-2xl border border-border bg-muted/40 animate-shimmer" />
    </div>
  )
}
