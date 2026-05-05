"use client"

import { useEffect, useRef, useState } from "react"
import { Crown, MoreVertical, Pencil, UserMinus } from "lucide-react"
import { useTranslations } from "next-intl"
import { Portrait } from "@/shared/components/ui/portrait"
import { EditMemberModal } from "./edit-member-modal"
import { RemoveMemberDialog } from "./remove-member-dialog"
import type { TeamMember } from "../types"

// Members list with permission-gated row actions. The "manage" permission
// is resolved upstream by the parent (team-page) so this component stays
// dumb — it just renders rows + opens dialogs.
//
// Layout in Soleil v2 is a vertical stack of rounded-2xl ivoire cards
// (NOT a table) — matches Notifications/Wallet/Invoices pattern: each
// row stands on its own with a soft sable border and calm shadow. The
// avatar is the Soleil Portrait primitive (deterministic palette by
// member id-hash), never initials.

type TeamMembersListProps = {
  orgID: string
  members: TeamMember[]
  canManage: boolean
}

export function TeamMembersList({ orgID, members, canManage }: TeamMembersListProps) {
  const t = useTranslations("team")
  const [editTarget, setEditTarget] = useState<TeamMember | null>(null)
  const [removeTarget, setRemoveTarget] = useState<TeamMember | null>(null)

  if (members.length === 0) {
    return (
      <div className="rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] p-10 text-center font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
        {t("emptyMembers")}
      </div>
    )
  }

  return (
    <>
      <ul className="space-y-2">
        {members.map((m, i) => (
          <li key={m.id}>
            <MemberRow
              member={m}
              portraitId={i}
              canManage={canManage}
              onEdit={() => setEditTarget(m)}
              onRemove={() => setRemoveTarget(m)}
            />
          </li>
        ))}
      </ul>

      {editTarget && (
        <EditMemberModal
          open={true}
          onClose={() => setEditTarget(null)}
          orgID={orgID}
          member={editTarget}
        />
      )}
      {removeTarget && (
        <RemoveMemberDialog
          open={true}
          onClose={() => setRemoveTarget(null)}
          orgID={orgID}
          member={removeTarget}
        />
      )}
    </>
  )
}

type MemberRowProps = {
  member: TeamMember
  portraitId: number
  canManage: boolean
  onEdit: () => void
  onRemove: () => void
}

function MemberRow({ member, portraitId, canManage, onEdit, onRemove }: MemberRowProps) {
  const t = useTranslations("team")
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const isOwner = member.role === "owner"
  const name =
    member.user?.display_name ||
    `${member.user?.first_name ?? ""} ${member.user?.last_name ?? ""}`.trim() ||
    t("memberFallbackName")

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false)
      }
    }
    if (menuOpen) {
      document.addEventListener("mousedown", handleClickOutside)
    }
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [menuOpen])

  return (
    <div className="flex items-center gap-4 rounded-2xl border border-[var(--border)] bg-[var(--surface)] px-4 py-3.5 shadow-[var(--shadow-card)] transition-colors hover:border-[var(--border-strong)]">
      <Portrait id={portraitId} size={44} alt={name} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="truncate font-serif text-[15px] font-medium text-[var(--foreground)]">
            {name}
          </p>
          <RoleBadge role={member.role} />
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-x-2 text-[12.5px]">
          {member.user?.email && (
            <span className="truncate text-[var(--muted-foreground)]">
              {member.user.email}
            </span>
          )}
          {member.title && (
            <>
              {member.user?.email && (
                <span aria-hidden="true" className="text-[var(--subtle-foreground)]">·</span>
              )}
              <span className="truncate font-serif italic text-[var(--muted-foreground)]">
                {member.title}
              </span>
            </>
          )}
        </div>
      </div>
      {canManage && !isOwner && (
        <div ref={menuRef} className="relative shrink-0">
          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            className="rounded-full p-1.5 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--background)] hover:text-[var(--foreground)]"
            aria-label={t("columns.actions")}
          >
            <MoreVertical className="h-4 w-4" />
          </button>
          {menuOpen && (
            <div className="absolute right-0 z-10 mt-1 min-w-[170px] overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--surface)] shadow-[var(--shadow-card-strong)]">
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false)
                  onEdit()
                }}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-[13px] font-medium text-[var(--foreground)] transition-colors hover:bg-[var(--background)]"
              >
                <Pencil className="h-3.5 w-3.5" strokeWidth={1.8} />
                {t("editAction")}
              </button>
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false)
                  onRemove()
                }}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-[13px] font-medium text-[var(--primary-deep)] transition-colors hover:bg-[var(--primary-soft)]"
              >
                <UserMinus className="h-3.5 w-3.5" strokeWidth={1.8} />
                {t("removeAction")}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function RoleBadge({ role }: { role: TeamMember["role"] }) {
  const t = useTranslations("team")
  const label = t(`roles.${role}`)
  if (role === "owner") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-[var(--amber-soft)] px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--warning)]">
        <Crown className="h-3 w-3" strokeWidth={2} />
        {label}
      </span>
    )
  }
  // Soleil simplifies role tiers into the same calm palette family.
  // Admin keeps a corail accent (matches the operator-CTA tone), Member
  // is the neutral tabac pill, Viewer is the subtler sable variant.
  if (role === "admin") {
    return (
      <span className="inline-flex items-center rounded-full bg-[var(--primary-soft)] px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--primary-deep)]">
        {label}
      </span>
    )
  }
  if (role === "member") {
    return (
      <span className="inline-flex items-center rounded-full bg-[var(--success-soft)] px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--success)]">
        {label}
      </span>
    )
  }
  return (
    <span className="inline-flex items-center rounded-full bg-[var(--background)] px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--muted-foreground)]">
      {label}
    </span>
  )
}
