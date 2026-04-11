import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { ProviderCard } from "../provider-card"
import type { PublicProfileSummary, SearchType } from "../../api/search-api"

const messages = {
  search: {
    noTitle: "No title",
  },
}

vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    className,
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}))

vi.mock("next/image", () => ({
  default: ({ src, alt, ...rest }: {
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
    organization_id: "org-123",
    name: "John Doe",
    org_type: "provider_personal",
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
  type: SearchType = "freelancer",
) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProviderCard profile={profile} type={type} />
    </NextIntlClientProvider>,
  )
}

describe("ProviderCard", () => {
  it("renders the org name for freelancer type", () => {
    const profile = createProfile()
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("John Doe")).toBeInTheDocument()
  })

  it("renders the org name for agency type", () => {
    const profile = createProfile({ name: "Acme Agency", org_type: "agency" })
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

  it("renders the Freelancer badge for the freelancer directory", () => {
    const profile = createProfile()
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("Freelancer")).toBeInTheDocument()
  })

  it("renders the Agency badge for the agency directory", () => {
    const profile = createProfile({ org_type: "agency" })
    renderProviderCard(profile, "agency")

    expect(screen.getByText("Agency")).toBeInTheDocument()
  })

  it("renders the Referrer badge for the referrer directory", () => {
    const profile = createProfile()
    renderProviderCard(profile, "referrer")

    expect(screen.getByText("Referrer")).toBeInTheDocument()
  })

  it("shows two-character initials when the name has two words", () => {
    const profile = createProfile({
      name: "John Smith",
      photo_url: "",
    })
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("JS")).toBeInTheDocument()
  })

  it("shows a single initial when the name is one word", () => {
    const profile = createProfile({
      name: "TechAgency",
      photo_url: "",
      org_type: "agency",
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
    const profile = createProfile({ organization_id: "abc-123" })
    renderProviderCard(profile, "freelancer")

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "/freelancers/abc-123")
  })

  it("links to correct agency public profile URL", () => {
    const profile = createProfile({ organization_id: "agency-456", org_type: "agency" })
    renderProviderCard(profile, "agency")

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "/agencies/agency-456")
  })

  it("links to correct referrer public profile URL", () => {
    const profile = createProfile({ organization_id: "ref-789" })
    renderProviderCard(profile, "referrer")

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "/referrers/ref-789")
  })

  it("renders star rating when review_count > 0", () => {
    const profile = createProfile({
      average_rating: 4.7,
      review_count: 12,
    })
    renderProviderCard(profile, "freelancer")

    expect(screen.getByText("4.7")).toBeInTheDocument()
    expect(screen.getByText("(12)")).toBeInTheDocument()
  })

  it("hides rating when review_count is 0", () => {
    const profile = createProfile({ review_count: 0 })
    renderProviderCard(profile, "freelancer")

    expect(screen.queryByText("0.0")).not.toBeInTheDocument()
  })
})
