"use client"

import { useState, useEffect, useCallback } from "react"
import { X, ChevronLeft, ChevronRight, Send, User, Flag } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { openChatWithOrg } from "@/shared/components/chat-widget/use-chat-widget"
import { ReportDialog } from "@/features/reporting/components/report-dialog"
import type { ApplicationWithProfile } from "../types"

const ORG_TYPE_COLORS: Record<string, string> = {
  provider_personal: "bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-400",
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
  enterprise: "bg-purple-50 text-purple-700 dark:bg-purple-500/10 dark:text-purple-400",
}

function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return "?"
  if (parts.length === 1) return parts[0][0].toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

interface CandidateDetailPanelProps {
  candidate: ApplicationWithProfile | null
  candidates: ApplicationWithProfile[]
  onClose: () => void
  onNavigate: (applicationId: string) => void
  jobId: string
}

export function CandidateDetailPanel({
  candidate,
  candidates,
  onClose,
  onNavigate,
  jobId: _jobId,
}: CandidateDetailPanelProps) {
  const t = useTranslations("opportunity")
  const tReport = useTranslations("reporting")
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")
  const canSendMessage = useHasPermission("messaging.send")
  const isOpen = candidate !== null
  const [showReportDialog, setShowReportDialog] = useState(false)

  const currentIndex = candidate
    ? candidates.findIndex((c) => c.application.id === candidate.application.id)
    : -1
  const hasPrev = currentIndex > 0
  const hasNext = currentIndex < candidates.length - 1

  const navigatePrev = useCallback(() => {
    if (hasPrev) onNavigate(candidates[currentIndex - 1].application.id)
  }, [hasPrev, candidates, currentIndex, onNavigate])

  const navigateNext = useCallback(() => {
    if (hasNext) onNavigate(candidates[currentIndex + 1].application.id)
  }, [hasNext, candidates, currentIndex, onNavigate])

  // Keyboard navigation
  useEffect(() => {
    if (!isOpen) return

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
      if (e.key === "ArrowLeft") navigatePrev()
      if (e.key === "ArrowRight") navigateNext()
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [isOpen, onClose, navigatePrev, navigateNext])

  // Lock body scroll when open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden"
    } else {
      document.body.style.overflow = ""
    }
    return () => {
      document.body.style.overflow = ""
    }
  }, [isOpen])

  if (!candidate) return null

  const { application, profile } = candidate
  const displayName = profile.name
  const initials = initialsFromName(displayName)

  function handleSendMessage() {
    if (isDesktop) {
      openChatWithOrg(application.applicant_id, displayName)
    } else {
      router.push(`/messages?to=${application.applicant_id}&name=${encodeURIComponent(displayName)}`)
    }
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className={cn(
          "fixed inset-0 z-40 bg-black/40 backdrop-blur-sm transition-opacity duration-300",
          isOpen ? "opacity-100" : "opacity-0 pointer-events-none",
        )}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <aside
        role="dialog"
        aria-label={t("candidateDetails")}
        aria-modal="true"
        className={cn(
          "fixed right-0 top-0 z-50 h-full",
          "w-full lg:w-[55%] lg:min-w-[480px] lg:max-w-[640px]",
          "bg-white dark:bg-slate-900 shadow-xl rounded-l-2xl",
          "flex flex-col",
          "transition-transform duration-300 ease-out",
          isOpen ? "translate-x-0" : "translate-x-full",
        )}
      >
        {/* Top bar */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100 dark:border-slate-700 shrink-0">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={navigatePrev}
              disabled={!hasPrev}
              aria-label={t("previousCandidate")}
              className={cn(
                "flex h-8 w-8 items-center justify-center rounded-lg transition-all",
                hasPrev
                  ? "text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
                  : "text-slate-300 dark:text-slate-600 cursor-not-allowed",
              )}
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="text-xs text-slate-400 dark:text-slate-500 tabular-nums">
              {currentIndex + 1} / {candidates.length}
            </span>
            <button
              type="button"
              onClick={navigateNext}
              disabled={!hasNext}
              aria-label={t("nextCandidate")}
              className={cn(
                "flex h-8 w-8 items-center justify-center rounded-lg transition-all",
                hasNext
                  ? "text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
                  : "text-slate-300 dark:text-slate-600 cursor-not-allowed",
              )}
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>

          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => setShowReportDialog(true)}
              aria-label={tReport("reportApplication")}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10 dark:hover:text-red-400 transition-all"
            >
              <Flag className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-all"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
          {/* Header: Avatar + Name + Org type + Title */}
          <div className="flex items-start gap-4">
            {profile.photo_url ? (
              // eslint-disable-next-line @next/next/no-img-element -- user-uploaded MinIO URL, see profile-header.tsx
              <img
                src={profile.photo_url}
                alt={displayName}
                className="h-14 w-14 shrink-0 rounded-full object-cover ring-2 ring-white dark:ring-slate-800 shadow-sm"
              />
            ) : (
              <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-rose-100 text-base font-semibold text-rose-700 dark:bg-rose-500/20 dark:text-rose-400 ring-2 ring-white dark:ring-slate-800 shadow-sm">
                {initials}
              </div>
            )}
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-bold text-slate-900 dark:text-white truncate">
                  {displayName}
                </h2>
                <span
                  className={cn(
                    "rounded-full px-2.5 py-0.5 text-[11px] font-medium shrink-0",
                    ORG_TYPE_COLORS[profile.org_type] ?? "bg-slate-100 text-slate-600",
                  )}
                >
                  {profile.org_type}
                </span>
              </div>
              {profile.title && (
                <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
                  {profile.title}
                </p>
              )}
            </div>
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-3">
            <Link
              href={`/freelancers/${application.applicant_id}`}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition-all",
                "border border-slate-200 text-slate-700 hover:bg-slate-50",
                "dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700",
              )}
            >
              <User className="h-4 w-4" />
              {t("viewProfile")}
            </Link>
            {canSendMessage && (
            <button
              type="button"
              onClick={handleSendMessage}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition-all",
                "bg-rose-500 text-white hover:bg-rose-600 shadow-sm hover:shadow-md",
                "active:scale-[0.98]",
              )}
            >
              <Send className="h-4 w-4" />
              {t("sendMessage")}
            </button>
            )}
          </div>

          {/* Application date */}
          <p className="text-xs text-slate-400 dark:text-slate-500">
            {t("applied")}{" "}
            {new Date(application.created_at).toLocaleDateString("fr-FR", {
              day: "numeric",
              month: "long",
              year: "numeric",
            })}
          </p>

          {/* Application message */}
          {application.message && (
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                {t("applicationMessage")}
              </h3>
              <div className="rounded-xl bg-slate-50 dark:bg-slate-800/60 p-4">
                <p className="text-sm text-slate-600 dark:text-slate-300 whitespace-pre-wrap break-words overflow-wrap-anywhere leading-relaxed">
                  {application.message}
                </p>
              </div>
            </div>
          )}

          {/* Video */}
          {application.video_url && (
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-2">
                {t("applicationVideo")}
              </h3>
              <div className="aspect-video overflow-hidden rounded-xl bg-black shadow-sm">
                <video
                  src={application.video_url}
                  controls
                  className="h-full w-full object-contain"
                  aria-label={`${displayName} - ${t("applicationVideo")}`}
                >
                  <track kind="captions" />
                </video>
              </div>
            </div>
          )}
        </div>

      </aside>

      {candidate && (
        <ReportDialog
          open={showReportDialog}
          onClose={() => setShowReportDialog(false)}
          targetType="application"
          targetId={candidate.application.id}
        />
      )}
    </>
  )
}
