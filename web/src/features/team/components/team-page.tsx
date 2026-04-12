"use client"

import { useState } from "react"
import { UserPlus, LogOut, Users2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSession } from "@/shared/hooks/use-user"
import { useTeamMembers, useTeamInvitations } from "../hooks/use-team"
import { TeamHeader } from "./team-header"
import { TeamMembersList } from "./team-members-list"
import { TeamInvitationsList } from "./team-invitations-list"
import { InviteMemberModal } from "./invite-member-modal"
import { TransferOwnershipModal } from "./transfer-ownership-modal"
import { PendingTransferBanner } from "./pending-transfer-banner"
import { LeaveOrgDialog } from "./leave-org-dialog"
import { TeamPageSkeleton } from "./team-page-skeleton"
import { AboutRolesPanel } from "./about-roles-panel"
import { RolePermissionsEditor } from "./role-permissions-editor"

// Client-side entry point for /team. Pulls the session slice (which
// carries the current org + permissions + pending transfer) and
// wires the three list views + all the action modals.
//
// Permission gates (all driven by organization.permissions string[]
// which comes from the backend domain permission map):
//   team.invite              → "Inviter" button + invitation actions
//   team.manage              → member row dropdown (edit / remove)
//   team.transfer_ownership  → "Transférer l'ownership" section
//
// The "Quitter l'organisation" button is gated on member_role, not
// on a permission, because leaving is a self-action that the Owner
// is the only role forbidden to perform.

export function TeamPage() {
  const t = useTranslations("team")
  const { data: session, isLoading: sessionLoading } = useSession()

  const organization = session?.organization
  const orgID = organization?.id

  const { data: membersData, isLoading: membersLoading } = useTeamMembers(orgID)
  const { data: invitationsData, isLoading: invitationsLoading } = useTeamInvitations(orgID)

  const [showInviteModal, setShowInviteModal] = useState(false)
  const [showTransferModal, setShowTransferModal] = useState(false)
  const [showLeaveDialog, setShowLeaveDialog] = useState(false)

  if (sessionLoading || membersLoading || invitationsLoading) {
    return <TeamPageSkeleton />
  }

  // No org means: solo Provider, unprovisioned account, or logged out.
  // The sidebar link is role-gated so this is rare in practice —
  // still, render a clean empty state instead of crashing.
  if (!organization || !orgID) {
    return (
      <div className="rounded-xl border border-dashed border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-12 text-center">
        <Users2 className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
        <h1 className="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
          {t("noOrgTitle")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t("noOrgDescription")}
        </p>
      </div>
    )
  }

  const permissions = organization.permissions || []
  const canInvite = permissions.includes("team.invite")
  const canManage = permissions.includes("team.manage")
  const canTransfer = permissions.includes("team.transfer_ownership")
  const canManageRolePermissions = permissions.includes(
    "team.manage_role_permissions",
  )
  const memberRole = organization.member_role
  const isOwner = memberRole === "owner"

  const members = membersData?.data ?? []
  const invitations = invitationsData?.data ?? []

  // Pending transfer state — derived from the session payload which
  // carries pending_transfer_* on the organization slice (added to
  // OrganizationResponse in Phase 7). Undefined when no transfer is
  // in flight.
  const pendingTransferToUserID = organization.pending_transfer_to_user_id
  const pendingTransferExpiresAt = organization.pending_transfer_expires_at
  const currentUserID = session?.user?.id
  const transferIsPending = !!pendingTransferToUserID
  let pendingTransferBannerRole: "target" | "initiator" | null = null
  if (transferIsPending) {
    if (pendingTransferToUserID === currentUserID) {
      pendingTransferBannerRole = "target"
    } else if (isOwner) {
      pendingTransferBannerRole = "initiator"
    }
  }

  return (
    <div className="space-y-6">
      {pendingTransferBannerRole && (
        <PendingTransferBanner
          orgID={orgID}
          viewerRole={pendingTransferBannerRole}
          expiresAt={pendingTransferExpiresAt}
        />
      )}

      <TeamHeader
        organization={organization}
        memberCount={members.length}
        pendingInvitationCount={invitations.length}
      />

      {/* Members section + "Inviter" button when allowed */}
      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">
            {t("sections.members")}
          </h2>
          {canInvite && !transferIsPending && (
            <button
              type="button"
              onClick={() => setShowInviteModal(true)}
              className="inline-flex items-center gap-2 rounded-lg bg-rose-500 px-3.5 py-2 text-sm font-semibold text-white hover:bg-rose-600"
            >
              <UserPlus className="h-4 w-4" />
              {t("inviteButton")}
            </button>
          )}
        </div>
        <TeamMembersList orgID={orgID} members={members} canManage={canManage} />
      </section>

      {/* About roles — collapsible reference panel listing every
          role and its permissions. Always visible because every team
          member benefits from knowing what each role can do, even
          if they themselves can't manage the team. */}
      <AboutRolesPanel />

      {/* Role permissions editor — Owner-only UI to customize per-org
          role permissions (R17). The backend additionally enforces
          Owner-only at the service layer so a compromised frontend
          cannot bypass the gate. */}
      {canManageRolePermissions && !transferIsPending && (
        <RolePermissionsEditor orgID={orgID} />
      )}

      {/* Pending invitations — only visible if there is at least one
          or if the caller can invite (so Members/Viewers don't see
          an empty "pending invitations" block that serves them no
          purpose). */}
      {(invitations.length > 0 || canInvite) && (
        <section className="space-y-3">
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">
            {t("sections.pendingInvitations")}
          </h2>
          <TeamInvitationsList
            orgID={orgID}
            invitations={invitations}
            canInvite={canInvite}
          />
        </section>
      )}

      {/* Ownership transfer — only visible if the caller is the Owner
          and no transfer is already pending. */}
      {canTransfer && !transferIsPending && (
        <section className="rounded-xl border border-amber-200 dark:border-amber-500/30 bg-amber-50/50 dark:bg-amber-500/5 p-5">
          <h2 className="text-base font-semibold text-amber-900 dark:text-amber-100">
            {t("sections.transferOwnership")}
          </h2>
          <p className="mt-1 text-sm text-amber-800 dark:text-amber-200">
            {t("sections.transferOwnershipDescription")}
          </p>
          <button
            type="button"
            onClick={() => setShowTransferModal(true)}
            className="mt-4 inline-flex items-center gap-2 rounded-lg border border-amber-400 dark:border-amber-500/50 px-3.5 py-2 text-sm font-semibold text-amber-800 dark:text-amber-200 hover:bg-amber-100 dark:hover:bg-amber-500/10"
          >
            {t("transferButton")}
          </button>
        </section>
      )}

      {/* Leave org — hidden for the Owner (Owners must transfer first). */}
      {!isOwner && (
        <section className="rounded-xl border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-5">
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">
            {t("sections.leave")}
          </h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("sections.leaveDescription")}
          </p>
          <button
            type="button"
            onClick={() => setShowLeaveDialog(true)}
            className="mt-4 inline-flex items-center gap-2 rounded-lg border border-rose-200 dark:border-rose-500/30 px-3.5 py-2 text-sm font-semibold text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10"
          >
            <LogOut className="h-4 w-4" />
            {t("leaveButton")}
          </button>
        </section>
      )}

      {/* Modals */}
      <InviteMemberModal
        open={showInviteModal}
        onClose={() => setShowInviteModal(false)}
        orgID={orgID}
      />
      <TransferOwnershipModal
        open={showTransferModal}
        onClose={() => setShowTransferModal(false)}
        orgID={orgID}
        members={members}
        currentOwnerID={currentUserID || ""}
      />
      <LeaveOrgDialog
        open={showLeaveDialog}
        onClose={() => setShowLeaveDialog(false)}
        orgID={orgID}
      />
    </div>
  )
}
