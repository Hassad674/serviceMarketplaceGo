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
import { useSearchParams } from "next/navigation"
import { Link } from "@i18n/navigation"
import { useUser } from "@/shared/hooks/use-user"
import { cn } from "@/shared/lib/utils"

type StatCard = {
  icon: React.ElementType
  labelKey: string
  value: string
  iconBg: string
  iconColor: string
}

const AGENCY_STATS: StatCard[] = [
  {
    icon: Briefcase,
    labelKey: "activeMissions",
    value: "0",
    iconBg: "bg-blue-50 dark:bg-blue-500/15",
    iconColor: "text-blue-600 dark:text-blue-400",
  },
  {
    icon: MessageSquare,
    labelKey: "unreadMessages",
    value: "0",
    iconBg: "bg-violet-50 dark:bg-violet-500/15",
    iconColor: "text-violet-600 dark:text-violet-400",
  },
  {
    icon: TrendingUp,
    labelKey: "monthlyRevenue",
    value: "0 \u20AC",
    iconBg: "bg-emerald-50 dark:bg-emerald-500/15",
    iconColor: "text-emerald-600 dark:text-emerald-400",
  },
]

const PROVIDER_STATS: StatCard[] = [
  {
    icon: Briefcase,
    labelKey: "activeMissions",
    value: "0",
    iconBg: "bg-blue-50 dark:bg-blue-500/15",
    iconColor: "text-blue-600 dark:text-blue-400",
  },
  {
    icon: MessageSquare,
    labelKey: "unreadMessages",
    value: "0",
    iconBg: "bg-violet-50 dark:bg-violet-500/15",
    iconColor: "text-violet-600 dark:text-violet-400",
  },
  {
    icon: TrendingUp,
    labelKey: "monthlyRevenue",
    value: "0 \u20AC",
    iconBg: "bg-emerald-50 dark:bg-emerald-500/15",
    iconColor: "text-emerald-600 dark:text-emerald-400",
  },
]

const ENTERPRISE_STATS: StatCard[] = [
  {
    icon: FolderOpen,
    labelKey: "activeProjects",
    value: "0",
    iconBg: "bg-blue-50 dark:bg-blue-500/15",
    iconColor: "text-blue-600 dark:text-blue-400",
  },
  {
    icon: MessageSquare,
    labelKey: "unreadMessages",
    value: "0",
    iconBg: "bg-violet-50 dark:bg-violet-500/15",
    iconColor: "text-violet-600 dark:text-violet-400",
  },
  {
    icon: Wallet,
    labelKey: "totalBudget",
    value: "0 \u20AC",
    iconBg: "bg-emerald-50 dark:bg-emerald-500/15",
    iconColor: "text-emerald-600 dark:text-emerald-400",
  },
]

const REFERRER_STATS: StatCard[] = [
  {
    icon: Handshake,
    labelKey: "referrals",
    value: "0",
    iconBg: "bg-blue-50 dark:bg-blue-500/15",
    iconColor: "text-blue-600 dark:text-blue-400",
  },
  {
    icon: Clock,
    labelKey: "activeMissions",
    value: "0",
    iconBg: "bg-violet-50 dark:bg-violet-500/15",
    iconColor: "text-violet-600 dark:text-violet-400",
  },
  {
    icon: CheckCircle,
    labelKey: "completedMissions",
    value: "0",
    iconBg: "bg-emerald-50 dark:bg-emerald-500/15",
    iconColor: "text-emerald-600 dark:text-emerald-400",
  },
  {
    icon: TrendingUp,
    labelKey: "commissions",
    value: "0 \u20AC",
    iconBg: "bg-rose-50 dark:bg-rose-500/15",
    iconColor: "text-rose-600 dark:text-rose-400",
  },
]

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

export default function DashboardPage() {
  const { data: user } = useUser()
  const searchParams = useSearchParams()
  const t = useTranslations("dashboard")

  const role = user?.role ?? "enterprise"
  const userOrNull = user ?? null
  const isReferrerMode = searchParams.get("mode") === "referrer" && role === "provider"
  const stats = getStatsForRole(role, isReferrerMode)
  const subtitleKey = getSubtitleKey(role, isReferrerMode)
  const displayName = getDisplayName(userOrNull, isReferrerMode)

  const gridCols = stats.length === 4
    ? "grid-cols-1 sm:grid-cols-2 lg:grid-cols-4"
    : "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"

  return (
    <div className="space-y-5">
      {/* Welcome banner */}
      <div className="animate-slide-up relative overflow-hidden rounded-xl gradient-hero p-6 text-white">
        <div className="relative z-10">
          <h1 className="text-xl font-bold">
            {t("welcomeBack", { name: displayName })}
          </h1>
          <p className="mt-1 text-sm text-white/70">
            {t(subtitleKey)}
          </p>
        </div>
        <div className="absolute -right-6 -top-6 h-32 w-32 rounded-full bg-white/10" />
        <div className="absolute -right-2 top-10 h-20 w-20 rounded-full bg-white/5" />
        <div className="absolute left-1/2 -bottom-4 h-16 w-16 rounded-full bg-white/5" />
      </div>

      {/* Referrer switch for provider users */}
      {role === "provider" && (
        <div className="flex justify-end">
          {isReferrerMode ? (
            <Link
              href="/dashboard"
              className={cn(
                "flex items-center gap-2 rounded-lg px-4 py-2",
                "text-sm font-medium transition-all duration-200",
                "bg-emerald-50 text-emerald-700 hover:bg-emerald-100",
                "dark:bg-emerald-500/15 dark:text-emerald-400 dark:hover:bg-emerald-500/25",
              )}
            >
              <ArrowRightLeft className="h-4 w-4" strokeWidth={1.5} />
              {t("freelanceDashboard")}
            </Link>
          ) : (
            <Link
              href="/dashboard?mode=referrer"
              className={cn(
                "flex items-center gap-2 rounded-lg px-4 py-2",
                "text-sm font-medium text-white transition-all duration-200",
                "gradient-referrer hover:opacity-90 hover:shadow-md",
              )}
            >
              <Sparkles className="h-4 w-4" strokeWidth={1.5} />
              {t("businessReferrer")}
            </Link>
          )}
        </div>
      )}

      {/* Stats */}
      <div className={cn("grid gap-4", gridCols)}>
        {stats.map((stat, index) => (
          <div
            key={stat.labelKey}
            className={`animate-slide-up-delay-${index + 1} group rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-5 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md`}
          >
            <div className="flex items-center justify-between">
              <div className={`flex h-11 w-11 items-center justify-center rounded-lg ${stat.iconBg}`}>
                <stat.icon className={`h-5 w-5 ${stat.iconColor}`} strokeWidth={1.5} />
              </div>
              <span className="text-xs font-medium text-gray-400">&mdash;</span>
            </div>
            <div className="mt-3">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{t(stat.labelKey)}</p>
              <p className="mt-1 text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
