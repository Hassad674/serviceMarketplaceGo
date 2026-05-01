/**
 * Pinning test for ReferrerProfileCard's avatar migration to
 * next/image. Mirrors the freelance-profile-card test so the
 * directory listing tile stays in sync visually.
 */
import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { ReferrerProfileCard } from "../referrer-profile-card"
import type { ReferrerProfile } from "../../api/referrer-profile-api"

const messages = {} as Record<string, unknown>
const LOCALE = "en"

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children, className }: { href: string; children: React.ReactNode; className?: string }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
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

function createProfile(overrides: Partial<ReferrerProfile> = {}): ReferrerProfile {
  return {
    id: "ref-1",
    organization_id: "org-1",
    title: "Connector",
    about: "",
    video_url: "",
    availability_status: "available",
    expertise_domains: [],
    photo_url: "",
    city: "Paris",
    country_code: "FR",
    latitude: null,
    longitude: null,
    work_mode: [],
    travel_radius_km: null,
    languages_professional: [],
    languages_conversational: [],
    pricing: null,
    created_at: "",
    updated_at: "",
    ...overrides,
  } as ReferrerProfile
}

function renderCard(profile: ReferrerProfile, displayName = "John Smith") {
  return render(
    <NextIntlClientProvider locale={LOCALE} messages={messages}>
      <ReferrerProfileCard profile={profile} displayName={displayName} />
    </NextIntlClientProvider>,
  )
}

describe("ReferrerProfileCard avatar", () => {
  it("renders an Image with width=48 height=48 when photo_url is set", () => {
    renderCard(createProfile({ photo_url: "https://cdn.example.com/john.jpg" }))
    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://cdn.example.com/john.jpg")
    expect(img).toHaveAttribute("width", "48")
    expect(img).toHaveAttribute("height", "48")
    expect(img).toHaveAttribute("alt", "John Smith")
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
