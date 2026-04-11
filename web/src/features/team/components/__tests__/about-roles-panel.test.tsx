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
  it("starts collapsed", () => {
    renderPanel()
    expect(screen.queryByText(/Permissions granted/i)).not.toBeInTheDocument()
  })

  it("toggles expanded state when the header is clicked", async () => {
    const user = userEvent.setup()
    renderPanel()
    const toggle = screen.getByRole("button", { name: /About roles/i })
    expect(toggle).toHaveAttribute("aria-expanded", "false")

    await user.click(toggle)
    expect(toggle).toHaveAttribute("aria-expanded", "true")
  })

  it("renders one card per role with its permissions when expanded", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      expect(screen.getAllByText("Owner").length).toBeGreaterThan(0)
      expect(screen.getAllByText("Admin").length).toBeGreaterThan(0)
      expect(screen.getAllByText("Member").length).toBeGreaterThan(0)
      expect(screen.getAllByText("Viewer").length).toBeGreaterThan(0)
    })
    // Permission labels show through
    expect(screen.getAllByText("Invite members").length).toBeGreaterThan(0)
    expect(screen.getAllByText("Create jobs").length).toBeGreaterThan(0)
  })

  it("groups permissions by their resource family inside each card", async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole("button", { name: /About roles/i }))

    await waitFor(() => {
      // The "Team" group label appears at least once for the Owner/Admin
      // cards which both grant team.* permissions.
      expect(screen.getAllByText(/Team/i).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/Jobs/i).length).toBeGreaterThan(0)
    })
  })
})
