/**
 * CandidateDetailPanel — persona-aware "View profile" routing.
 *
 * Regression coverage for the bug where the right-side panel hardcoded
 * /freelancers/${applicant_id}, sending the enterprise to a 404. The
 * link must use profile.organization_id and the applicant_kind to pick
 * the correct route prefix (/agencies, /freelancers, /referrers).
 */
import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { CandidateDetailPanel } from "../candidate-detail-panel"
import type { ApplicantKind, ApplicationWithProfile } from "../../types"

const messages = {
  opportunity: {
    viewProfile: "View profile",
    sendMessage: "Send message",
    candidateDetails: "Candidate details",
    nextCandidate: "Next",
    previousCandidate: "Previous",
    applicationVideo: "Application video",
  },
  job: {
    jobDetail_w08_orgFreelance: "Freelance",
    jobDetail_w08_orgAgency: "Agence",
    jobDetail_w08_orgEnterprise: "Entreprise",
    jobDetail_w08_panelEyebrow: "ATELIER",
    jobDetail_w08_appliedRelative: "Postulé {when}",
    jobDetail_w08_messageHeading: "Message",
    jobDetail_w08_videoHeading: "Vidéo",
  },
  reporting: {
    reportApplication: "Report application",
  },
}

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children, className }: { href: string; children: React.ReactNode; className?: string }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock("@/shared/hooks/use-media-query", () => ({
  useMediaQuery: () => false,
}))

vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: () => true,
}))

vi.mock("@/shared/components/chat-widget/use-chat-widget", () => ({
  openChatWithOrg: vi.fn(),
}))

vi.mock("@/shared/components/reporting/report-dialog", () => ({
  ReportDialog: () => null,
}))

vi.mock("next/image", () => ({
  default: ({
    src,
    alt,
    width,
    height,
    className,
  }: {
    src: string
    alt: string
    width: number
    height: number
    className?: string
  }) => (
    // eslint-disable-next-line @next/next/no-img-element -- test mock for next/image
    <img src={src} alt={alt} width={width} height={height} className={className} />
  ),
}))

function createItem(
  applicantKind: ApplicantKind,
  organizationId: string,
  orgType: string,
): ApplicationWithProfile {
  return {
    application: {
      id: "app-1",
      job_id: "job-1",
      applicant_id: "user-id-NOT-org",
      applicant_kind: applicantKind,
      message: "Hello",
      created_at: "2026-04-01T00:00:00Z",
    },
    profile: {
      organization_id: organizationId,
      name: "Display",
      org_type: orgType,
      title: "",
      photo_url: "",
      referrer_enabled: false,
      average_rating: 0,
      review_count: 0,
    },
  } as ApplicationWithProfile
}

function renderPanel(item: ApplicationWithProfile) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <CandidateDetailPanel
        candidate={item}
        candidates={[item]}
        onClose={() => {}}
        onNavigate={() => {}}
        jobId="job-1"
      />
    </NextIntlClientProvider>,
  )
}

describe("CandidateDetailPanel view-profile link", () => {
  beforeEach(() => {
    document.body.style.overflow = ""
  })

  it("routes agency candidate to /agencies/<orgId>", () => {
    renderPanel(createItem("agency", "org-agency", "agency"))
    const link = screen.getByRole("link", { name: /View profile/i })
    expect(link).toHaveAttribute("href", "/agencies/org-agency")
  })

  it("routes freelance candidate to /freelancers/<orgId>", () => {
    renderPanel(createItem("freelance", "org-freelance", "provider_personal"))
    const link = screen.getByRole("link", { name: /View profile/i })
    expect(link).toHaveAttribute("href", "/freelancers/org-freelance")
  })

  it("routes referrer candidate to /referrers/<orgId>", () => {
    renderPanel(createItem("referrer", "org-referrer", "provider_personal"))
    const link = screen.getByRole("link", { name: /View profile/i })
    expect(link).toHaveAttribute("href", "/referrers/org-referrer")
  })

  it("never appends the applicant user id to the href", () => {
    renderPanel(createItem("freelance", "org-real", "provider_personal"))
    const href = screen
      .getByRole("link", { name: /View profile/i })
      .getAttribute("href")
    expect(href).toBe("/freelancers/org-real")
    expect(href).not.toContain("user-id-NOT-org")
  })
})
