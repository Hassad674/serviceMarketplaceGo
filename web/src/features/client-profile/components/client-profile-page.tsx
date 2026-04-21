"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useProjectHistory } from "@/shared/hooks/profile/use-project-history"
import { ApiError } from "@/shared/lib/api-client"
import { ClientProfileHeader } from "./client-profile-header"
import { ClientProfileEditor } from "./client-profile-editor"
import { ClientProfileDescription } from "./client-profile-description"
import { ClientProjectHistory } from "./client-project-history"
import { ClientReviewsList } from "./client-reviews-list"
import { useMyClientProfile } from "../hooks/use-my-client-profile"
import { useUpdateClientProfile } from "../hooks/use-update-client-profile"
import type { ClientProjectHistoryEntry } from "../api/client-profile-api"

// ClientProfilePage is the editable private `/client-profile` view.
// Providers with the `org_client_profile.edit` permission see the
// editor form; others get a read-only view with a permission banner.
// `provider_personal` orgs should never reach this page — the sidebar
// hides the entry and the component renders a localized 404 state as
// a belt-and-braces fallback.
export function ClientProfilePage() {
  const t = useTranslations("clientProfile")
  const { data: org } = useOrganization()
  const { data: profile, isLoading, isError } = useMyClientProfile()
  const projectHistory = useProjectHistory(org?.id)
  const canEdit = useHasPermission("org_client_profile.edit")
  const updateMutation = useUpdateClientProfile()
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [saveSuccess, setSaveSuccess] = useState(false)

  if (!org || org.type === "provider_personal") {
    return <NotFoundState />
  }

  if (isLoading) return <PageSkeleton />
  if (isError || !profile) return <ErrorState />

  const companyName = profile.company_name
  const description = profile.client_description

  async function handleSave(values: {
    company_name?: string
    client_description?: string
  }) {
    setSubmitError(null)
    setSaveSuccess(false)
    try {
      await updateMutation.mutateAsync(values)
      setSaveSuccess(true)
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setSubmitError(t("permissionDeniedBody"))
      } else {
        setSubmitError(t("saveError"))
      }
    }
  }

  // Private page does not expose a public `project_history[]` payload
  // — we reuse the shared /project-history endpoint and project it
  // into the client-side shape. The owned provider data doesn't carry
  // counterparty info on that endpoint, so we render an empty
  // provider block rather than a fake name.
  const historyEntries = mapOwnedHistoryToClientEntries(
    projectHistory.data?.data ?? [],
  )

  return (
    <main className="mx-auto max-w-5xl space-y-6 px-4 py-8" aria-label={t("pageTitle")}>
      <ClientProfileHeader
        companyName={companyName}
        avatarUrl={profile.avatar_url}
        stats={{
          totalSpent: profile.total_spent,
          reviewCount: profile.client_review_count,
          averageRating: profile.client_avg_rating,
          projectsCompleted: profile.projects_completed_as_client,
        }}
      />

      {canEdit ? (
        <>
          <ClientProfileEditor
            initialValues={{
              company_name: companyName,
              client_description: description,
            }}
            onSubmit={handleSave}
            saving={updateMutation.isPending}
            submitError={submitError}
          />
          {saveSuccess ? (
            <p
              role="status"
              aria-live="polite"
              className="text-sm text-emerald-600 dark:text-emerald-400"
            >
              {t("saveSuccess")}
            </p>
          ) : null}
        </>
      ) : (
        <>
          <PermissionBanner />
          <ClientProfileDescription description={description} />
        </>
      )}

      <ClientProjectHistory entries={historyEntries} />

      <ClientReviewsList reviews={[]} />
    </main>
  )
}

function PermissionBanner() {
  const t = useTranslations("clientProfile")
  return (
    <div
      role="alert"
      className="rounded-2xl border border-amber-300/60 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200"
    >
      <p className="font-medium">{t("permissionDeniedTitle")}</p>
      <p className="mt-1 text-xs">{t("permissionDeniedBody")}</p>
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

function ErrorState() {
  const t = useTranslations("clientProfile")
  return (
    <div
      role="alert"
      className="mx-auto mt-8 max-w-xl rounded-xl border border-destructive/30 bg-destructive/5 p-6 text-center text-sm text-destructive"
    >
      {t("saveError")}
    </div>
  )
}

function PageSkeleton() {
  const t = useTranslations("clientProfile")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loading")}
      className="mx-auto max-w-5xl space-y-6 px-4 py-8"
    >
      <div className="h-32 rounded-2xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-64 rounded-2xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-40 rounded-2xl border border-border bg-muted/40 animate-shimmer" />
    </div>
  )
}

// mapOwnedHistoryToClientEntries bridges the shared project-history
// endpoint (counterparty-agnostic) into the shape the client history
// card expects. We drop the counterparty — the private page shows the
// current user's own paid missions, which is sufficient context for
// the owner. The public /clients/[id] page gets a richer payload with
// the provider counterparty embedded by the backend.
function mapOwnedHistoryToClientEntries(
  entries: {
    proposal_id: string
    title: string
    amount: number
    currency: string
    completed_at: string
  }[],
): ClientProjectHistoryEntry[] {
  return entries.map((entry) => ({
    proposal_id: entry.proposal_id,
    title: entry.title,
    amount: entry.amount,
    completed_at: entry.completed_at,
    provider: {
      organization_id: "",
      display_name: "—",
      avatar_url: null,
    },
  }))
}
