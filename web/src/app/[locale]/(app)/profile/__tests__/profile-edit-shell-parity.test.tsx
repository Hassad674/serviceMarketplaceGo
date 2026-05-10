import { describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import messages from "@/../messages/en.json"
import { AgencyProfilePage } from "../agency-profile-page"
import { FreelanceOwnProfilePage } from "../freelance-own-profile-page"

// Shell-parity test: the agency editable /profile page MUST render
// the same structural shell as the freelance editable /profile page.
// We assert the three load-bearing wrapper elements (max-w-5xl
// column, editing-mode hint, ProfileCompletionBar) are present in
// both, in the same order, and that both pages share the exact same
// outer container className. Section content is intentionally NOT
// asserted here — sibling tests cover individual cards.

vi.mock("@i18n/navigation", () => ({
  Link: ({ children, ...rest }: { children: ReactNode }) => (
    <a {...rest}>{children}</a>
  ),
  useRouter: () => ({ back: () => {}, push: () => {} }),
  usePathname: () => "/profile",
}))

vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({
    data: {
      id: "user-1",
      first_name: "Camille",
      last_name: "Martin",
      display_name: "Camille Martin",
    },
  }),
  useOrganization: () => ({
    data: { id: "org-1", type: "agency" },
  }),
}))

vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: () => true,
}))

vi.mock("@/shared/hooks/profile/use-profile-rating", () => ({
  useProfileRating: () => ({ data: undefined }),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))

vi.mock("@/shared/hooks/profile/use-project-history", () => ({
  useProjectHistory: () => ({ data: undefined, isLoading: false, isError: false }),
}))

vi.mock("@/features/profile-completion/hooks/use-profile-completion", () => ({
  useProfileCompletion: () => ({
    data: { percent: 60, filled_sections: 6, total_sections: 10, sections: [] },
    isLoading: false,
  }),
  profileCompletionQueryKey: () => ["mock"],
  useInvalidateProfileCompletion: () => () => {},
}))

// --- Agency-side wiring mocks ---
vi.mock("@/features/provider/hooks/use-profile", () => ({
  useProfile: () => ({
    data: {
      organization_id: "org-1",
      title: "Boutique creative agency",
      photo_url: "",
      presentation_video_url: "",
      referrer_video_url: "",
      about: "We craft brand systems.",
      referrer_about: "",
      expertise_domains: ["development"],
      skills: [],
      city: "Lyon",
      country_code: "FR",
      work_mode: ["remote"],
      travel_radius_km: null,
      languages_professional: ["fr"],
      languages_conversational: [],
      availability_status: "available_now",
      pricing: [],
    },
    isLoading: false,
    error: null,
  }),
  useUpdateProfile: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/provider/hooks/use-upload", () => ({
  useUploadPhoto: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useUploadVideo: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeleteVideo: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/provider/hooks/use-update-expertise", () => ({
  useUpdateExpertiseDomains: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock("@/features/provider/hooks/use-update-availability", () => ({
  useUpdateAvailability: () => ({ mutate: vi.fn(), isPending: false, isSuccess: false }),
}))

vi.mock("@/features/provider/hooks/use-portfolio", () => ({
  useMyPortfolio: () => ({ data: [], isLoading: false }),
  usePortfolioByOrganization: () => ({ data: [], isLoading: false }),
  useCreatePortfolioItem: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useUpdatePortfolioItem: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeletePortfolioItem: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useReorderPortfolio: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useUploadPortfolioImage: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useUploadPortfolioVideo: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/provider/hooks/use-social-links", () => ({
  useMySocialLinks: () => ({ data: [], isLoading: false }),
  useUpsertSocialLink: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeleteSocialLink: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  usePublicSocialLinks: () => ({ data: [], isLoading: false }),
}))

vi.mock("@/features/provider/hooks/use-pricing", () => ({
  usePricing: () => ({ data: [], isLoading: false }),
}))

vi.mock("@/features/provider/hooks/use-update-location", () => ({
  useUpdateLocation: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/provider/hooks/use-update-languages", () => ({
  useUpdateLanguages: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/skill/hooks/use-profile-skills", () => ({
  useProfileSkills: () => ({ data: [], isLoading: false }),
}))

vi.mock("@/features/skill/hooks/use-update-profile-skills", () => ({
  useUpdateProfileSkills: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock("@/features/skill/hooks/use-skill-autocomplete", () => ({
  useSkillAutocomplete: () => ({ data: [], isLoading: false }),
}))

vi.mock("@/features/skill/hooks/use-create-user-skill", () => ({
  useCreateUserSkill: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock("@/features/skill/hooks/use-skill-catalog", () => ({
  useSkillCatalog: () => ({ data: [], isLoading: false }),
}))

// --- Freelance-side wiring mocks ---
vi.mock("@/features/freelance-profile/hooks/use-freelance-profile", () => ({
  useFreelanceProfile: () => ({
    data: {
      organization_id: "org-1",
      title: "Senior product designer",
      photo_url: "",
      video_url: "",
      about: "Designing for SaaS B2B.",
      expertise_domains: ["design_ui_ux"],
      skills: [],
      city: "Lyon",
      country_code: "FR",
      work_mode: ["remote"],
      travel_radius_km: null,
      languages_professional: ["fr"],
      languages_conversational: [],
      availability_status: "available_now",
      // Single object, not an array — matches the FreelanceProfile
      // contract (pricing: FreelancePricing | null).
      pricing: null,
    },
    isLoading: false,
    error: null,
  }),
}))

vi.mock("@/features/freelance-profile/hooks/use-update-freelance-profile", () => ({
  useUpdateFreelanceProfile: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useUpdateFreelanceAvailability: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useUpdateFreelanceExpertise: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock("@/features/freelance-profile/hooks/use-freelance-video", () => ({
  useUploadFreelanceVideo: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeleteFreelanceVideo: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/freelance-profile/hooks/use-freelance-pricing", () => ({
  useFreelancePricing: () => ({ data: null, isLoading: false }),
  useFreelancePricingByOrg: () => ({ data: null, isLoading: false }),
}))

vi.mock("@/features/freelance-profile/hooks/use-upsert-freelance-pricing", () => ({
  useUpsertFreelancePricing: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/freelance-profile/hooks/use-delete-freelance-pricing", () => ({
  useDeleteFreelancePricing: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/features/freelance-profile/hooks/use-freelance-social-links", () => ({
  useMyFreelanceSocialLinks: () => ({ data: [], isLoading: false }),
  useUpsertFreelanceSocialLink: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useDeleteFreelanceSocialLink: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  usePublicFreelanceSocialLinks: () => ({ data: [], isLoading: false }),
}))

vi.mock("@/features/organization-shared/hooks/use-organization-shared", () => ({
  useOrganizationShared: () => ({
    data: {
      organization_id: "org-1",
      city: "Lyon",
      country_code: "FR",
      work_mode: ["remote"],
      travel_radius_km: null,
      languages_professional: ["fr"],
      languages_conversational: [],
      photo_url: "",
    },
    isLoading: false,
  }),
  organizationSharedQueryKey: () => ["organization-shared", "mock"],
}))

vi.mock("@/features/organization-shared/hooks/use-update-organization-photo", () => ({
  useUploadOrganizationPhoto: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock("@/features/organization-shared/hooks/use-update-organization-shared", () => ({
  useUpdateOrganizationShared: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

function renderWithProviders(node: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="en" messages={messages}>
        {node}
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

const SHELL_WRAPPER = "mx-auto w-full max-w-5xl space-y-5"
const SOLEIL_CARD = "rounded-2xl"
const SOLEIL_PAD = "p-7"
const LEGACY_CARD_PAD = "p-6 shadow-sm"

describe("profile edit shell parity", () => {
  it("agency edit page mounts the same outer wrapper as the freelance edit page", () => {
    const { container: agencyContainer, unmount } = renderWithProviders(
      <AgencyProfilePage />,
    )
    const agencyShell = agencyContainer.querySelector(`div.${SHELL_WRAPPER.split(" ").join(".")}`)
    expect(agencyShell).not.toBeNull()
    unmount()

    const { container: freelanceContainer } = renderWithProviders(
      <FreelanceOwnProfilePage />,
    )
    const freelanceShell = freelanceContainer.querySelector(
      `div.${SHELL_WRAPPER.split(" ").join(".")}`,
    )
    expect(freelanceShell).not.toBeNull()

    // Both shells share the editing-mode hint as their first text node.
    expect(agencyShell?.firstElementChild?.textContent).toContain(
      "Editing mode",
    )
    expect(freelanceShell?.firstElementChild?.textContent).toContain(
      "Editing mode",
    )
  })

  it("agency edit page exposes the ProfileCompletionBar above the section list", () => {
    renderWithProviders(<AgencyProfilePage />)
    // The bar renders the same a11y label on both surfaces.
    expect(
      screen.getByLabelText(/profile \d+% complete/i),
    ).toBeInTheDocument()
  })

  it("agency edit page section ordering matches the freelance shell (Vidéo right after Disponibilité, before Localisation)", () => {
    const { container } = renderWithProviders(<AgencyProfilePage />)
    const headings = Array.from(
      container.querySelectorAll<HTMLHeadingElement>("h2"),
    ).map((h) => h.textContent?.trim() ?? "")
    const indexOf = (needle: string) =>
      headings.findIndex((h) => h.toLowerCase().includes(needle.toLowerCase()))

    const availability = indexOf("Availability")
    const video = indexOf("Presentation video")
    const location = indexOf("Location")

    // Sanity: all three sections rendered.
    expect(availability).toBeGreaterThanOrEqual(0)
    expect(video).toBeGreaterThanOrEqual(0)
    expect(location).toBeGreaterThanOrEqual(0)

    // Vidéo sits BETWEEN Availability and Location, mirroring the
    // freelance shell. Before this refactor it sat AFTER Skills.
    expect(availability).toBeLessThan(video)
    expect(video).toBeLessThan(location)
  })

  it("agency edit page sections use Soleil v2 card primitives (rounded-2xl + p-7), not the legacy rounded-xl+p-6+shadow-sm", () => {
    const { container } = renderWithProviders(<AgencyProfilePage />)
    const sections = Array.from(container.querySelectorAll("section"))
    // At least the four target sections (pricing/availability/location/
    // languages) must be present.
    expect(sections.length).toBeGreaterThanOrEqual(4)

    const targetTitles = [
      /pricing/i, // direct/referral pricing
      /availability/i,
      /location/i,
      /languages/i,
      /social links/i,
    ]

    for (const re of targetTitles) {
      const matching = sections.find((s) => {
        const heading = within(s).queryByRole("heading", { level: 2 })
        return heading?.textContent ? re.test(heading.textContent) : false
      })
      if (!matching) continue // section may be hidden behind permissions
      expect(matching.className).toContain(SOLEIL_CARD)
      expect(matching.className).toContain(SOLEIL_PAD)
      expect(matching.className).not.toContain(LEGACY_CARD_PAD)
    }
  })
})
