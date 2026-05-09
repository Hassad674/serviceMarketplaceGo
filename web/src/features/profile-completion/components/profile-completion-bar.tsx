"use client"

import { ChevronRight } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"

import { useProfileCompletion } from "../hooks/use-profile-completion"
import type { CompletionPersona } from "../api/profile-completion-api"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { cn } from "@/shared/lib/utils"

// ProfileCompletionBarProps controls how the bar lays out — sidebar
// vs page header — without forking the component. Keeps the prop
// surface under the 4-cap.
type ProfileCompletionBarProps = {
  // variant changes the visual density: "sidebar" is compact (used in
  // the sidebar user card); "page" is the larger card used at the top
  // of the profile page. Default "page".
  variant?: "sidebar" | "page"
  // collapsed is forwarded by the sidebar so the bar can shrink to a
  // single percent label when the sidebar is collapsed. Has no effect
  // for the page variant.
  collapsed?: boolean
  // hideWhenComplete suppresses the bar at 100%. Default false — the
  // page variant keeps the bar visible to celebrate the milestone.
  hideWhenComplete?: boolean
  // persona scopes the report to a specific facet (used on /referral
  // for the apporteur checklist). When omitted the backend
  // auto-selects from the org type — provider_personal +
  // freelance, agency + agency, enterprise + enterprise.
  persona?: CompletionPersona
}

// profilePathFor maps the authenticated user's role + org type to the
// in-app URL of their own profile page. Hard-coded here (instead of
// reading the backend `completion_path` per section) because the bar
// now navigates straight to the profile shell — sections are edited
// in place from there. Kept feature-local to avoid a cross-feature
// import on the role-aware routing helper.
//
// The optional `persona` argument is honoured first: the apporteur
// surface lives at `/referral`, distinct from the shared `/profile`
// shell that hosts the freelance and agency editors.
function profilePathFor(
  role: string | undefined,
  orgType: string | undefined,
  persona: CompletionPersona,
): string {
  if (persona === "referrer") {
    return "/referral"
  }
  // Enterprise (client) orgs surface the dedicated client-profile shell.
  if (role === "enterprise" || orgType === "enterprise") {
    return "/client-profile"
  }
  // Provider, freelance, and agency all live under /profile — the page
  // itself dispatches between freelance and agency layouts based on the
  // authenticated user's org type.
  return "/profile"
}

// ProfileCompletionBar renders "Profil rempli à X%" with a Soleil-
// corail progress fill. Clicking the bar navigates to the
// authenticated user's own profile page so the user lands directly
// where they can fill the missing sections — no intermediate modal,
// no extra tap. The optional `persona` prop scopes the report (and
// the navigation target) to the apporteur surface for /referral.
export function ProfileCompletionBar(props: ProfileCompletionBarProps) {
  const {
    variant = "page",
    collapsed = false,
    hideWhenComplete = false,
    persona,
  } = props
  const t = useTranslations("profileCompletion")
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const { data, isLoading } = useProfileCompletion(persona)

  if (isLoading || !data) return null
  if (hideWhenComplete && data.percent >= 100) return null

  const missingCount = data.total_sections - data.filled_sections
  const isComplete = data.percent >= 100
  const a11yLabel = t("a11yLabel", { percent: data.percent })
  const profilePath = profilePathFor(user?.role, org?.type, persona)

  if (variant === "sidebar" && collapsed) {
    return (
      <Link
        href={profilePath}
        className="mx-auto flex h-9 w-9 items-center justify-center rounded-full bg-primary-soft text-[11px] font-semibold text-primary-deep transition-colors hover:bg-primary-soft/70 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        title={a11yLabel}
        aria-label={a11yLabel}
      >
        {data.percent}%
      </Link>
    )
  }

  return (
    <Link
      href={profilePath}
      className={cn(
        "group block w-full rounded-xl text-left transition-colors",
        "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
        variant === "page"
          ? "bg-card border border-border p-5 shadow-[var(--shadow-card)] hover:bg-primary-soft/40"
          : "bg-background hover:bg-primary-soft/40 p-3",
      )}
      aria-label={a11yLabel}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p
            className={cn(
              "truncate font-medium text-foreground",
              variant === "page" ? "font-serif text-lg" : "text-sm",
            )}
          >
            {t("title", { percent: data.percent })}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {isComplete
              ? t("subtitleComplete")
              : t("subtitle", {
                  filled: data.filled_sections,
                  total: data.total_sections,
                })}
          </p>
        </div>
        {!isComplete && (
          <span
            className="inline-flex shrink-0 items-center gap-1 rounded-full bg-primary-soft px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wider text-primary-deep"
          >
            {missingCount}
            <ChevronRight className="h-3 w-3" aria-hidden="true" />
          </span>
        )}
      </div>
      <div
        className="mt-3 h-2 w-full overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-valuenow={data.percent}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className={cn(
            "h-full rounded-full transition-[width] duration-500 ease-out",
            isComplete ? "bg-success" : "bg-primary",
          )}
          style={{ width: `${data.percent}%` }}
        />
      </div>
    </Link>
  )
}
