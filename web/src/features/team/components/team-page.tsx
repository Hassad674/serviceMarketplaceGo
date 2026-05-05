"use client"

import { useState } from "react"
import { UserPlus, LogOut, Users2, AlertTriangle } from "lucide-react"
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
import { RolePermissionsEditor } from "./role-permissions-editor"

// W-22 Team page — Soleil v2 visual port.
//
// Editorial header (TeamHeader: corail eyebrow + Fraunces italic-corail
// title + tabac subtitle), Soleil card sections for members/invitations,
// rounded-2xl ivoire surfaces, corail-soft accents for the transfer
// flow. ALL hooks/mutations/permission gates are unchanged — this is
// purely a visual identity refit.
//
// Permission gates (driven by organization.permissions string[]):
//   team.invite              -> "Inviter" CTA + invitation actions
//   team.manage              -> member row dropdown
//   team.transfer_ownership  -> Transfer ownership section
// member_role drives Leave (Owner cannot leave; must transfer first).
// Read-only mode of the role permissions editor is enforced backend-side
// at the service layer — this UI only reflects the gate.

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
  // Render a Soleil empty state.
  if (!organization || !orgID) {
    return (
      <div className="rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] p-12 text-center shadow-[var(--shadow-card)]">
        <span
          aria-hidden="true"
          className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
        >
          <Users2 className="h-6 w-6" strokeWidth={1.6} />
        </span>
        <h1 className="mt-4 font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
          {t("noOrgTitle")}
        </h1>
        <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
          {t("noOrgDescription")}
        </p>
      </div>
    )
  }

  const permissions = organization.permissions || []
  const canInvite = permissions.includes("team.invite")
  const canManage = permissions.includes("team.manage")
  const canTransfer = permissions.includes("team.transfer_ownership")
  const memberRole = organization.member_role
  const isOwner = memberRole === "owner"

  const members = membersData?.data ?? []
  const invitations = invitationsData?.data ?? []

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
    <div className="space-y-7">
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
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-mono text-[11px] font-bold uppercase tracking-[0.06em] text-[var(--subtle-foreground)]">
            {t("sections.members")}
          </h2>
          {canInvite && !transferIsPending && (
            <button
              type="button"
              onClick={() => setShowInviteModal(true)}
              className="inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)]"
            >
              <UserPlus className="h-3.5 w-3.5" strokeWidth={2} />
              {t("inviteButton")}
            </button>
          )}
        </div>
        <TeamMembersList orgID={orgID} members={members} canManage={canManage} />
      </section>

      {/* Roles and permissions — always rendered; the editor is read-only
          for non-Owners (and during a pending ownership transfer).
          Backend enforces Owner-only writes at the service layer (R17). */}
      <RolePermissionsEditor orgID={orgID} readOnly={!isOwner || transferIsPending} />

      {/* Pending invitations — only render when relevant */}
      {(invitations.length > 0 || canInvite) && (
        <section className="space-y-3">
          <h2 className="font-mono text-[11px] font-bold uppercase tracking-[0.06em] text-[var(--subtle-foreground)]">
            {t("sections.pendingInvitations")}
          </h2>
          <TeamInvitationsList
            orgID={orgID}
            invitations={invitations}
            canInvite={canInvite}
          />
        </section>
      )}

      {/* Ownership transfer — only when caller is Owner and no transfer pending */}
      {canTransfer && !transferIsPending && (
        <section className="rounded-2xl border border-[var(--primary-soft)] bg-gradient-to-br from-[var(--amber-soft)] to-[var(--primary-soft)] p-5 shadow-[var(--shadow-card)]">
          <div className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--surface)] text-[var(--primary)]"
            >
              <AlertTriangle className="h-5 w-5" strokeWidth={1.8} />
            </span>
            <div className="flex-1">
              <h2 className="font-serif text-[18px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
                {t("sections.transferOwnership")}
              </h2>
              <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
                {t("sections.transferOwnershipDescription")}
              </p>
              <button
                type="button"
                onClick={() => setShowTransferModal(true)}
                className="mt-4 inline-flex items-center gap-2 rounded-full border border-[var(--border-strong)] bg-[var(--surface)] px-4 py-2 text-[13px] font-semibold text-[var(--foreground)] transition-colors hover:bg-[var(--primary-soft)] hover:text-[var(--primary-deep)]"
              >
                {t("transferButton")}
              </button>
            </div>
          </div>
        </section>
      )}

      {/* Leave org — hidden for the Owner */}
      {!isOwner && (
        <section className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-5 shadow-[var(--shadow-card)]">
          <h2 className="font-serif text-[18px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
            {t("sections.leave")}
          </h2>
          <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
            {t("sections.leaveDescription")}
          </p>
          <button
            type="button"
            onClick={() => setShowLeaveDialog(true)}
            className="mt-4 inline-flex items-center gap-2 rounded-full border border-[var(--primary-soft)] bg-[var(--surface)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-deep)] transition-colors hover:bg-[var(--primary-soft)]"
          >
            <LogOut className="h-3.5 w-3.5" strokeWidth={2} />
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
