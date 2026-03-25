"use client"

import {
  Briefcase,
  MessageSquare,
  TrendingUp,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"

const STATS = [
  {
    icon: Briefcase,
    label: "Active Missions",
    value: "0",
    iconBg: "bg-blue-50",
    iconColor: "text-blue-600",
  },
  {
    icon: MessageSquare,
    label: "Unread Messages",
    value: "0",
    iconBg: "bg-violet-50",
    iconColor: "text-violet-600",
  },
  {
    icon: TrendingUp,
    label: "Monthly Revenue",
    value: "0 \u20AC",
    iconBg: "bg-emerald-50",
    iconColor: "text-emerald-600",
  },
] as const

export default function ProviderDashboardPage() {
  const { user } = useAuth()

  const displayName = user?.first_name || user?.display_name || "Freelance"

  return (
    <div className="space-y-6">
      {/* Welcome banner */}
      <div className="animate-slide-up relative overflow-hidden rounded-2xl gradient-hero p-8 text-white">
        <div className="relative z-10">
          <h1 className="text-2xl font-bold">
            Welcome back, {displayName}
          </h1>
          <p className="mt-1 text-white/80">
            Manage your missions and grow your activity
          </p>
        </div>
        <div className="absolute -right-8 -top-8 h-40 w-40 rounded-full bg-white/10" />
        <div className="absolute -right-4 top-12 h-24 w-24 rounded-full bg-white/5" />
        <div className="absolute left-1/2 -bottom-6 h-20 w-20 rounded-full bg-white/5" />
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {STATS.map((stat, index) => (
          <div
            key={stat.label}
            className={`animate-slide-up-delay-${index + 1} group rounded-2xl border border-gray-100 bg-white p-6 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md`}
          >
            <div className="flex items-center justify-between">
              <div className={`flex h-12 w-12 items-center justify-center rounded-xl ${stat.iconBg}`}>
                <stat.icon className={`h-5 w-5 ${stat.iconColor}`} strokeWidth={1.5} />
              </div>
              <span className="text-xs font-medium text-gray-400">&mdash;</span>
            </div>
            <div className="mt-4">
              <p className="text-sm font-medium text-gray-500">{stat.label}</p>
              <p className="mt-1 text-3xl font-bold tracking-tight text-gray-900">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
