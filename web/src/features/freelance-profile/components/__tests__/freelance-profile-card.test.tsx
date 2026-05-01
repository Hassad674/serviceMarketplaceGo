/**
 * Pinning test for FreelanceProfileCard's avatar migration to
 * next/image. Renders the directory tile with and without a photo
 * URL and asserts the resulting markup so the migration cannot be
 * silently undone.
 */
import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { FreelanceProfileCard } from "../freelance-profile-card"
import type { FreelanceProfile } from "../../api/freelance-profile-api"

const messages = {} as Record<string, unknown>
const LOCALE = "en"

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children, className }: { href: string; children: React.ReactNode; className?: string }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}))

// next/image is mocked to a plain <img> in unit tests so we can assert
// the rendered DOM without invoking the optimizer pipeline.
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

function createProfile(overrides: Partial<FreelanceProfile> = {}): FreelanceProfile {
  return {
    organization_id: "org-1",
    photo_url: "",
    title: "Developer",
    availability_status: "available",
    city: "Paris",
    country_code: "FR",
    work_mode: [],
    languages_professional: [],
    languages_conversational: [],
    pricing: null,
    ...overrides,
  } as FreelanceProfile
}

function renderCard(profile: FreelanceProfile, displayName = "Jane Doe") {
  return render(
    <NextIntlClientProvider locale={LOCALE} messages={messages}>
      <FreelanceProfileCard profile={profile} displayName={displayName} />
    </NextIntlClientProvider>,
  )
}

describe("FreelanceProfileCard avatar", () => {
  it("renders an Image with width=48 height=48 when photo_url is set", () => {
    renderCard(createProfile({ photo_url: "https://cdn.example.com/jane.jpg" }))
    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://cdn.example.com/jane.jpg")
    expect(img).toHaveAttribute("width", "48")
    expect(img).toHaveAttribute("height", "48")
    expect(img).toHaveAttribute("alt", "Jane Doe")
    expect(img.className).toMatch(/h-12/)
    expect(img.className).toMatch(/w-12/)
    expect(img.className).toMatch(/rounded-full/)
  })

  it("renders the placeholder div when photo_url is empty", () => {
    const { container } = renderCard(createProfile({ photo_url: "" }))
    expect(screen.queryByRole("img")).toBeNull()
    expect(container.querySelector(".rounded-full.bg-muted")).not.toBeNull()
  })
})
