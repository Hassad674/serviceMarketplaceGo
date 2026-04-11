import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { EditMemberModal } from "../edit-member-modal"
import type { TeamMember, RoleDefinitionsResponse } from "../../types"

// Mock the team-api module so the modal's hooks resolve against
// deterministic data instead of hitting the network.
vi.mock("../../api/team-api", () => ({
  listMembers: vi.fn(),
  listInvitations: vi.fn(),
  sendInvitation: vi.fn(),
  resendInvitation: vi.fn(),
  cancelInvitation: vi.fn(),
  updateMember: vi.fn(async () => ({})),
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

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

import { getRoleDefinitions, updateMember } from "../../api/team-api"

const mockGetRoleDefinitions = vi.mocked(getRoleDefinitions)
const mockUpdateMember = vi.mocked(updateMember)

const ROLE_DEFINITIONS: RoleDefinitionsResponse = {
  roles: [
    {
      key: "owner",
      label: "Owner",
      description: "Full control",
      permissions: ["team.invite", "team.manage", "team.transfer_ownership"],
    },
    {
      key: "admin",
      label: "Admin",
      description: "Trusted operator",
      permissions: ["team.invite", "team.manage"],
    },
    {
      key: "member",
      label: "Member",
      description: "Daily ops",
      permissions: ["jobs.create"],
    },
    {
      key: "viewer",
      label: "Viewer",
      description: "Read only",
      permissions: ["jobs.view"],
    },
  ],
  permissions: [
    { key: "team.invite", group: "team", label: "Invite members", description: "" },
    { key: "team.manage", group: "team", label: "Manage team", description: "" },
    { key: "team.transfer_ownership", group: "team", label: "Transfer ownership", description: "" },
    { key: "jobs.create", group: "jobs", label: "Create jobs", description: "" },
    { key: "jobs.view", group: "jobs", label: "View jobs", description: "" },
  ],
}

const memberFixture: TeamMember = {
  id: "m1",
  organization_id: "o1",
  user_id: "u1",
  role: "member",
  title: "Designer",
  joined_at: "2026-01-01T00:00:00Z",
  user: {
    id: "u1",
    email: "alice@example.com",
    display_name: "Alice Cooper",
    first_name: "Alice",
    last_name: "Cooper",
  },
}

function renderModal(member: TeamMember = memberFixture, onClose = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={queryClient}>
        <EditMemberModal
          open={true}
          onClose={onClose}
          orgID="o1"
          member={member}
        />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockGetRoleDefinitions.mockResolvedValue(ROLE_DEFINITIONS)
})

describe("EditMemberModal", () => {
  it("renders the member's display name in the title", () => {
    renderModal()
    expect(screen.getByText(/Edit Alice Cooper/)).toBeInTheDocument()
  })

  it("populates the role dropdown with admin/member/viewer (no owner)", () => {
    renderModal()
    const select = screen.getByLabelText(/Role/i)
    const optionValues = Array.from(
      (select as HTMLSelectElement).options,
    ).map((o) => o.value)
    expect(optionValues).toEqual(["admin", "member", "viewer"])
  })

  it("shows the inline permissions preview from the catalogue", async () => {
    renderModal()
    // Catalogue loads via TanStack Query — wait for the permission
    // labels to appear in the preview block.
    await waitFor(() => {
      expect(screen.getByText("Create jobs")).toBeInTheDocument()
    })
  })

  it("refreshes the preview when the user picks a different role", async () => {
    const user = userEvent.setup()
    renderModal()
    await waitFor(() => screen.getByText("Create jobs"))

    await user.selectOptions(screen.getByLabelText(/Role/i), "admin")

    await waitFor(() => {
      expect(screen.getByText("Manage team")).toBeInTheDocument()
      expect(screen.getByText("Invite members")).toBeInTheDocument()
    })
  })

  it("disables Save when no changes are made", () => {
    renderModal()
    expect(screen.getByRole("button", { name: /Save/i })).toBeDisabled()
  })

  it("submits role + title changes via updateMember and closes on success", async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    renderModal(memberFixture, onClose)
    await waitFor(() => screen.getByText("Create jobs"))

    await user.selectOptions(screen.getByLabelText(/Role/i), "admin")
    const titleInput = screen.getByLabelText(/Title/i)
    await user.clear(titleInput)
    await user.type(titleInput, "Lead designer")

    await user.click(screen.getByRole("button", { name: /Save/i }))

    await waitFor(() => {
      expect(mockUpdateMember).toHaveBeenCalledWith(
        "o1",
        "u1",
        expect.objectContaining({ role: "admin", title: "Lead designer" }),
      )
    })
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled()
    })
  })

  it("falls back to the generic name when the user record is missing", () => {
    const orphan: TeamMember = { ...memberFixture, user: undefined }
    renderModal(orphan)
    // English fallback from messages.team.memberFallbackName = "Member"
    expect(screen.getByText(/Edit Member/)).toBeInTheDocument()
  })
})
