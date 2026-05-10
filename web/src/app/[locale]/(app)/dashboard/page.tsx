"use client"

import { useMemo } from "react"
import { useTranslations } from "next-intl"
import { useUser } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { useUnreadNotificationCount } from "@/features/notification/hooks/use-unread-notification-count"
import { useProfileCompletion } from "@/features/profile-completion/hooks/use-profile-completion"
import { useBillingProfileCompleteness } from "@/shared/hooks/billing-profile/use-billing-profile-completeness"
import {
  useApplicationsStats,
  useVisibilityStats,
} from "@/features/stats/hooks/use-stats"
import { ProviderDashboard } from "@/features/dashboard/components/provider-dashboard"
import { EnterpriseDashboard } from "@/features/dashboard/components/enterprise-dashboard"
import { ReferrerDashboard } from "@/features/dashboard/components/referrer-dashboard"
import { DashboardGreeting } from "@/features/dashboard/components/dashboard-greeting"
import type { DashboardAction, DashboardLayout } from "@/features/dashboard/types"

// Dashboard composition root. Owns every TanStack Query read so the
// per-role layouts inside `features/dashboard/components/` stay pure
// presentation. The `enabled` flag on `useVisibilityStats` /
// `useApplicationsStats` gates the request to the role that actually
// needs the data — an Enterprise never burns a /me/stats/visibility
// round trip and an Agency never hits /me/stats/enterprise-applications.

const KYC_DEADLINE_WARN_DAYS = 7

export default function DashboardPage() {
  const { data: user } = useUser()
  const { isReferrerMode } = useWorkspace()

  const role = user?.role ?? "enterprise"
  const layout: DashboardLayout =
    role === "provider" && isReferrerMode ? "referrer" : role

  const visibility = useVisibilityStats(7, {
    enabled: layout === "agency" || layout === "provider",
  })
  const applications = useApplicationsStats(7, {
    enabled: layout === "enterprise",
  })

  const actions = useDashboardActions()

  const greetingName = resolveDisplayName(user, layout)

  return (
    <div className="space-y-6">
      <DashboardGreeting
        layout={layout}
        displayName={greetingName}
        canSwitchWorkspace={role === "provider"}
      />
      {renderLayoutBody(layout, {
        visibility,
        applications,
        actions,
      })}
    </div>
  )
}

interface LayoutBodyDeps {
  visibility: ReturnType<typeof useVisibilityStats>
  applications: ReturnType<typeof useApplicationsStats>
  actions: DashboardActionsResult
}

function renderLayoutBody(layout: DashboardLayout, deps: LayoutBodyDeps) {
  if (layout === "referrer") {
    return (
      <ReferrerDashboard
        activeReferrals={0}
        pendingCommissionsLabel="—"
        paid30dLabel="—"
        lifetimeTotalLabel="—"
        actions={deps.actions.actions}
        actionsLoading={deps.actions.isLoading}
      />
    )
  }
  if (layout === "enterprise") {
    return (
      <EnterpriseDashboard
        applicationsStats={deps.applications.data}
        isApplicationsLoading={deps.applications.isLoading}
        activeRecruitments={0}
        pendingProposals={0}
        spendingLabel="—"
        actions={deps.actions.actions}
        actionsLoading={deps.actions.isLoading}
      />
    )
  }
  return (
    <ProviderDashboard
      visibilityStats={deps.visibility.data}
      isVisibilityLoading={deps.visibility.isLoading}
      monthlyRevenueLabel="—"
      pipelineCount={0}
      pipelineCtaHref="/missions"
      actions={deps.actions.actions}
      actionsLoading={deps.actions.isLoading}
    />
  )
}

interface DashboardActionsResult {
  actions: DashboardAction[]
  isLoading: boolean
}

// useDashboardActions lives at the page level — the dashboard
// feature folder is forbidden from importing other feature hooks
// (notification, profile-completion) by the project's
// `import/no-restricted-paths` lint rule. Hosting the aggregator at
// the composition root keeps the rule green while still putting the
// logic in one place.
function useDashboardActions(): DashboardActionsResult {
  const t = useTranslations("dashboard.actions")
  const { data: user, isLoading: userLoading } = useUser()
  const { data: completion, isLoading: completionLoading } = useProfileCompletion()
  const billingState = useBillingProfileCompleteness()
  const { data: unread, isLoading: unreadLoading } = useUnreadNotificationCount()

  return useMemo(() => {
    const list: DashboardAction[] = []
    pushKycAction(list, user?.kyc_status, user?.kyc_deadline, t)
    if (completion && completion.percent < 80) {
      list.push({
        id: "profile-completion",
        severity: completion.percent < 50 ? "warning" : "info",
        label: t("profileCompletion", { percent: completion.percent }),
        ctaLabel: t("profileCompletionCta"),
        href: "/profile",
      })
    }
    if (!billingState.isLoading && !billingState.isComplete) {
      list.push({
        id: "billing-profile",
        severity: "warning",
        label: t("billingProfile"),
        ctaLabel: t("billingProfileCta"),
        href: "/billing",
      })
    }
    if (typeof unread === "number" && unread > 0) {
      list.push({
        id: "messages-unread",
        severity: "info",
        label: t("unreadMessages", { count: unread }),
        ctaLabel: t("unreadMessagesCta"),
        href: "/messages",
      })
    }
    return {
      actions: list,
      isLoading:
        userLoading || completionLoading || billingState.isLoading || unreadLoading,
    }
  }, [
    user,
    completion,
    billingState.isComplete,
    billingState.isLoading,
    unread,
    userLoading,
    completionLoading,
    unreadLoading,
    t,
  ])
}

function pushKycAction(
  list: DashboardAction[],
  status: string | undefined,
  deadline: string | undefined,
  t: ReturnType<typeof useTranslations>,
): void {
  if (status === "restricted") {
    list.push({
      id: "kyc-restricted",
      severity: "critical",
      label: t("kycRestricted"),
      ctaLabel: t("kycCta"),
      href: "/payment-info",
    })
    return
  }
  if (status !== "pending" || !deadline) return
  const target = new Date(deadline).getTime()
  if (Number.isNaN(target)) return
  const days = Math.ceil((target - Date.now()) / (24 * 60 * 60 * 1000))
  if (days > KYC_DEADLINE_WARN_DAYS) return
  list.push({
    id: "kyc-pending",
    severity: days <= 2 ? "critical" : "warning",
    label: t("kycPending", { days: Math.max(days, 0) }),
    ctaLabel: t("kycCta"),
    href: "/payment-info",
  })
}

function resolveDisplayName(
  user:
    | { first_name?: string; display_name?: string; role?: string }
    | undefined,
  layout: DashboardLayout,
): string {
  if (!user) return ""
  if (layout === "provider" || layout === "referrer") {
    return user.first_name || user.display_name || "Freelance"
  }
  return user.display_name ?? ""
}
