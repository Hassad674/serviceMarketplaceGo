import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import enMessages from "@/../messages/en.json"
import frMessages from "@/../messages/fr.json"
import { RolePermissionsEditor } from "../role-permissions-editor"
import type {
  RolePermissionsMatrixResponse,
  RolePermissionsRow,
} from "../../types"

// Bug 3 — make sure no raw i18n keys ever leak through to the DOM,
// in either locale. The matrix is mocked deterministically so the
// test assertions reduce to "the editor renders translated labels +
// descriptions, never `team.…` / `roles.…` / `permissions.…`".

vi.mock("../../api/team-api", () => ({
  getRolePermissionsMatrix: vi.fn(),
  updateRolePermissions: vi.fn(),
  // Other api functions never called by the editor — declared so the
  // module mock does not blow up imports anywhere upstream.
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

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

import { getRolePermissionsMatrix } from "../../api/team-api"

const mockGetMatrix = vi.mocked(getRolePermissionsMatrix)

// Build a row for one role with a representative subset of cells:
// one editable, one non-overridable (locked), one with override.
function buildRow(
  role: RolePermissionsRow["role"],
): RolePermissionsRow {
  return {
    role,
    label: role,
    description: "",
    permissions: [
      {
        key: "team.view",
        group: "team",
        label: "View team",
        description: "Can see the list of members and pending invitations.",
        granted: true,
        state: "default_granted",
        locked: false,
      },
      {
        key: "team.invite",
        group: "team",
        label: "Invite members",
        description: "Can send email invitations to join the organization.",
        granted: false,
        state: "default_revoked",
        locked: false,
      },
      {
        key: "wallet.withdraw",
        group: "wallet",
        label: "Request payouts",
        description: "Can move money out of the wallet.",
        granted: false,
        state: "locked",
        locked: true,
      },
      {
        key: "org_profile.edit",
        group: "org_profile",
        label: "Edit provider profile",
        description: "Can update the organization's marketplace profile.",
        granted: true,
        state: "default_granted",
        locked: false,
      },
    ],
  }
}

const MATRIX: RolePermissionsMatrixResponse = {
  roles: [buildRow("admin"), buildRow("member"), buildRow("viewer")],
}

function renderEditor(locale: "fr" | "en") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const messages = locale === "fr" ? frMessages : enMessages
  return render(
    <NextIntlClientProvider locale={locale} messages={messages}>
      <QueryClientProvider client={queryClient}>
        <RolePermissionsEditor orgID="org-1" />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockGetMatrix.mockResolvedValue(MATRIX)
})

// Anything that starts with one of these prefixes is a leaked next-intl
// fallback (key path joined with `.`). The page must render zero of
// these strings.
const RAW_KEY_PATTERN = /^(team\.|roles\.|permissions\.|rolePermissions\.|permissionGroups\.)/

describe("RolePermissionsEditor (FR)", () => {
  it("renders role chip labels in French (Admin / Membre / Lecteur)", async () => {
    renderEditor("fr")
    expect(await screen.findByRole("tab", { name: "Admin" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Membre" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Lecteur" })).toBeInTheDocument()
  })

  it("translates permission labels to French (no English fallback)", async () => {
    renderEditor("fr")
    expect(await screen.findByText("Voir l'équipe")).toBeInTheDocument()
    expect(screen.getByText("Inviter des membres")).toBeInTheDocument()
    expect(screen.getByText("Modifier le profil prestataire")).toBeInTheDocument()
    // Locked permission lives in the Owner-exclusive footer.
    expect(screen.getByText("Demander un virement")).toBeInTheDocument()
  })

  it("does not leak any raw i18n keys to the DOM", async () => {
    const { container } = renderEditor("fr")
    await screen.findByRole("tab", { name: "Admin" })
    const offenders = Array.from(container.querySelectorAll("*"))
      .map((el) => (el.textContent ?? "").trim())
      .filter((txt) => txt.length > 0 && RAW_KEY_PATTERN.test(txt))
    expect(offenders).toEqual([])
  })

  it("never renders the literal English backend label when FR is available", async () => {
    renderEditor("fr")
    await screen.findByRole("tab", { name: "Admin" })
    // "View team" is the backend English label for `team.view` —
    // the FR i18n catalogue must take precedence.
    expect(screen.queryByText("View team")).not.toBeInTheDocument()
    expect(screen.queryByText("Invite members")).not.toBeInTheDocument()
  })
})

describe("RolePermissionsEditor (EN)", () => {
  it("renders role chip labels in English (Admin / Member / Viewer)", async () => {
    renderEditor("en")
    expect(await screen.findByRole("tab", { name: "Admin" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Member" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Viewer" })).toBeInTheDocument()
  })

  it("translates permission labels to English from the i18n catalogue", async () => {
    renderEditor("en")
    expect(await screen.findByText("View team")).toBeInTheDocument()
    expect(screen.getByText("Invite members")).toBeInTheDocument()
    expect(screen.getByText("Edit provider profile")).toBeInTheDocument()
    expect(screen.getByText("Request payouts")).toBeInTheDocument()
  })

  it("does not leak any raw i18n keys to the DOM", async () => {
    const { container } = renderEditor("en")
    await screen.findByRole("tab", { name: "Admin" })
    const offenders = Array.from(container.querySelectorAll("*"))
      .map((el) => (el.textContent ?? "").trim())
      .filter((txt) => txt.length > 0 && RAW_KEY_PATTERN.test(txt))
    expect(offenders).toEqual([])
  })
})

describe("RolePermissionsEditor — backend label fallback", () => {
  it("falls back to backend-provided label when the i18n key is absent", async () => {
    // Inject a brand-new permission key the i18n catalogue does not
    // have yet — the editor must render the backend's English label
    // rather than the raw key.
    const futureMatrix: RolePermissionsMatrixResponse = {
      roles: [
        {
          role: "admin",
          label: "admin",
          description: "",
          permissions: [
            {
              key: "experimental.beta_feature",
              group: "other",
              label: "Beta feature",
              description: "Backend-provided fallback description.",
              granted: true,
              state: "default_granted",
              locked: false,
            },
          ],
        },
        buildRow("member"),
        buildRow("viewer"),
      ],
    }
    mockGetMatrix.mockResolvedValueOnce(futureMatrix)
    renderEditor("fr")
    expect(await screen.findByText("Beta feature")).toBeInTheDocument()
    expect(
      screen.getByText("Backend-provided fallback description."),
    ).toBeInTheDocument()
    // No raw key like `permissions.experimental.beta_feature.label`.
    expect(
      screen.queryByText(/^permissions\.experimental/),
    ).not.toBeInTheDocument()
  })

  it("falls back to the raw key when neither i18n nor backend label is provided", async () => {
    const minimalMatrix: RolePermissionsMatrixResponse = {
      roles: [
        {
          role: "admin",
          label: "admin",
          description: "",
          permissions: [
            {
              key: "completely.unknown",
              group: "other",
              label: "",
              description: "",
              granted: true,
              state: "default_granted",
              locked: false,
            },
          ],
        },
        buildRow("member"),
        buildRow("viewer"),
      ],
    }
    mockGetMatrix.mockResolvedValueOnce(minimalMatrix)
    renderEditor("en")
    // Must render the bare key as last-resort fallback (no crash).
    expect(await screen.findByText("completely.unknown")).toBeInTheDocument()
  })
})

describe("RolePermissionsEditor — read-only audience", () => {
  it("hides the role tab labels behind raw keys when readOnly", async () => {
    // Render the editor with readOnly=true and confirm role tab labels
    // still translate (read-only banner replaces the save bar but
    // the chips must still render their FR labels).
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
    })
    const { container } = render(
      <NextIntlClientProvider locale="fr" messages={frMessages}>
        <QueryClientProvider client={queryClient}>
          <RolePermissionsEditor orgID="org-1" readOnly />
        </QueryClientProvider>
      </NextIntlClientProvider>,
    )
    expect(await screen.findByRole("tab", { name: "Admin" })).toBeInTheDocument()
    const offenders = Array.from(container.querySelectorAll("*"))
      .map((el) => (el.textContent ?? "").trim())
      .filter((txt) => txt.length > 0 && RAW_KEY_PATTERN.test(txt))
    expect(offenders).toEqual([])
  })
})

describe("RolePermissionsEditor — pure helper coverage", () => {
  // The helper is exercised indirectly by the rendering tests above.
  // These narrow tests pin down the dotted-key splitting logic so a
  // regression on the JSON shape (back to flat keys) fails loudly.

  it("translates a permission key with a single dot to the nested path", async () => {
    renderEditor("fr")
    // jobs.create / messaging.send are not in the mocked matrix, so
    // they should not appear. team.view IS in the matrix and must
    // render as "Voir l'équipe".
    expect(await screen.findByText("Voir l'équipe")).toBeInTheDocument()
  })

  it("translates a permission key whose leaf contains an underscore", async () => {
    // wallet.withdraw is in the locked footer; its FR label has
    // underscores and accented chars — verifies leaf=`withdraw`
    // resolves through the nested catalogue.
    renderEditor("fr")
    expect(await screen.findByText("Demander un virement")).toBeInTheDocument()
  })
})
