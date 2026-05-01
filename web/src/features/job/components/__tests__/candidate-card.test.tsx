/**
 * Pinning test for CandidateCard's avatar migration to next/image.
 * Asserts the rendered <img> carries width/height props (next/image
 * mock substitutes a plain <img> for the assertion).
 */
import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { CandidateCard } from "../candidate-card"
import type { ApplicationWithProfile } from "../../types"

const messages = {
  opportunity: {
    viewProfile: "View profile",
    sendMessage: "Send message",
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

vi.mock("@/shared/components/chat-widget/use-chat-widget", () => ({
  openChatWithOrg: vi.fn(),
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
    // eslint-disable-next-line @next/next/no-img-element -- test mock substituting next/image
    <img src={src} alt={alt} width={width} height={height} className={className} />
  ),
}))

function createItem(overrides: Partial<ApplicationWithProfile["profile"]> = {}): ApplicationWithProfile {
  return {
    application: {
      id: "app-1",
      job_id: "job-1",
      applicant_id: "user-1",
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
      ...overrides,
    },
  } as ApplicationWithProfile
}

function renderCard(item: ApplicationWithProfile) {
  return render(
    <NextIntlClientProvider locale={LOCALE} messages={messages}>
      <CandidateCard item={item} />
    </NextIntlClientProvider>,
  )
}

describe("CandidateCard avatar", () => {
  it("renders an Image with width=40 height=40 when photo_url is set", () => {
    renderCard(createItem({ photo_url: "https://cdn.example.com/jane.jpg" }))
    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://cdn.example.com/jane.jpg")
    expect(img).toHaveAttribute("width", "40")
    expect(img).toHaveAttribute("height", "40")
    expect(img.className).toMatch(/h-10/)
    expect(img.className).toMatch(/w-10/)
    expect(img.className).toMatch(/rounded-full/)
  })

  it("renders initials placeholder when photo_url is empty", () => {
    renderCard(createItem({ photo_url: "" }))
    expect(screen.queryByRole("img")).toBeNull()
    expect(screen.getByText("JD")).toBeInTheDocument()
  })
})
