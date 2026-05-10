"use client"

import { AlertCircle, AlertTriangle, ChevronRight, CheckCircle, Info } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import type { DashboardAction } from "../../types"

// ActionsTodoCard renders the "Actions à faire" widget — a stacked
// list of actionable rows (KYC, profile completion, billing, unread
// messages, …). Pure presentation: the parent computes the action
// list (typically by composing several feature hooks) and hands it
// down as props. This keeps the widget testable without mocking the
// auth / billing / messaging stores.

interface ActionsTodoCardProps {
  actions: DashboardAction[]
  isLoading?: boolean
}

const SEVERITY_TONE: Record<DashboardAction["severity"], string> = {
  info: "bg-muted text-muted-foreground",
  warning: "bg-amber-soft text-foreground",
  critical: "bg-primary-soft text-primary-deep",
}

const SEVERITY_ICON = {
  info: Info,
  warning: AlertTriangle,
  critical: AlertCircle,
} as const

export function ActionsTodoCard({ actions, isLoading }: ActionsTodoCardProps) {
  const t = useTranslations("dashboard.actions")

  return (
    <section
      aria-labelledby="actions-todo-heading"
      className="rounded-2xl border border-border bg-card p-5 shadow-card"
    >
      <header className="flex items-center justify-between">
        <h2
          id="actions-todo-heading"
          className="font-serif text-[18px] font-medium tracking-[-0.01em] text-foreground"
        >
          {t("title")}
        </h2>
        <span className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-muted-foreground">
          {actions.length === 0 ? t("countAllClear") : t("count", { count: actions.length })}
        </span>
      </header>
      <ActionsBody actions={actions} isLoading={Boolean(isLoading)} t={t} />
    </section>
  )
}

interface ActionsBodyProps {
  actions: DashboardAction[]
  isLoading: boolean
  t: ReturnType<typeof useTranslations>
}

function ActionsBody({ actions, isLoading, t }: ActionsBodyProps) {
  if (isLoading && actions.length === 0) {
    return (
      <ul className="mt-4 space-y-2" aria-label={t("loading")}>
        {Array.from({ length: 2 }).map((_, i) => (
          <li
            key={i}
            className="h-12 animate-pulse rounded-xl bg-muted/60"
            aria-hidden
          />
        ))}
      </ul>
    )
  }
  if (actions.length === 0) {
    return (
      <div className="mt-4 flex items-center gap-3 rounded-xl border border-dashed border-border bg-card p-4">
        <CheckCircle className="h-5 w-5 text-success" strokeWidth={1.5} aria-hidden />
        <p className="text-sm text-muted-foreground">{t("allClear")}</p>
      </div>
    )
  }
  return (
    <ul className="mt-4 space-y-2">
      {actions.map((action) => (
        <li key={action.id}>
          <ActionRow action={action} />
        </li>
      ))}
    </ul>
  )
}

function ActionRow({ action }: { action: DashboardAction }) {
  const Icon = SEVERITY_ICON[action.severity]
  return (
    <Link
      href={action.href}
      className={cn(
        "group flex items-center gap-3 rounded-xl border border-transparent px-3 py-3",
        "transition-colors duration-150",
        "hover:border-border hover:bg-muted/30",
        "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/15",
      )}
    >
      <span
        className={cn(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
          SEVERITY_TONE[action.severity],
        )}
      >
        <Icon className="h-4 w-4" strokeWidth={1.75} aria-hidden />
      </span>
      <span className="flex-1 text-sm text-foreground">{action.label}</span>
      <span className="hidden text-[12px] font-medium text-primary-deep sm:inline">
        {action.ctaLabel}
      </span>
      <ChevronRight
        className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5"
        aria-hidden
      />
    </Link>
  )
}
