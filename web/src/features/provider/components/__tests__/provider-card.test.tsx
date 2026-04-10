import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ProviderCard } from "../provider-card"
import type { PublicProfileSummary } from "../../api/search-api"

// Mock next-intl navigation (Link)
vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    ...rest
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

// Mock next/image
vi.mock("next/image", () => ({
  default: ({
    src,
    alt,
    ...rest
  }: {
    src: string
    alt: string
    width: number
    height: number
    className?: string
  }) => <img src={src} alt={alt} {...rest} />,
}))

function createProfile(
  overrides: Partial<PublicProfileSummary> = {},
): PublicProfileSummary {
  return {
    user_id: "user-123",
    display_name: "Acme Corp",
    first_name: "John",
    last_name: "Doe",
    role: "provider",
    title: "Full-Stack Developer",
    photo_url: "",
    referrer_enabled: false,
    average_rating: 0,
    review_count: 0,
    ...overrides,
  }
}

function renderProviderCard(
  profile: PublicProfileSummary,
  type: "freelancer" | "agency" | "referrer" = "freelancer",
) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProviderCard profile={profile} type={type} />
    </NextIntlClientProvider>,
  )
}

describe("ProviderCard", () => {
  it("renders display name for freelancer type", () => {
    const profile = createProfile()
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("John Doe")).toBeInTheDocument()
  })

  it("renders display name for agency type", () => {
    const profile = createProfile({ display_name: "Acme Agency" })
    renderProviderCard(profile, "agency")

    expect(screen.getByText("Acme Agency")).toBeInTheDocument()
  })

  it("renders title", () => {
    const profile = createProfile({ title: "React Expert" })
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("React Expert")).toBeInTheDocument()
  })

  it("shows 'No title' when title is empty", () => {
    const profile = createProfile({ title: "" })
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText(messages.search.noTitle)).toBeInTheDocument()
  })

  it("renders role badge with correct label for provider", () => {
    const profile = createProfile({ role: "provider" })
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("Freelancer")).toBeInTheDocument()
  })

  it("renders role badge with correct label for agency", () => {
    const profile = createProfile({ role: "agency" })
    renderProviderCard(profile, "agency")

    expect(screen.getByText("Agency")).toBeInTheDocument()
  })

  it("renders role badge with correct label for referrer type", () => {
    const profile = createProfile({ role: "provider" })
    renderProviderCard(profile, "referrer")

    expect(screen.getByText("Referrer")).toBeInTheDocument()
  })

  it("shows initials when no photo_url", () => {
    const profile = createProfile({
      first_name: "Jane",
      last_name: "Smith",
      photo_url: "",
    })
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("JS")).toBeInTheDocument()
  })

  it("shows single initial from display_name when no first/last name", () => {
    const profile = createProfile({
      first_name: "",
      last_name: "",
      display_name: "TechAgency",
      photo_url: "",
    })
    renderProviderCard(profile, "agency")

    expect(screen.getByText("T")).toBeInTheDocument()
  })

  it("renders photo when photo_url is provided", () => {
    const profile = createProfile({
      photo_url: "https://example.com/photo.jpg",
    })
    renderProviderCard(profile, "freelancer")

    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://example.com/photo.jpg")
  })

  it("links to correct freelancer public profile URL", () => {
    const profile = createProfile({ user_id: "abc-123" })
    renderProviderCard(profile, "freelancer")

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "/freelancers/abc-123")
  })

  it("links to correct agency public profile URL", () => {
    const profile = createProfile({ user_id: "agency-456" })
    renderProviderCard(profile, "agency")

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "/agencies/agency-456")
  })

  it("links to correct referrer public profile URL", () => {
    const profile = createProfile({ user_id: "ref-789" })
    renderProviderCard(profile, "referrer")

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "/referrers/ref-789")
  })
})
