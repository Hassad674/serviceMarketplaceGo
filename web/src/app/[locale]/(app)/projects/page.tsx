"use client"

import { useState, useMemo } from "react"
import {
  FolderOpen,
  Calendar,
  Search,
  TrendingUp,
  Clock,
  DollarSign,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { useProjects } from "@/features/proposal/hooks/use-proposals"
import type { ProposalResponse, ProposalStatus } from "@/features/proposal/types"

type TabKey = "inProgress" | "completed" | "all"

const MISSION_STATUSES: ProposalStatus[] = [
  "paid",
  "active",
  "completion_requested",
  "completed",
]

const IN_PROGRESS_STATUSES: ProposalStatus[] = [
  "paid",
  "active",
  "completion_requested",
]

export default function ProjectsListPage() {
  const t = useTranslations("projects")
  const { data, isLoading } = useProjects()
  const [activeTab, setActiveTab] = useState<TabKey>("inProgress")
  const [searchQuery, setSearchQuery] = useState("")

  const allProjects = data?.data ?? []

  // Filter to mission-relevant statuses only
  const missions = useMemo(
    () => allProjects.filter((p) => MISSION_STATUSES.includes(p.status)),
    [allProjects],
  )

  const inProgress = useMemo(
    () => missions.filter((p) => IN_PROGRESS_STATUSES.includes(p.status)),
    [missions],
  )

  const completed = useMemo(
    () => missions.filter((p) => p.status === "completed"),
    [missions],
  )

  const tabProjects = useMemo(() => {
    const base =
      activeTab === "inProgress"
        ? inProgress
        : activeTab === "completed"
          ? completed
          : missions

    if (!searchQuery.trim()) return base

    const query = searchQuery.toLowerCase()
    return base.filter((p) =>
      p.title.toLowerCase().includes(query) ||
      p.client_name.toLowerCase().includes(query) ||
      p.provider_name.toLowerCase().includes(query)
    )
  }, [activeTab, inProgress, completed, missions, searchQuery])

  const stats = useMemo(() => computeStats(inProgress), [inProgress])

  return (
    <div className="space-y-6">
      {/* Header */}
      <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
        {t("title")}
      </h1>

      {/* Loading skeleton */}
      {isLoading && <ProjectsSkeleton />}

      {!isLoading && (
        <>
          {/* Stats row */}
          <StatsRow stats={stats} />

          {/* Tabs */}
          <TabBar
            activeTab={activeTab}
            onTabChange={setActiveTab}
            inProgressCount={inProgress.length}
            completedCount={completed.length}
            allCount={missions.length}
          />

          {/* Search */}
          <SearchInput value={searchQuery} onChange={setSearchQuery} />

          {/* Project list */}
          {tabProjects.length === 0 ? (
            <EmptyState />
          ) : (
            <div className="flex flex-col gap-3">
              {tabProjects.map((project) => (
                <ProjectCard key={project.id} project={project} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}

interface StatsData {
  activeCount: number
  totalAmount: number
  nextDeadline: string | null
}

function computeStats(inProgress: ProposalResponse[]): StatsData {
  const activeCount = inProgress.filter(
    (p) => p.status === "active" || p.status === "completion_requested",
  ).length

  const totalAmount = inProgress.reduce(
    (sum, p) => sum + p.amount / 100,
    0,
  )

  const deadlines = inProgress
    .filter((p) => p.deadline !== null)
    .map((p) => new Date(p.deadline as string))
    .filter((d) => d > new Date())
    .sort((a, b) => a.getTime() - b.getTime())

  const nextDeadline = deadlines.length > 0
    ? formatShortDate(deadlines[0])
    : null

  return { activeCount, totalAmount, nextDeadline }
}

function formatShortDate(date: Date): string {
  return new Intl.DateTimeFormat("fr-FR", {
    day: "numeric",
    month: "short",
  }).format(date)
}

function StatsRow({ stats }: { stats: StatsData }) {
  const t = useTranslations("projects")

  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <StatCard
        icon={TrendingUp}
        label={t("activeMissions")}
        value={String(stats.activeCount)}
        iconBg="bg-emerald-50 dark:bg-emerald-500/10"
        iconColor="text-emerald-600 dark:text-emerald-400"
      />
      <StatCard
        icon={DollarSign}
        label={t("totalAmount")}
        value={formatCurrency(stats.totalAmount)}
        iconBg="bg-blue-50 dark:bg-blue-500/10"
        iconColor="text-blue-600 dark:text-blue-400"
      />
      <StatCard
        icon={Clock}
        label={t("nextDeadline")}
        value={stats.nextDeadline ?? "\u2014"}
        iconBg="bg-amber-50 dark:bg-amber-500/10"
        iconColor="text-amber-600 dark:text-amber-400"
      />
    </div>
  )
}

interface StatCardProps {
  icon: React.ElementType
  label: string
  value: string
  iconBg: string
  iconColor: string
}

function StatCard({ icon: Icon, label, value, iconBg, iconColor }: StatCardProps) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-slate-100 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <div className={cn("flex h-10 w-10 shrink-0 items-center justify-center rounded-lg", iconBg)}>
        <Icon className={cn("h-5 w-5", iconColor)} strokeWidth={1.5} />
      </div>
      <div>
        <p className="text-2xl font-semibold text-slate-900 dark:text-white">
          {value}
        </p>
        <p className="text-xs text-slate-500 dark:text-slate-400">{label}</p>
      </div>
    </div>
  )
}

interface TabBarProps {
  activeTab: TabKey
  onTabChange: (tab: TabKey) => void
  inProgressCount: number
  completedCount: number
  allCount: number
}

function TabBar({ activeTab, onTabChange, inProgressCount, completedCount, allCount }: TabBarProps) {
  const t = useTranslations("projects")

  const tabs: { key: TabKey; label: string; count: number }[] = [
    { key: "inProgress", label: t("inProgress"), count: inProgressCount },
    { key: "completed", label: t("completedTab"), count: completedCount },
    { key: "all", label: t("all"), count: allCount },
  ]

  return (
    <div className="flex gap-1 border-b border-slate-200 dark:border-slate-700" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.key}
          onClick={() => onTabChange(tab.key)}
          className={cn(
            "px-4 py-2.5 text-sm font-medium transition-colors relative",
            activeTab === tab.key
              ? "text-rose-600 dark:text-rose-400"
              : "text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300",
          )}
        >
          {tab.label} ({tab.count})
          {activeTab === tab.key && (
            <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-rose-500 rounded-full" />
          )}
        </button>
      ))}
    </div>
  )
}

function SearchInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const t = useTranslations("projects")

  return (
    <div className="relative">
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" strokeWidth={1.5} />
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={t("searchPlaceholder")}
        className={cn(
          "w-full h-10 rounded-lg border border-slate-200 bg-white pl-9 pr-4 text-sm",
          "placeholder:text-slate-400 text-slate-900",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "shadow-xs transition-all duration-200",
          "dark:border-slate-600 dark:bg-slate-800 dark:text-white dark:placeholder:text-slate-500",
          "dark:focus:border-rose-500 dark:focus:ring-rose-500/10",
        )}
      />
    </div>
  )
}

function ProjectCard({ project }: { project: ProposalResponse }) {
  const t = useTranslations("projects")
  const { data: user } = useUser()

  const statusConfig = getStatusDot(project.status)
  const isCompletionRequested = project.status === "completion_requested"
  const isCompleted = project.status === "completed"
  const partnerName = user?.id === project.client_id
    ? project.provider_name
    : project.client_name

  return (
    <Link
      href={`/projects/${project.id}`}
      className={cn(
        "flex items-center gap-4 rounded-2xl border bg-white px-5 py-4",
        "transition-all duration-200",
        "border-slate-100 shadow-sm",
        "hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5",
        "dark:bg-slate-800/80 dark:border-slate-700 dark:hover:border-rose-500/30",
      )}
    >
      {/* Status dot */}
      <div className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusConfig.dotClass)} />

      {/* Title */}
      <div className="min-w-0 flex-1">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-white truncate">
          {project.title}
        </h3>
        {partnerName && (
          <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
            {t("with")}: {partnerName}
          </p>
        )}
        {isCompletionRequested && (
          <p className="mt-0.5 text-xs text-amber-600 dark:text-amber-400">
            {t("completionPending")}
          </p>
        )}
        {isCompleted && (
          <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
            {t("completed")}
          </p>
        )}
      </div>

      {/* Amount + deadline */}
      <div className="shrink-0 text-right">
        <p className="text-sm font-bold text-slate-900 dark:text-white">
          {formatCurrency(project.amount / 100)}
        </p>
        {project.deadline && (
          <p className="mt-0.5 flex items-center justify-end gap-1 text-xs text-slate-500 dark:text-slate-400">
            <Calendar className="h-3 w-3" strokeWidth={1.5} />
            {formatShortDate(new Date(project.deadline))}
          </p>
        )}
      </div>
    </Link>
  )
}

function getStatusDot(status: ProposalStatus): { dotClass: string } {
  const map: Record<string, string> = {
    active: "bg-green-500",
    completion_requested: "bg-amber-500",
    completed: "bg-blue-500",
    paid: "bg-emerald-500",
  }
  return { dotClass: map[status] ?? "bg-slate-400" }
}

function EmptyState() {
  const t = useTranslations("projects")

  return (
    <div className="rounded-xl border border-dashed border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-900 p-12 text-center">
      <FolderOpen className="mx-auto h-10 w-10 text-slate-300 dark:text-slate-600" />
      <p className="mt-4 text-sm font-medium text-slate-700 dark:text-slate-300">
        {t("emptyState")}
      </p>
      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
        {t("emptyStateSubtitle")}
      </p>
    </div>
  )
}

function ProjectsSkeleton() {
  return (
    <div className="space-y-6">
      {/* Stats skeleton */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="flex items-center gap-3 rounded-xl border border-slate-100 bg-white p-4 dark:border-slate-700 dark:bg-slate-800"
          >
            <div className="h-10 w-10 animate-shimmer rounded-lg bg-slate-200 dark:bg-slate-700" />
            <div className="space-y-1.5">
              <div className="h-5 w-16 animate-shimmer rounded bg-slate-200 dark:bg-slate-700" />
              <div className="h-3 w-24 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
            </div>
          </div>
        ))}
      </div>
      {/* Cards skeleton */}
      <div className="flex flex-col gap-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="flex items-center gap-4 rounded-2xl border border-slate-100 bg-white px-5 py-4 dark:border-slate-700 dark:bg-slate-800/80"
          >
            <div className="h-2.5 w-2.5 animate-shimmer rounded-full bg-slate-200 dark:bg-slate-700" />
            <div className="flex-1 space-y-1.5">
              <div className="h-4 w-3/4 animate-shimmer rounded bg-slate-200 dark:bg-slate-700" />
            </div>
            <div className="space-y-1.5 text-right">
              <div className="h-4 w-20 animate-shimmer rounded bg-slate-200 dark:bg-slate-700" />
              <div className="h-3 w-14 animate-shimmer rounded bg-slate-100 dark:bg-slate-700 ml-auto" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
