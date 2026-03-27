"use client"

import {
  FolderOpen,
  Plus,
  Calendar,
  Clock,
  CheckCircle2,
  DollarSign,
  Star,
  ArrowRight,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn, formatCurrency } from "@/shared/lib/utils"
import { useProjects } from "@/features/proposal/hooks/use-proposals"
import type { ProposalResponse, ProposalStatus } from "@/features/proposal/types"

export default function ProjectsListPage() {
  const t = useTranslations("projects")
  const tp = useTranslations("proposal")
  const { data, isLoading } = useProjects()

  const projects = data?.data ?? []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
          {t("title")}
        </h1>
        <Link
          href="/projects/new"
          className="inline-flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white gradient-primary transition-all duration-200 hover:shadow-glow active:scale-[0.98]"
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
          {t("createProject")}
        </Link>
      </div>

      {/* Loading skeleton */}
      {isLoading && <ProjectsSkeleton />}

      {/* Empty state */}
      {!isLoading && projects.length === 0 && (
        <div className="rounded-xl border border-dashed border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 p-12 text-center">
          <FolderOpen className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
          <p className="mt-4 text-sm font-medium text-gray-500 dark:text-gray-400">
            {t("noProjects")}
          </p>
          <Link
            href="/projects/new"
            className="mt-4 inline-flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white gradient-primary transition-all duration-200 hover:shadow-glow active:scale-[0.98]"
          >
            <Plus className="h-4 w-4" strokeWidth={2} />
            {t("createProject")}
          </Link>
        </div>
      )}

      {/* Project cards — horizontal full-width list */}
      {!isLoading && projects.length > 0 && (
        <div className="flex flex-col gap-3">
          {projects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}
    </div>
  )
}

function ProjectCard({ project }: { project: ProposalResponse }) {
  const tp = useTranslations("proposal")

  return (
    <Link
      href={`/projects/${project.id}`}
      className={cn(
        "flex items-center gap-5 rounded-2xl border bg-white px-6 py-5",
        "transition-all duration-200",
        "border-gray-100 shadow-sm",
        "hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5",
        "dark:bg-gray-800/80 dark:border-gray-700 dark:hover:border-rose-500/30",
      )}
    >
      {/* Status badge */}
      <div className="shrink-0">
        <ProjectStatusBadge status={project.status} />
      </div>

      {/* Title + description */}
      <div className="min-w-0 flex-1">
        <h3 className="text-sm font-bold text-gray-900 dark:text-white truncate">
          {project.title}
        </h3>
        <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400 line-clamp-1">
          {project.description}
        </p>
      </div>

      {/* Amount */}
      <div className="shrink-0 text-right">
        <p className="text-sm font-bold text-gray-900 dark:text-white">
          {formatCurrency(project.amount / 100)}
        </p>
        {project.deadline && (
          <p className="mt-0.5 flex items-center justify-end gap-1 text-xs text-gray-500 dark:text-gray-400">
            <Calendar className="h-3 w-3" strokeWidth={1.5} />
            {new Intl.DateTimeFormat("fr-FR", {
              day: "numeric",
              month: "short",
            }).format(new Date(project.deadline))}
          </p>
        )}
      </div>

      {/* Arrow */}
      <div className="shrink-0">
        <ArrowRight className="h-4 w-4 text-gray-300 dark:text-gray-600" strokeWidth={1.5} />
      </div>
    </Link>
  )
}

function ProjectStatusBadge({ status }: { status: ProposalStatus }) {
  const t = useTranslations("proposal")

  const config: Record<string, { label: string; icon: React.ElementType; className: string }> = {
    pending: {
      label: t("pending"),
      icon: Clock,
      className: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
    },
    accepted: {
      label: t("accepted"),
      icon: CheckCircle2,
      className: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400",
    },
    paid: {
      label: t("paid"),
      icon: DollarSign,
      className: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
    },
    active: {
      label: t("active"),
      icon: Star,
      className: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
    },
    completion_requested: {
      label: t("completionRequested"),
      icon: Clock,
      className: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
    },
    completed: {
      label: t("completed"),
      icon: CheckCircle2,
      className: "bg-gray-50 text-gray-600 dark:bg-gray-500/10 dark:text-gray-400",
    },
  }

  const entry = config[status]
  if (!entry) return null

  const Icon = entry.icon

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium",
        entry.className,
      )}
    >
      <Icon className="h-3.5 w-3.5" strokeWidth={2} />
      {entry.label}
    </span>
  )
}

function ProjectsSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {[1, 2, 3].map((i) => (
        <div
          key={i}
          className="flex items-center gap-5 rounded-2xl border border-gray-100 bg-white px-6 py-5 dark:border-gray-700 dark:bg-gray-800/80"
        >
          <div className="h-6 w-20 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
          <div className="flex-1 space-y-1.5">
            <div className="h-4 w-3/4 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
          </div>
          <div className="space-y-1.5 text-right">
            <div className="h-4 w-24 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-3 w-16 animate-pulse rounded bg-gray-100 dark:bg-gray-700 ml-auto" />
          </div>
        </div>
      ))}
    </div>
  )
}
