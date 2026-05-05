"use client"

import { useState, useMemo } from "react"
import { FolderOpen, Calendar, Search } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { useProjects } from "@/features/proposal/hooks/use-proposals"
import type { ProposalResponse, ProposalStatus } from "@/features/proposal/types"
import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

// Soleil v2 — Projects (active missions) list page.
// Editorial header (corail eyebrow + Fraunces italic-corail title +
// tabac subtitle), Soleil pill filter tabs, ivoire search input, Soleil
// project cards with status pill + Geist Mono budget.

type TabKey = "inProgress" | "completed" | "all"

const MISSION_STATUSES: ProposalStatus[] = [
  "paid",
  "active",
  "completion_requested",
  "completed",
  "disputed",
]

const IN_PROGRESS_STATUSES: ProposalStatus[] = [
  "paid",
  "active",
  "completion_requested",
  "disputed",
]

export default function ProjectsListPage() {
  const t = useTranslations("projects")
  const tFlow = useTranslations("proposal")
  const { data, isLoading } = useProjects()
  const [activeTab, setActiveTab] = useState<TabKey>("inProgress")
  const [searchQuery, setSearchQuery] = useState("")

  const allProjects = useMemo(() => data?.data ?? [], [data?.data])

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
    return base.filter(
      (p) =>
        p.title.toLowerCase().includes(query) ||
        p.client_name.toLowerCase().includes(query) ||
        p.provider_name.toLowerCase().includes(query),
    )
  }, [activeTab, inProgress, completed, missions, searchQuery])

  return (
    <div className="space-y-8">
      {/* Editorial header */}
      <div className="space-y-2">
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {tFlow("proposalFlow_list_eyebrow")}
        </p>
        <h1 className="font-serif text-[28px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[36px]">
          {tFlow("proposalFlow_list_titlePrefix")}{" "}
          <span className="italic text-primary">
            {tFlow("proposalFlow_list_titleAccent")}
          </span>
        </h1>
        <p className="max-w-2xl text-[14.5px] leading-relaxed text-muted-foreground">
          {tFlow("proposalFlow_list_subtitle")}
        </p>
      </div>

      {/* Loading skeleton */}
      {isLoading && <ProjectsSkeleton />}

      {!isLoading && (
        <>
          {/* Filter pills */}
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

function formatShortDate(date: Date): string {
  return new Intl.DateTimeFormat("fr-FR", {
    day: "numeric",
    month: "short",
  }).format(date)
}

interface TabBarProps {
  activeTab: TabKey
  onTabChange: (tab: TabKey) => void
  inProgressCount: number
  completedCount: number
  allCount: number
}

function TabBar({
  activeTab,
  onTabChange,
  inProgressCount,
  completedCount,
  allCount,
}: TabBarProps) {
  const t = useTranslations("projects")

  const tabs: { key: TabKey; label: string; count: number }[] = [
    { key: "inProgress", label: t("inProgress"), count: inProgressCount },
    { key: "completed", label: t("completedTab"), count: completedCount },
    { key: "all", label: t("all"), count: allCount },
  ]

  return (
    <div role="tablist" className="flex flex-wrap gap-2">
      {tabs.map((tab) => {
        const isActive = activeTab === tab.key
        return (
          <Button
            variant="ghost"
            size="auto"
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onTabChange(tab.key)}
            className={cn(
              "inline-flex items-center gap-2 rounded-full border px-4 py-2 text-[13px] font-medium",
              "transition-colors duration-150",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
              isActive
                ? "border-primary bg-primary-soft text-primary-deep"
                : "border-border bg-card text-foreground hover:border-border-strong",
            )}
          >
            {tab.label}
            <span
              className={cn(
                "inline-flex h-5 min-w-[1.25rem] items-center justify-center rounded-full px-1.5",
                "font-mono text-[11px] font-bold",
                isActive ? "bg-primary text-primary-foreground" : "bg-border text-muted-foreground",
              )}
            >
              {tab.count}
            </span>
          </Button>
        )
      })}
    </div>
  )
}

function SearchInput({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  const t = useTranslations("projects")

  return (
    <div className="relative">
      <Search
        className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-subtle-foreground"
        strokeWidth={1.7}
        aria-hidden="true"
      />
      <Input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={t("searchPlaceholder")}
        className={cn(
          "h-11 w-full rounded-full border border-border bg-card pl-10 pr-4 text-[14px]",
          "placeholder:text-subtle-foreground text-foreground",
          "transition-all duration-200 ease-out",
          "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
        )}
      />
    </div>
  )
}

function ProjectCard({ project }: { project: ProposalResponse }) {
  const t = useTranslations("projects")
  const { data: user } = useUser()

  const statusConfig = getStatusConfig(project.status)
  const isCompletionRequested = project.status === "completion_requested"
  const isCompleted = project.status === "completed"
  const partnerName =
    user?.id === project.client_id ? project.provider_name : project.client_name

  return (
    <Link
      href={`/projects/${project.id}`}
      className={cn(
        "group flex items-center gap-4 rounded-2xl border border-border bg-card px-5 py-4",
        "transition-all duration-200 ease-out",
        "hover:-translate-y-0.5 hover:border-border-strong",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      {/* Status pill */}
      <span
        className={cn(
          "inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1",
          "font-mono text-[10.5px] font-bold uppercase tracking-[0.05em]",
          statusConfig.pillBg,
          statusConfig.pillText,
        )}
      >
        <span
          className={cn("h-1.5 w-1.5 rounded-full", statusConfig.dotClass)}
          aria-hidden="true"
        />
        {statusConfig.label}
      </span>

      {/* Title */}
      <div className="min-w-0 flex-1">
        <h3 className="truncate font-serif text-[16px] font-medium tracking-[-0.01em] text-foreground">
          {project.title}
        </h3>
        {partnerName && (
          <p className="mt-0.5 text-[12.5px] text-muted-foreground">
            {t("with")}: {partnerName}
          </p>
        )}
        {isCompletionRequested && (
          <p className="mt-0.5 font-mono text-[11px] font-medium text-warning">
            {t("completionPending")}
          </p>
        )}
        {isCompleted && (
          <p className="mt-0.5 font-mono text-[11px] font-medium text-muted-foreground">
            {t("completed")}
          </p>
        )}
      </div>

      {/* Amount + deadline */}
      <div className="shrink-0 text-right">
        <p className="font-mono text-[14.5px] font-bold text-foreground">
          {formatCurrency(project.amount / 100)}
        </p>
        {project.deadline && (
          <p className="mt-0.5 inline-flex items-center justify-end gap-1 font-mono text-[11px] text-subtle-foreground">
            <Calendar className="h-3 w-3" strokeWidth={1.7} aria-hidden="true" />
            {formatShortDate(new Date(project.deadline))}
          </p>
        )}
      </div>
    </Link>
  )
}

function getStatusConfig(status: ProposalStatus): {
  label: string
  pillBg: string
  pillText: string
  dotClass: string
} {
  switch (status) {
    case "active":
    case "paid":
      return {
        label: "Active",
        pillBg: "bg-success-soft",
        pillText: "text-success",
        dotClass: "bg-success",
      }
    case "completion_requested":
      return {
        label: "À valider",
        pillBg: "bg-amber-soft",
        pillText: "text-warning",
        dotClass: "bg-warning",
      }
    case "completed":
      return {
        label: "Terminé",
        pillBg: "bg-border",
        pillText: "text-muted-foreground",
        dotClass: "bg-subtle-foreground",
      }
    case "disputed":
      return {
        label: "Litige",
        pillBg: "bg-amber-soft",
        pillText: "text-warning",
        dotClass: "bg-warning",
      }
    default:
      return {
        label: status,
        pillBg: "bg-border",
        pillText: "text-muted-foreground",
        dotClass: "bg-subtle-foreground",
      }
  }
}

function EmptyState() {
  const t = useTranslations("projects")

  return (
    <div
      className={cn(
        "rounded-2xl border-2 border-dashed border-border-strong bg-background p-12 text-center",
      )}
    >
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary-soft text-primary">
        <FolderOpen className="h-5 w-5" strokeWidth={1.7} aria-hidden="true" />
      </div>
      <p className="mt-4 font-serif text-[18px] font-medium tracking-[-0.01em] text-foreground">
        {t("emptyState")}
      </p>
      <p className="mt-1.5 text-[13.5px] text-muted-foreground">
        {t("emptyStateSubtitle")}
      </p>
    </div>
  )
}

function ProjectsSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap gap-2">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-9 w-28 animate-shimmer rounded-full bg-border"
          />
        ))}
      </div>
      <div className="h-11 w-full animate-shimmer rounded-full bg-border" />
      <div className="flex flex-col gap-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="flex items-center gap-4 rounded-2xl border border-border bg-card px-5 py-4"
            style={{ boxShadow: "var(--shadow-card)" }}
          >
            <div className="h-6 w-20 animate-shimmer rounded-full bg-border" />
            <div className="flex-1 space-y-1.5">
              <div className="h-4 w-3/4 animate-shimmer rounded bg-border" />
              <div className="h-3 w-1/2 animate-shimmer rounded bg-border/60" />
            </div>
            <div className="space-y-1.5 text-right">
              <div className="h-4 w-20 animate-shimmer rounded bg-border" />
              <div className="ml-auto h-3 w-14 animate-shimmer rounded bg-border/60" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
