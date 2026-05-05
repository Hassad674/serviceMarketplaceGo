"use client"

import Image from "next/image"
import { useState, useEffect, useCallback } from "react"
import { X, ChevronLeft, ChevronRight, Send, User, Flag } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { openChatWithOrg } from "@/shared/components/chat-widget/use-chat-widget"
import { ReportDialog } from "@/shared/components/reporting/report-dialog"
import type { ApplicationWithProfile } from "../types"
import { Button } from "@/shared/components/ui/button"

// W-08 candidate detail panel — Soleil v2.
// Right-side aside (desktop) / full-screen (<lg) modal panel. Top bar
// holds prev/next pager + report flag + close. Body uses Fraunces
// section heads and corail/soft tokens. Send-message becomes a
// corail-filled pill, view-profile a ghost-outline pill.

const ORG_PILL_CLASSES: Record<string, string> = {
  provider_personal: "bg-primary-soft text-primary-deep",
  agency: "bg-success-soft text-success",
  enterprise: "bg-amber-soft text-foreground",
}

function orgLabelKey(orgType: string): string {
  switch (orgType) {
    case "agency":
      return "jobDetail_w08_orgAgency"
    case "enterprise":
      return "jobDetail_w08_orgEnterprise"
    default:
      return "jobDetail_w08_orgFreelance"
  }
}

function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return "?"
  if (parts.length === 1) return parts[0][0].toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

function formatDateFr(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleDateString("fr-FR", {
    day: "numeric",
    month: "long",
    year: "numeric",
  })
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
  const tJob = useTranslations("job")
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
  const pillClass =
    ORG_PILL_CLASSES[profile.org_type] ?? "bg-border text-muted-foreground"

  function handleSendMessage() {
    if (isDesktop) {
      openChatWithOrg(application.applicant_id, displayName)
    } else {
      router.push(
        `/messages?to=${application.applicant_id}&name=${encodeURIComponent(displayName)}`,
      )
    }
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className={cn(
          "fixed inset-0 z-40 bg-foreground/40 backdrop-blur-sm transition-opacity duration-300",
          isOpen ? "opacity-100" : "pointer-events-none opacity-0",
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
          "flex flex-col bg-card",
          "border-l border-border rounded-l-2xl",
          "transition-transform duration-300 ease-out",
          isOpen ? "translate-x-0" : "translate-x-full",
        )}
        style={{ boxShadow: "var(--shadow-card-strong)" }}
      >
        {/* Top bar */}
        <div className="flex shrink-0 items-center justify-between border-b border-border px-5 py-3.5 sm:px-6">
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={navigatePrev}
              disabled={!hasPrev}
              aria-label={t("previousCandidate")}
              className={cn(
                "flex h-9 w-9 items-center justify-center rounded-full transition-colors",
                hasPrev
                  ? "text-muted-foreground hover:bg-primary-soft hover:text-primary-deep"
                  : "cursor-not-allowed text-subtle-foreground",
              )}
            >
              <ChevronLeft className="h-4 w-4" strokeWidth={1.7} />
            </Button>
            <span className="font-mono text-[11px] font-medium uppercase tracking-[0.06em] text-subtle-foreground">
              {currentIndex + 1} / {candidates.length}
            </span>
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={navigateNext}
              disabled={!hasNext}
              aria-label={t("nextCandidate")}
              className={cn(
                "flex h-9 w-9 items-center justify-center rounded-full transition-colors",
                hasNext
                  ? "text-muted-foreground hover:bg-primary-soft hover:text-primary-deep"
                  : "cursor-not-allowed text-subtle-foreground",
              )}
            >
              <ChevronRight className="h-4 w-4" strokeWidth={1.7} />
            </Button>
          </div>

          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setShowReportDialog(true)}
              aria-label={tReport("reportApplication")}
              className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-primary-soft hover:text-primary-deep"
            >
              <Flag className="h-4 w-4" strokeWidth={1.7} />
            </Button>
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-border hover:text-foreground"
            >
              <X className="h-4 w-4" strokeWidth={1.7} />
            </Button>
          </div>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 space-y-6 overflow-y-auto px-5 py-6 sm:px-6">
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
            {tJob("jobDetail_w08_panelEyebrow")}
          </p>

          {/* Identity row */}
          <div className="flex items-start gap-4">
            {profile.photo_url ? (
              <Image
                src={profile.photo_url}
                alt={displayName}
                width={56}
                height={56}
                className="h-14 w-14 shrink-0 rounded-full object-cover"
                style={{ boxShadow: "var(--shadow-portrait)" }}
              />
            ) : (
              <div
                className={cn(
                  "flex h-14 w-14 shrink-0 items-center justify-center rounded-full",
                  "bg-primary-soft text-[16px] font-semibold text-primary-deep",
                )}
                style={{ boxShadow: "var(--shadow-portrait)" }}
              >
                {initials}
              </div>
            )}
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="truncate font-serif text-[22px] font-medium leading-tight tracking-[-0.015em] text-foreground">
                  {displayName}
                </h2>
                <span
                  className={cn(
                    "inline-flex items-center rounded-full px-2.5 py-0.5",
                    "text-[10.5px] font-bold uppercase tracking-[0.05em]",
                    pillClass,
                  )}
                >
                  {tJob(orgLabelKey(profile.org_type))}
                </span>
              </div>
              {profile.title && (
                <p className="mt-1 text-[13.5px] text-muted-foreground">
                  {profile.title}
                </p>
              )}
              <p className="mt-2 font-mono text-[11px] uppercase tracking-[0.06em] text-subtle-foreground">
                {tJob("jobDetail_w08_appliedRelative", {
                  when: formatDateFr(application.created_at),
                })}
              </p>
            </div>
          </div>

          {/* Action row */}
          <div className="flex flex-wrap items-center gap-2.5">
            <Link
              href={`/freelancers/${application.applicant_id}`}
              className={cn(
                "inline-flex flex-1 items-center justify-center gap-1.5 rounded-full px-4 py-2.5",
                "border border-border-strong bg-card text-[13px] font-medium text-foreground",
                "transition-colors duration-150 hover:bg-primary-soft hover:text-primary-deep",
              )}
            >
              <User className="h-4 w-4" strokeWidth={1.7} />
              {t("viewProfile")}
            </Link>
            {canSendMessage && (
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={handleSendMessage}
                className={cn(
                  "inline-flex flex-1 items-center justify-center gap-1.5 rounded-full px-4 py-2.5",
                  "bg-primary text-[13px] font-semibold text-white",
                  "transition-colors duration-150 hover:bg-primary-deep active:scale-[0.98]",
                )}
                style={{ boxShadow: "var(--shadow-message)" }}
              >
                <Send className="h-4 w-4" strokeWidth={1.7} />
                {t("sendMessage")}
              </Button>
            )}
          </div>

          {/* Application message */}
          {application.message && (
            <PanelSection heading={tJob("jobDetail_w08_messageHeading")}>
              <div className="rounded-2xl border border-border bg-background p-4">
                <p className="overflow-wrap-anywhere whitespace-pre-wrap break-words text-[14px] leading-relaxed text-foreground">
                  {application.message}
                </p>
              </div>
            </PanelSection>
          )}

          {/* Application video */}
          {application.video_url && (
            <PanelSection heading={tJob("jobDetail_w08_videoHeading")}>
              <div className="aspect-video overflow-hidden rounded-2xl bg-foreground">
                <video
                  src={application.video_url}
                  controls
                  className="h-full w-full object-contain"
                  aria-label={`${displayName} - ${t("applicationVideo")}`}
                >
                  <track kind="captions" />
                </video>
              </div>
            </PanelSection>
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

function PanelSection({
  heading,
  children,
}: {
  heading: string
  children: React.ReactNode
}) {
  return (
    <section>
      <h3 className="mb-3 font-serif text-[16px] font-medium tracking-[-0.005em] text-foreground">
        {heading}
      </h3>
      {children}
    </section>
  )
}
