"use client"

import {
  Briefcase,
  MessageSquare,
  TrendingUp,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useAuth } from "@/shared/hooks/use-auth"

export default function AgencyDashboardPage() {
  const { user } = useAuth()
  const t = useTranslations("dashboard")

  const stats = [
    {
      icon: Briefcase,
      labelKey: "activeMissions" as const,
      value: "0",
      iconBg: "bg-blue-50",
      iconColor: "text-blue-600",
    },
    {
      icon: MessageSquare,
      labelKey: "unreadMessages" as const,
      value: "0",
      iconBg: "bg-violet-50",
      iconColor: "text-violet-600",
    },
    {
      icon: TrendingUp,
      labelKey: "monthlyRevenue" as const,
      value: "0 \u20AC",
      iconBg: "bg-emerald-50",
      iconColor: "text-emerald-600",
    },
  ]

  return (
    <div className="space-y-5">
      {/* Welcome banner */}
      <div className="animate-slide-up relative overflow-hidden rounded-xl gradient-hero p-6 text-white">
        <div className="relative z-10">
          <h1 className="text-xl font-bold">
            {t("welcomeBack", { name: user?.display_name ?? "Agency" })}
          </h1>
          <p className="mt-1 text-sm text-white/70">
            {t("agencySubtitle")}
          </p>
        </div>
        <div className="absolute -right-6 -top-6 h-32 w-32 rounded-full bg-white/10" />
        <div className="absolute -right-2 top-10 h-20 w-20 rounded-full bg-white/5" />
        <div className="absolute left-1/2 -bottom-4 h-16 w-16 rounded-full bg-white/5" />
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {stats.map((stat, index) => (
          <div
            key={stat.labelKey}
            className={`animate-slide-up-delay-${index + 1} group rounded-xl border border-gray-100 bg-white p-5 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md`}
          >
            <div className="flex items-center justify-between">
              <div className={`flex h-11 w-11 items-center justify-center rounded-lg ${stat.iconBg}`}>
                <stat.icon className={`h-5 w-5 ${stat.iconColor}`} strokeWidth={1.5} />
              </div>
              <span className="text-xs font-medium text-gray-400">&mdash;</span>
            </div>
            <div className="mt-3">
              <p className="text-sm font-medium text-gray-500">{t(stat.labelKey)}</p>
              <p className="mt-1 text-2xl font-bold tracking-tight text-gray-900">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
