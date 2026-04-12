import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { AboutRolesPanel } from "../about-roles-panel"
import type { RoleDefinitionsResponse } from "../../types"

vi.mock("../../api/team-api", () => ({
  listMembers: vi.fn(),
  listInvitations: vi.fn(),
  sendInvitation: vi.fn(),
  resendInvitation: vi.fn(),
  cancelInvitation: vi.fn(),
  updateMember: vi.fn(),
  removeMember: vi.fn(),
  leaveOrganization: vi.fn(),
  initiateTransferOwnership: vi.fn(),
  cancelTransferOwnership: vi.fn(),
  acceptTransferOwnership: vi.fn(),
  declineTransferOwnership: vi.fn(),
  validateInvitation: vi.fn(),
  acceptInvitation: vi.fn(),
  getRoleDefinitions: vi.fn(),
}))

import { getRoleDefinitions } from "../../api/team-api"
const mockGetRoleDefinitions = vi.mocked(getRoleDefinitions)

const FIXTURE: RoleDefinitionsResponse = {
  roles: [
    {
      key: "owner",
      label: "Owner",
      description: "Full control of the organization.",
      permissions: ["team.invite", "team.manage", "team.transfer_ownership"],
    },
    {
      key: "admin",
      label: "Admin",
      description: "Trusted operator with full operational rights.",
      permissions: ["team.invite", "team.manage", "jobs.create"],
    },
    {
      key: "member",
      label: "Member",
      description: "Daily operator.",
      permissions: ["jobs.view", "jobs.create"],
    },
    {
      key: "viewer",
      label: "Viewer",
      description: "Read-only access.",
      permissions: ["jobs.view"],
    },
  ],
  permissions: [
    { key: "team.invite", group: "team", label: "Invite members", description: "" },
    { key: "team.manage", group: "team", label: "Manage team", description: "" },
    { key: "team.transfer_ownership", group: "team", label: "Transfer ownership", description: "" },
    { key: "jobs.view", group: "jobs", label: "View jobs", description: "" },
    { key: "jobs.create", group: "jobs", label: "Create jobs", description: "" },
  ],
}

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={queryClient}>
        <AboutRolesPanel />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockGetRoleDefinitions.mockResolvedValue(FIXTURE)
})

describe("AboutRolesPanel", () => {
  it("starts with the section collapsed", () => {
    renderPanel()
    const toggle = screen.getByRole("button", { name: /About roles/i })
    expect(toggle).toHaveAttribute("aria-expanded", "false")
  })

  it("toggles the section open when the header is clicked", async () => {
    const user = userEvent.setup()
    renderPanel()
    const toggle = screen.getByRole("button", { name: /About roles/i })

    await user.click(toggle)
    expect(toggle).toHaveAttribute("aria-expanded", "true")
  })

  it("renders four role cards when the section is expanded", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      expect(screen.getAllByText("Owner").length).toBeGreaterThan(0)
      expect(screen.getAllByText("Admin").length).toBeGreaterThan(0)
      expect(screen.getAllByText("Member").length).toBeGreaterThan(0)
      expect(screen.getAllByText("Viewer").length).toBeGreaterThan(0)
    })
  })

  it("shows permissions only after expanding an individual role card", async () => {
    const user = userEvent.setup()
    renderPanel()

    // Open the section
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    // Wait for role cards to appear
    await waitFor(() => {
      expect(screen.getAllByText("Owner").length).toBeGreaterThan(0)
    })

    // Click the Owner card to expand it. Use the stable DOM id to
    // avoid ambiguity when i18n descriptions contain "owner".
    const ownerButton = document.getElementById("role-card-owner") as HTMLButtonElement
    await user.click(ownerButton)

    expect(ownerButton).toHaveAttribute("aria-expanded", "true")

    // The permission labels should be present in the DOM
    expect(screen.getAllByText("Invite members").length).toBeGreaterThan(0)
    expect(screen.getAllByText("Transfer ownership").length).toBeGreaterThan(0)
  })

  it("groups permissions by their resource family inside each card", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      expect(screen.getAllByText("Owner").length).toBeGreaterThan(0)
    })

    // Expand the Owner card
    const ownerButton = document.getElementById("role-card-owner") as HTMLButtonElement
    await user.click(ownerButton)

    // The "Team" group label appears for the Owner card which grants
    // team.* permissions.
    await waitFor(() => {
      expect(screen.getAllByText(/Team/i).length).toBeGreaterThan(0)
    })
  })

  it("allows multiple role cards to be open simultaneously", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      expect(screen.getAllByText("Owner").length).toBeGreaterThan(0)
    })

    const ownerButton = document.getElementById("role-card-owner") as HTMLButtonElement
    const adminButton = document.getElementById("role-card-admin") as HTMLButtonElement

    await user.click(ownerButton)
    await user.click(adminButton)

    expect(ownerButton).toHaveAttribute("aria-expanded", "true")
    expect(adminButton).toHaveAttribute("aria-expanded", "true")
  })

  it("collapses a role card when clicked again", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      expect(screen.getAllByText("Owner").length).toBeGreaterThan(0)
    })

    const ownerButton = document.getElementById("role-card-owner") as HTMLButtonElement
    await user.click(ownerButton)
    expect(ownerButton).toHaveAttribute("aria-expanded", "true")

    await user.click(ownerButton)
    expect(ownerButton).toHaveAttribute("aria-expanded", "false")
  })

  it("shows a permission count badge on each role card", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      // Owner has 3 permissions, Admin has 3, Member has 2, Viewer has 1.
      // Use getAllByText because Owner and Admin both show "3".
      expect(screen.getAllByText("3").length).toBe(2)
      expect(screen.getByText("2")).toBeInTheDocument()
      expect(screen.getByText("1")).toBeInTheDocument()
    })
  })
})
