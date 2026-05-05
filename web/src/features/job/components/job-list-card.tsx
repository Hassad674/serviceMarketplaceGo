"use client"

import { useState, useRef, useEffect } from "react"
import {
  Clock,
  Users,
  MoreVertical,
  Trash2,
  XCircle,
  Pencil,
  RotateCcw,
  Eye,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
import type { JobWithCountsResponse } from "../types"

// W-06 — Single job card on the entreprise listing. Soleil v2 anatomy:
//   - status pill (sapin-soft / mute) + italic mono relative date
//   - Fraunces title + tabac excerpt + skill chips
//   - dashed divider then mono budget pill, project type, applicants
//   - quick edit/view ghost icon buttons + kebab menu

export interface JobListCardProps {
  job: JobWithCountsResponse
  canEdit: boolean
  canDelete: boolean
  onClose: (id: string) => void
  onReopen: (id: string) => void
  onDelete: (id: string) => void
  isActing: boolean
}

export function JobListCard({
  job,
  canEdit,
  canDelete,
  onClose,
  onReopen,
  onDelete,
  isActing,
}: JobListCardProps) {
  const t = useTranslations("job")
  const router = useRouter()
  const isOpen = job.status === "open"
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    if (menuOpen) document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [menuOpen])

  return (
    <article
      onClick={() => router.push(`/jobs/${job.id}`)}
      onKeyDown={(e) => {
        if (e.target !== e.currentTarget) return
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault()
          router.push(`/jobs/${job.id}`)
        }
      }}
      role="link"
      tabIndex={0}
      className={cn(
        "group relative cursor-pointer rounded-2xl border border-border bg-card p-5 sm:p-6",
        "transition-all duration-200 ease-out",
        "hover:-translate-y-0.5 hover:border-border-strong",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <CardMeta job={job} isOpen={isOpen} />
          <h3 className="mt-2 truncate font-serif text-[20px] font-medium leading-snug tracking-[-0.015em] text-foreground sm:text-[22px]">
            {job.title}
          </h3>
          <p className="mt-2 line-clamp-2 max-w-2xl text-[14px] leading-relaxed text-muted-foreground">
            {job.description}
          </p>
        </div>

        <CardActions
          job={job}
          canEdit={canEdit}
          canDelete={canDelete}
          isOpen={isOpen}
          isActing={isActing}
          menuOpen={menuOpen}
          setMenuOpen={setMenuOpen}
          menuRef={menuRef}
          onClose={onClose}
          onReopen={onReopen}
          onDelete={onDelete}
        />
      </div>

      {job.skills.length > 0 && (
        <div className="mt-4 flex flex-wrap gap-1.5">
          {job.skills.slice(0, 6).map((skill) => (
            <span
              key={skill}
              className={cn(
                "inline-flex items-center rounded-full bg-background px-2.5 py-1",
                "text-[11.5px] font-medium text-muted-foreground",
              )}
            >
              {skill}
            </span>
          ))}
          {job.skills.length > 6 && (
            <span className="inline-flex items-center text-[11.5px] font-medium text-subtle-foreground">
              +{job.skills.length - 6}
            </span>
          )}
        </div>
      )}

      <div className="mt-5 flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-dashed border-border pt-4">
        <BudgetPill min={job.min_budget} max={job.max_budget} />
        <Separator />
        <span className="inline-flex items-center gap-1.5 text-[12.5px] text-muted-foreground">
          <Clock className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden="true" />
          {job.budget_type === "one_shot"
            ? t("oneShotShort")
            : t("longTermShort")}
        </span>
        <Separator />
        <ApplicantsBlock total={job.total_applicants} fresh={job.new_applicants} />
      </div>
    </article>
  )
}

// ─── Sub-pieces ───────────────────────────────────────────────────

function CardMeta({
  job,
  isOpen,
}: {
  job: JobWithCountsResponse
  isOpen: boolean
}) {
  const t = useTranslations("job")
  const dateLabel = formatRelative(job.created_at, t)
  const closedLabel =
    !isOpen && job.closed_at ? formatRelative(job.closed_at, t) : null
  return (
    <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
      <StatusPill isOpen={isOpen} />
      <span className="font-mono text-[10.5px] font-medium uppercase tracking-[0.06em] text-subtle-foreground">
        {closedLabel
          ? t("closedRelative", { when: closedLabel })
          : t("publishedRelative", { when: dateLabel })}
      </span>
    </div>
  )
}

function StatusPill({ isOpen }: { isOpen: boolean }) {
  const t = useTranslations("job")
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5",
        "text-[11px] font-bold leading-tight",
        isOpen
          ? "bg-success-soft text-success"
          : "bg-border text-muted-foreground",
      )}
    >
      {isOpen ? t("statusOpen") : t("statusClosed")}
    </span>
  )
}

function BudgetPill({ min, max }: { min: number; max: number }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full bg-background px-2.5 py-1",
        "font-mono text-[12px] font-semibold text-foreground",
      )}
    >
      {min.toLocaleString("fr-FR")}&nbsp;€&nbsp;–&nbsp;
      {max.toLocaleString("fr-FR")}&nbsp;€
    </span>
  )
}

function ApplicantsBlock({
  total,
  fresh,
}: {
  total: number
  fresh: number
}) {
  const t = useTranslations("job")
  if (total === 0) return null
  const label = total === 1 ? t("applicantsLabelOne") : t("applicantsLabel")
  return (
    <span className="inline-flex items-center gap-1.5 text-[12.5px] text-muted-foreground">
      <Users className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden="true" />
      <span>
        <span className="font-semibold text-foreground">{total}</span> {label}
      </span>
      {fresh > 0 && (
        <span
          className={cn(
            "inline-flex items-center rounded-full bg-primary-soft px-2 py-0.5",
            "text-[10.5px] font-bold text-primary",
          )}
        >
          {t("applicantsNew", { count: fresh })}
        </span>
      )}
    </span>
  )
}

function Separator() {
  return (
    <span
      aria-hidden="true"
      className="inline-block h-1 w-1 rounded-full bg-border-strong"
    />
  )
}

// ─── Card actions (icon row + kebab) ──────────────────────────────

interface CardActionsProps {
  job: JobWithCountsResponse
  canEdit: boolean
  canDelete: boolean
  isOpen: boolean
  isActing: boolean
  menuOpen: boolean
  setMenuOpen: (open: boolean) => void
  menuRef: React.RefObject<HTMLDivElement | null>
  onClose: (id: string) => void
  onReopen: (id: string) => void
  onDelete: (id: string) => void
}

function CardActions({
  job,
  canEdit,
  canDelete,
  isOpen,
  isActing,
  menuOpen,
  setMenuOpen,
  menuRef,
  onClose,
  onReopen,
  onDelete,
}: CardActionsProps) {
  const t = useTranslations("job")
  const router = useRouter()
  return (
    <div className="flex shrink-0 items-center gap-1.5">
      {canEdit && (
        <Button
          variant="ghost"
          size="auto"
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            router.push(`/jobs/${job.id}/edit`)
          }}
          aria-label={t("editJob")}
          className={cn(
            "hidden h-9 w-9 items-center justify-center rounded-full",
            "text-muted-foreground transition-colors duration-150",
            "hover:bg-primary-soft hover:text-primary sm:inline-flex",
          )}
        >
          <Pencil className="h-4 w-4" strokeWidth={1.7} />
        </Button>
      )}
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          router.push(`/jobs/${job.id}`)
        }}
        aria-label={t("viewDetail")}
        className={cn(
          "hidden h-9 w-9 items-center justify-center rounded-full",
          "text-muted-foreground transition-colors duration-150",
          "hover:bg-primary-soft hover:text-primary sm:inline-flex",
        )}
      >
        <Eye className="h-4 w-4" strokeWidth={1.7} />
      </Button>
      {(canEdit || canDelete) && (
        <div className="relative" ref={menuRef}>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              setMenuOpen(!menuOpen)
            }}
            aria-haspopup="menu"
            aria-expanded={menuOpen}
            className={cn(
              "inline-flex h-9 w-9 items-center justify-center rounded-full",
              "text-muted-foreground transition-colors duration-150",
              "hover:bg-border hover:text-foreground",
            )}
          >
            <MoreVertical className="h-4 w-4" strokeWidth={1.7} />
          </Button>
          {menuOpen && (
            <div
              role="menu"
              className={cn(
                "absolute right-0 top-full z-10 mt-2 w-48 overflow-hidden rounded-2xl",
                "border border-border bg-card py-1 text-sm",
              )}
              style={{ boxShadow: "var(--shadow-card-strong)" }}
              onClick={(e) => e.stopPropagation()}
            >
              {canEdit && (
                <KebabItem
                  icon={<Pencil className="h-4 w-4" strokeWidth={1.7} />}
                  label={t("editJob")}
                  onClick={() => {
                    setMenuOpen(false)
                    router.push(`/jobs/${job.id}/edit`)
                  }}
                />
              )}
              {canEdit &&
                (isOpen ? (
                  <KebabItem
                    icon={<XCircle className="h-4 w-4" strokeWidth={1.7} />}
                    label={t("closeJob")}
                    tone="warning"
                    disabled={isActing}
                    onClick={() => {
                      setMenuOpen(false)
                      onClose(job.id)
                    }}
                  />
                ) : (
                  <KebabItem
                    icon={<RotateCcw className="h-4 w-4" strokeWidth={1.7} />}
                    label={t("reopenJob")}
                    tone="success"
                    disabled={isActing}
                    onClick={() => {
                      setMenuOpen(false)
                      onReopen(job.id)
                    }}
                  />
                ))}
              {canDelete && (
                <KebabItem
                  icon={<Trash2 className="h-4 w-4" strokeWidth={1.7} />}
                  label={t("deleteJob")}
                  tone="destructive"
                  disabled={isActing}
                  onClick={() => {
                    setMenuOpen(false)
                    onDelete(job.id)
                  }}
                />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

interface KebabItemProps {
  icon: React.ReactNode
  label: string
  tone?: "default" | "warning" | "success" | "destructive"
  disabled?: boolean
  onClick: () => void
}

function KebabItem({
  icon,
  label,
  tone = "default",
  disabled = false,
  onClick,
}: KebabItemProps) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      role="menuitem"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "flex w-full items-center gap-2 px-3.5 py-2 text-left text-[13.5px] font-medium",
        "transition-colors duration-150",
        tone === "destructive" && "text-destructive hover:bg-destructive/10",
        tone === "warning" && "text-warning hover:bg-amber-soft",
        tone === "success" && "text-success hover:bg-success-soft",
        tone === "default" && "text-foreground hover:bg-primary-soft",
      )}
    >
      {icon}
      {label}
    </Button>
  )
}

// ─── Helpers ──────────────────────────────────────────────────────

type JobTranslator = ReturnType<typeof useTranslations<"job">>

function formatRelative(dateStr: string, t: JobTranslator): string {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ""
  const diffMs = Date.now() - date.getTime()
  const minutes = Math.max(0, Math.floor(diffMs / 60_000))
  if (minutes < 1) return t("justNow")
  if (minutes < 60) return t("minutesAgo", { count: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return t("hoursAgo", { count: hours })
  const days = Math.floor(hours / 24)
  if (days < 7) return t("daysAgo", { count: days })
  const weeks = Math.floor(days / 7)
  return t("weeksAgo", { count: weeks })
}
