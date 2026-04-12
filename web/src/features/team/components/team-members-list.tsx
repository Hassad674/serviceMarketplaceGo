"use client"

import { useEffect, useRef, useState } from "react"
import { Crown, MoreVertical, Pencil, UserMinus } from "lucide-react"
import { useTranslations } from "next-intl"
import { EditMemberModal } from "./edit-member-modal"
import { RemoveMemberDialog } from "./remove-member-dialog"
import type { TeamMember } from "../types"

// Members table with permission-gated row actions. The "manage"
// permission is resolved upstream by the parent (team-page) so this
// component stays dumb — it just renders rows + opens dialogs.
//
// Owners never expose a dropdown: there is no "change role" for an
// Owner (use Transfer Ownership) and no "remove" for an Owner (same
// reason). Solo Owners watching an otherwise-empty list see their
// own row without any menu.

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
      <div className="rounded-xl border border-dashed border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {t("emptyMembers")}
      </div>
    )
  }

  return (
    <>
      <div className="overflow-hidden rounded-xl border border-gray-100 dark:border-slate-700 bg-white dark:bg-slate-800">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-100 dark:border-slate-700 bg-gray-50 dark:bg-slate-900/50 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
              <th className="px-6 py-3">{t("columns.member")}</th>
              <th className="px-6 py-3">{t("columns.role")}</th>
              <th className="px-6 py-3">{t("columns.title")}</th>
              {canManage && <th className="px-6 py-3 text-right">{t("columns.actions")}</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 dark:divide-slate-700">
            {members.map((m) => (
              <MemberRow
                key={m.id}
                member={m}
                canManage={canManage}
                onEdit={() => setEditTarget(m)}
                onRemove={() => setRemoveTarget(m)}
              />
            ))}
          </tbody>
        </table>
      </div>

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
  canManage: boolean
  onEdit: () => void
  onRemove: () => void
}

function MemberRow({ member, canManage, onEdit, onRemove }: MemberRowProps) {
  const t = useTranslations("team")
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const isOwner = member.role === "owner"
  const name =
    member.user?.display_name ||
    `${member.user?.first_name ?? ""} ${member.user?.last_name ?? ""}`.trim() ||
    t("memberFallbackName")
  const initials = (
    (member.user?.first_name?.charAt(0) ?? "") +
    (member.user?.last_name?.charAt(0) ?? "")
  ).toUpperCase() || "?"

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
    <tr className="text-sm">
      <td className="px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-rose-50 dark:bg-rose-500/10 text-sm font-semibold text-rose-600 dark:text-rose-400">
            {initials}
          </div>
          <div>
            <p className="font-medium text-gray-900 dark:text-white">{name}</p>
            {member.user?.email && (
              <p className="text-xs text-gray-500 dark:text-gray-400">{member.user.email}</p>
            )}
          </div>
        </div>
      </td>
      <td className="px-6 py-4">
        <RoleBadge role={member.role} />
      </td>
      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
        {member.title || <span className="text-gray-400 dark:text-gray-500">—</span>}
      </td>
      {canManage && (
        <td className="px-6 py-4 text-right">
          {!isOwner && (
            <div ref={menuRef} className="relative inline-block">
              <button
                type="button"
                onClick={() => setMenuOpen((v) => !v)}
                className="rounded-lg p-1.5 hover:bg-gray-100 dark:hover:bg-slate-700"
                aria-label={t("columns.actions")}
              >
                <MoreVertical className="h-4 w-4 text-gray-500" />
              </button>
              {menuOpen && (
                <div className="absolute right-0 z-10 mt-1 min-w-[160px] rounded-lg border border-gray-100 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-lg">
                  <button
                    type="button"
                    onClick={() => {
                      setMenuOpen(false)
                      onEdit()
                    }}
                    className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-slate-700"
                  >
                    <Pencil className="h-4 w-4" />
                    {t("editAction")}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setMenuOpen(false)
                      onRemove()
                    }}
                    className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10"
                  >
                    <UserMinus className="h-4 w-4" />
                    {t("removeAction")}
                  </button>
                </div>
              )}
            </div>
          )}
        </td>
      )}
    </tr>
  )
}

function RoleBadge({ role }: { role: TeamMember["role"] }) {
  const t = useTranslations("team")
  const label = t(`roles.${role}`)
  if (role === "owner") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 dark:bg-amber-500/10 px-2.5 py-0.5 text-xs font-semibold text-amber-700 dark:text-amber-300">
        <Crown className="h-3 w-3" />
        {label}
      </span>
    )
  }
  const classes: Record<Exclude<TeamMember["role"], "owner">, string> = {
    admin: "bg-violet-50 dark:bg-violet-500/10 text-violet-700 dark:text-violet-300",
    member: "bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-300",
    viewer: "bg-gray-100 dark:bg-slate-700 text-gray-700 dark:text-gray-300",
  }
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${classes[role]}`}
    >
      {label}
    </span>
  )
}
