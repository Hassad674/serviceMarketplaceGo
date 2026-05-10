/**
 * Pinning test for CandidateCard's avatar migration to next/image.
 * Asserts the rendered <img> carries width/height props (next/image
 * mock substitutes a plain <img> for the assertion).
 *
 * Plus regression coverage for the persona-aware "View profile" routing
 * (org_id, never user_id) and the avatar onError fallback.
 */
import { describe, expect, it, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { CandidateCard } from "../candidate-card"
import type { ApplicantKind, ApplicationWithProfile } from "../../types"

const messages = {
  opportunity: {
    viewProfile: "View profile",
    sendMessage: "Send message",
  },
  job: {
    jobDetail_w08_orgFreelance: "Freelance",
    jobDetail_w08_orgAgency: "Agence",
    jobDetail_w08_orgReferrer: "Apporteur",
    jobDetail_w08_videoBadge: "Video",
    jobDetail_w08_appliedRelative: "Postulé {when}",
  },
}
const LOCALE = "en"

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

const openChatWithOrgMock = vi.fn()
vi.mock("@/shared/components/chat-widget/use-chat-widget", () => ({
  openChatWithOrg: (orgId: string, name: string) => openChatWithOrgMock(orgId, name),
}))

vi.mock("next/image", () => ({
  default: ({
    src,
    alt,
    width,
    height,
    className,
    onError,
  }: {
    src: string
    alt: string
    width: number
    height: number
    className?: string
    onError?: (e: unknown) => void
  }) => (
    // eslint-disable-next-line @next/next/no-img-element -- test mock substituting next/image
    <img src={src} alt={alt} width={width} height={height} className={className} onError={onError} />
  ),
}))

function createItem(
  overrides: {
    applicant_kind?: ApplicantKind
    profile?: Partial<ApplicationWithProfile["profile"]>
  } = {},
): ApplicationWithProfile {
  return {
    application: {
      id: "app-1",
      job_id: "job-1",
      applicant_id: "user-1",
      applicant_kind: overrides.applicant_kind ?? "freelance",
      message: "I'd love to apply",
      created_at: "2026-04-01T00:00:00Z",
    },
    profile: {
      organization_id: "org-1",
      name: "Jane Doe",
      org_type: "provider_personal",
      title: "Developer",
      photo_url: "",
      referrer_enabled: false,
      average_rating: 0,
      review_count: 0,
      ...overrides.profile,
    },
  } as ApplicationWithProfile
}

function renderCard(item: ApplicationWithProfile, options?: { isDesktop?: boolean }) {
  if (options?.isDesktop) {
    // Override the media query mock for this render only.
    vi.doMock("@/shared/hooks/use-media-query", () => ({
      useMediaQuery: () => true,
    }))
  }
  return render(
    <NextIntlClientProvider locale={LOCALE} messages={messages}>
      <CandidateCard item={item} />
    </NextIntlClientProvider>,
  )
}

describe("CandidateCard avatar", () => {
  it("renders an Image with width=40 height=40 when photo_url is set", () => {
    renderCard(createItem({ profile: { photo_url: "https://cdn.example.com/jane.jpg" } }))
    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://cdn.example.com/jane.jpg")
    expect(img).toHaveAttribute("width", "40")
    expect(img).toHaveAttribute("height", "40")
    expect(img.className).toMatch(/h-10/)
    expect(img.className).toMatch(/w-10/)
    expect(img.className).toMatch(/rounded-full/)
  })

  it("renders initials placeholder when photo_url is empty", () => {
    renderCard(createItem({ profile: { photo_url: "" } }))
    expect(screen.queryByRole("img")).toBeNull()
    expect(screen.getByText("JD")).toBeInTheDocument()
  })

  it("falls back to initials when next/image errors out", () => {
    renderCard(
      createItem({
        profile: { photo_url: "http://broken.example.com/missing.jpg", name: "Acme Corp" },
      }),
    )
    const img = screen.getByRole("img")
    fireEvent.error(img)
    // After the error, the image is hidden in favour of the initials disc.
    expect(screen.queryByRole("img")).toBeNull()
    expect(screen.getByText("AC")).toBeInTheDocument()
  })
})

describe("CandidateCard view-profile link", () => {
  it("routes agency candidates to /agencies/<orgId>", () => {
    renderCard(
      createItem({
        applicant_kind: "agency",
        profile: { organization_id: "org-agency", org_type: "agency" },
      }),
    )
    expect(screen.getByRole("link", { name: /View profile/i })).toHaveAttribute(
      "href",
      "/agencies/org-agency",
    )
  })

  it("routes freelance candidates to /freelancers/<orgId>", () => {
    renderCard(
      createItem({
        applicant_kind: "freelance",
        profile: { organization_id: "org-freelance" },
      }),
    )
    expect(screen.getByRole("link", { name: /View profile/i })).toHaveAttribute(
      "href",
      "/freelancers/org-freelance",
    )
  })

  it("routes referrer candidates to /referrers/<orgId>", () => {
    renderCard(
      createItem({
        applicant_kind: "referrer",
        profile: { organization_id: "org-referrer" },
      }),
    )
    expect(screen.getByRole("link", { name: /View profile/i })).toHaveAttribute(
      "href",
      "/referrers/org-referrer",
    )
  })

  it("uses the org id, NEVER the applicant user id, in the href", () => {
    // Regression: the previous implementation hardcoded
    // /freelancers/${application.applicant_id}, which sent the user
    // to a 404 because applicant_id is the user id, not the org id.
    renderCard(
      createItem({
        applicant_kind: "freelance",
        profile: { organization_id: "org-not-user" },
      }),
    )
    const href = screen
      .getByRole("link", { name: /View profile/i })
      .getAttribute("href")
    expect(href).toBe("/freelancers/org-not-user")
    expect(href).not.toContain("user-1")
  })
})
