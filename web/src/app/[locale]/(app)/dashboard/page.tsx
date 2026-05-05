"use client"

import {
  Briefcase,
  MessageSquare,
  TrendingUp,
  FolderOpen,
  Wallet,
  Handshake,
  Clock,
  CheckCircle,
  ArrowRightLeft,
  Sparkles,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { useUser } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"

// W-11 Dashboard freelance — Soleil v2.
// Source: design/assets/sources/phase1/soleil-lotC.jsx `SoleilFreelancerDashboard`
// + design/assets/sources/phase1/soleil.jsx `SoleilDashboard` (reference layout).
//
// Out-of-scope flagged (NOT shipped this batch — features absent from repo):
//   - "Cette semaine chez Atelier" editorial card (no blog/content feature)
//   - "Atelier Premium" sidebar bottom CTA (subscription tier UX
//     handled separately via the SubscriptionBadge in the header)
//   - "Tes missions du moment" 3-mission progress list (the dashboard
//     currently aggregates only stat counts, no detailed mission feed)
//   - "Conversations en cours" 3-thread list (would require a new hook)

type StatCard = {
  icon: React.ElementType
  labelKey: string
  value: string
  // Each stat tone uses one Soleil-compatible pair. Keeps the card
  // grid varied without breaking out of the warm palette.
  iconBg: string
  iconColor: string
}

// Soleil-compatible tone scale (4 distinguishable tints inside the
// warm palette). Each role's first three cards cycle through them.
const SOLEIL_TONE = {
  corail: { bg: "bg-primary-soft", color: "text-primary" },
  sapin: { bg: "bg-success-soft", color: "text-success" },
  pink: { bg: "bg-pink-soft", color: "text-primary-deep" },
  amber: { bg: "bg-amber-soft", color: "text-foreground" },
} as const

const AGENCY_STATS: StatCard[] = [
  { icon: Briefcase, labelKey: "activeMissions", value: "0", ...spread(SOLEIL_TONE.corail) },
  { icon: MessageSquare, labelKey: "unreadMessages", value: "0", ...spread(SOLEIL_TONE.pink) },
  { icon: TrendingUp, labelKey: "monthlyRevenue", value: "0 €", ...spread(SOLEIL_TONE.sapin) },
]

const PROVIDER_STATS: StatCard[] = [
  { icon: Briefcase, labelKey: "activeMissions", value: "0", ...spread(SOLEIL_TONE.corail) },
  { icon: MessageSquare, labelKey: "unreadMessages", value: "0", ...spread(SOLEIL_TONE.pink) },
  { icon: TrendingUp, labelKey: "monthlyRevenue", value: "0 €", ...spread(SOLEIL_TONE.sapin) },
]

const ENTERPRISE_STATS: StatCard[] = [
  { icon: FolderOpen, labelKey: "activeProjects", value: "0", ...spread(SOLEIL_TONE.corail) },
  { icon: MessageSquare, labelKey: "unreadMessages", value: "0", ...spread(SOLEIL_TONE.pink) },
  { icon: Wallet, labelKey: "totalBudget", value: "0 €", ...spread(SOLEIL_TONE.sapin) },
]

const REFERRER_STATS: StatCard[] = [
  { icon: Handshake, labelKey: "referrals", value: "0", ...spread(SOLEIL_TONE.corail) },
  { icon: Clock, labelKey: "activeMissions", value: "0", ...spread(SOLEIL_TONE.pink) },
  { icon: CheckCircle, labelKey: "completedMissions", value: "0", ...spread(SOLEIL_TONE.sapin) },
  { icon: TrendingUp, labelKey: "commissions", value: "0 €", ...spread(SOLEIL_TONE.amber) },
]

function spread(tone: { bg: string; color: string }) {
  return { iconBg: tone.bg, iconColor: tone.color }
}

function getStatsForRole(role: string, isReferrerMode: boolean): StatCard[] {
  if (role === "provider" && isReferrerMode) return REFERRER_STATS
  if (role === "provider") return PROVIDER_STATS
  if (role === "agency") return AGENCY_STATS
  if (role === "enterprise") return ENTERPRISE_STATS
  return ENTERPRISE_STATS
}

function getSubtitleKey(role: string, isReferrerMode: boolean): string {
  if (role === "provider" && isReferrerMode) return "referrerSubtitle"
  if (role === "provider") return "providerSubtitle"
  if (role === "agency") return "agencySubtitle"
  if (role === "enterprise") return "enterpriseSubtitle"
  return "enterpriseSubtitle"
}

function getDisplayName(
  user: { first_name: string; last_name: string; display_name: string; role: string } | null,
  isReferrerMode: boolean,
): string {
  if (!user) return ""
  if (user.role === "provider" || isReferrerMode) {
    return user.first_name || user.display_name || "Freelance"
  }
  if (user.role === "agency") return user.display_name ?? "Agency"
  if (user.role === "enterprise") return user.display_name ?? "Enterprise"
  return user.display_name ?? ""
}

function DashboardContent() {
  const router = useRouter()
  const { data: user } = useUser()
  const { isReferrerMode, switchToReferrer, switchToFreelance } = useWorkspace()
  const t = useTranslations("dashboard")

  const role = user?.role ?? "enterprise"
  const userOrNull = user ?? null
  const effectiveReferrerMode = isReferrerMode && role === "provider"
  const stats = getStatsForRole(role, effectiveReferrerMode)
  const subtitleKey = getSubtitleKey(role, effectiveReferrerMode)
  const displayName = getDisplayName(userOrNull, effectiveReferrerMode)

  const gridCols = stats.length === 4
    ? "grid-cols-1 sm:grid-cols-2 lg:grid-cols-4"
    : "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"

  return (
    <div className="space-y-6">
      {/* Editorial greeting — Fraunces display with italic corail accent */}
      <div className="animate-slide-up">
        <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("welcomeEyebrow")}
        </p>
        <h1 className="font-serif text-[32px] font-normal leading-[1.05] tracking-[-0.025em] text-foreground sm:text-[40px]">
          {t("welcomePrefix", { name: displayName })}{" "}
          <span className="italic text-primary">{t("welcomeAccent")}</span>
        </h1>
        <p className="mt-2 max-w-xl text-[14.5px] leading-relaxed text-muted-foreground">
          {t(subtitleKey)}
        </p>
      </div>

      {/* Referrer switch (provider only) */}
      {role === "provider" && (
        <div className="flex justify-end">
          {effectiveReferrerMode ? (
            <Button
              variant="ghost"
              size="auto"
              onClick={() => {
                const targetPath = switchToFreelance()
                router.push(targetPath)
              }}
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2",
                "text-sm font-medium transition-all duration-200",
                "bg-success-soft text-success hover:opacity-80",
              )}
            >
              <ArrowRightLeft className="h-4 w-4" strokeWidth={1.5} />
              {t("freelanceDashboard")}
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="auto"
              onClick={() => {
                const targetPath = switchToReferrer()
                router.push(targetPath)
              }}
              className={cn(
                "flex items-center gap-2 rounded-full px-4 py-2",
                "text-sm font-medium text-foreground transition-all duration-200",
                "gradient-coral hover:opacity-90",
              )}
            >
              <Sparkles className="h-4 w-4" strokeWidth={1.5} />
              {t("businessReferrer")}
            </Button>
          )}
        </div>
      )}

      {/* Stats — Soleil card grid */}
      <div className={cn("grid gap-4", gridCols)}>
        {stats.map((stat, index) => (
          <div
            key={stat.labelKey}
            className={cn(
              `animate-slide-up-delay-${index + 1}`,
              "group rounded-2xl border border-border bg-card p-6",
              "transition-all duration-200 hover:-translate-y-0.5 hover:border-border-strong",
            )}
            style={{ boxShadow: "var(--shadow-card)" }}
          >
            <div className={cn("flex h-12 w-12 items-center justify-center rounded-2xl", stat.iconBg)}>
              <stat.icon className={cn("h-5 w-5", stat.iconColor)} strokeWidth={1.5} />
            </div>
            <p className="mt-4 text-[13px] font-medium text-muted-foreground">
              {t(stat.labelKey)}
            </p>
            <p className="mt-1 font-serif text-[30px] font-medium leading-tight tracking-[-0.02em] text-foreground">
              {stat.value}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}

export default function DashboardPage() {
  return <DashboardContent />
}
