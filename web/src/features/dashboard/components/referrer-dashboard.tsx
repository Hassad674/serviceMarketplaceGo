"use client"

import { Handshake, Coins, CheckCircle, Trophy } from "lucide-react"
import { useTranslations } from "next-intl"
import { StatCard } from "./widgets/stat-card"
import { ActionsTodoCard } from "./widgets/actions-todo-card"
import { SectionCard } from "./widgets/section-card"
import type { DashboardAction } from "../types"

// ReferrerDashboard is the layout shipped when a Provider toggles
// `referrer_enabled` on. The numbers shown here are the "apporteur
// d'affaires" KPIs — pending intros, paid commissions, lifetime
// totals. It deliberately uses the existing referral feature data,
// passed in as plain props so the dashboard surface stays pure.

export interface ReferrerDashboardProps {
  activeReferrals: number
  pendingCommissionsLabel: string
  paid30dLabel: string
  lifetimeTotalLabel: string
  actions: DashboardAction[]
  actionsLoading: boolean
}

export function ReferrerDashboard(props: ReferrerDashboardProps) {
  const t = useTranslations("dashboard")
  const tStats = useTranslations("dashboard.stats")
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          icon={Handshake}
          tone="primary"
          label={tStats("activeReferrals")}
          value={String(props.activeReferrals)}
          hideDelta
        />
        <StatCard
          icon={Coins}
          tone="deep"
          label={tStats("pendingCommissions")}
          value={props.pendingCommissionsLabel}
          hideDelta
        />
        <StatCard
          icon={CheckCircle}
          tone="success"
          label={tStats("paid30d")}
          value={props.paid30dLabel}
          subLabel={tStats("last30Days")}
          hideDelta
        />
        <StatCard
          icon={Trophy}
          tone="amber"
          label={tStats("lifetimeTotal")}
          value={props.lifetimeTotalLabel}
          hideDelta
        />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ActionsTodoCard actions={props.actions} isLoading={props.actionsLoading} />
        <SectionCard
          title={t("referralsActive.title")}
          ctaLabel={t("referralsActive.cta")}
          ctaHref="/referrals"
          emptyMessage={t("referralsActive.empty")}
          isEmpty={props.activeReferrals === 0}
        >
          {props.activeReferrals > 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("referralsActive.count", { count: props.activeReferrals })}
            </p>
          ) : null}
        </SectionCard>
      </div>
    </div>
  )
}
