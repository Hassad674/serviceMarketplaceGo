"use client"

import { useState } from "react"
import { Loader2, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useUpdateMember } from "../hooks/use-team"
import type { TeamMember, OrgRole } from "../types"

// Dialog for changing a member's role and/or title. Owner is never
// the target here (the list hides the dropdown for Owner). The role
// selector never includes "owner" — promotions to Owner go through
// the Transfer Ownership flow.

type EditMemberModalProps = {
  open: boolean
  onClose: () => void
  orgID: string
  member: TeamMember
}

const EDITABLE_ROLES: Array<Exclude<OrgRole, "owner">> = ["admin", "member", "viewer"]

export function EditMemberModal({ open, onClose, orgID, member }: EditMemberModalProps) {
  const t = useTranslations("team")
  const mutation = useUpdateMember(orgID, member.user_id)

  const initialRole: Exclude<OrgRole, "owner"> =
    member.role === "owner" ? "admin" : member.role
  const [role, setRole] = useState<Exclude<OrgRole, "owner">>(initialRole)
  const [title, setTitle] = useState(member.title)

  if (!open) return null

  const displayName =
    member.user?.display_name ||
    `${member.user?.first_name ?? ""} ${member.user?.last_name ?? ""}`.trim() ||
    t("memberFallbackName")

  const hasChanges = role !== member.role || title !== member.title

  function handleSubmit() {
    if (!hasChanges) return
    mutation.mutate(
      {
        ...(role !== member.role ? { role } : {}),
        ...(title !== member.title ? { title } : {}),
      },
      { onSuccess: () => onClose() },
    )
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-6 w-full max-w-md mx-4 animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
            {t("editMemberTitle", { name: displayName })}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700"
          >
            <X className="h-5 w-5 text-slate-400" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("roleLabel")}
            </label>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as Exclude<OrgRole, "owner">)}
              className="w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
            >
              {EDITABLE_ROLES.map((r) => (
                <option key={r} value={r}>
                  {t(`roles.${r}`)}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("titleLabel")}
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={100}
              placeholder={t("titlePlaceholder")}
              className="w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
            />
          </div>

          {mutation.isError && (
            <p className="text-sm text-rose-600 dark:text-rose-400">
              {t("errors.generic")}
            </p>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={mutation.isPending}
            className="rounded-lg border border-slate-200 dark:border-slate-600 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={mutation.isPending || !hasChanges}
            className="inline-flex items-center gap-2 rounded-lg bg-rose-500 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("save")}
          </button>
        </div>
      </div>
    </div>
  )
}
