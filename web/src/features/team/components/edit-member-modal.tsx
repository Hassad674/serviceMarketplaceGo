"use client"

import { useMemo, useState } from "react"
import { Loader2, ShieldCheck, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"
import { useRoleDefinitions, useUpdateMember } from "../hooks/use-team"
import type { TeamMember, OrgRole, RoleDefinition } from "../types"

// Soleil v2 — Edit member modal. Ivoire surface, Fraunces title, corail
// permission preview chips. Owner is never the target (the list hides
// the dropdown for Owner). The role selector never includes "owner" —
// promotions to Owner go through the Transfer Ownership flow.

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
  const { data: roleDefinitions } = useRoleDefinitions()

  const initialRole: Exclude<OrgRole, "owner"> =
    member.role === "owner" ? "admin" : member.role
  const [role, setRole] = useState<Exclude<OrgRole, "owner">>(initialRole)
  const [title, setTitle] = useState(member.title)

  const selectedRoleDef = useMemo<RoleDefinition | undefined>(
    () => roleDefinitions?.roles.find((r) => r.key === role),
    [roleDefinitions, role],
  )

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
      {
        onSuccess: () => {
          toast.success(t("toasts.memberUpdated"))
          onClose()
        },
      },
    )
  }

  function permissionLabel(key: string): string {
    const found = roleDefinitions?.permissions.find((p) => p.key === key)
    return found?.label || key
  }

  const inputBase =
    "w-full rounded-xl border border-[var(--border)] bg-[var(--surface)] px-3 py-2.5 text-[14px] text-[var(--foreground)] focus:border-[var(--primary)] focus:outline-none focus:ring-2 focus:ring-[var(--primary-soft)]"

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(42,31,21,0.45)] p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-scale-in max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-6 shadow-[var(--shadow-card-strong)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between gap-3">
          <h3 className="font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
            {t("editMemberTitle", { name: displayName })}
          </h3>
          <button
            type="button"
            onClick={onClose}
            aria-label={t("cancel")}
            className="rounded-full p-1 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--background)] hover:text-[var(--foreground)]"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label
              htmlFor="edit-member-role"
              className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
            >
              {t("roleLabel")}
            </label>
            <select
              id="edit-member-role"
              value={role}
              onChange={(e) => setRole(e.target.value as Exclude<OrgRole, "owner">)}
              className={`${inputBase} cursor-pointer`}
            >
              {EDITABLE_ROLES.map((r) => (
                <option key={r} value={r}>
                  {t(`roles.${r}`)}
                </option>
              ))}
            </select>
          </div>

          {/* Inline permission preview — refreshes whenever the user
              picks a different role from the dropdown. */}
          <div className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-3.5">
            <div className="mb-2 flex items-center gap-2">
              <ShieldCheck
                className="h-4 w-4 text-[var(--primary)]"
                strokeWidth={1.8}
                aria-hidden="true"
              />
              <p className="font-mono text-[11px] font-bold uppercase tracking-[0.05em] text-[var(--muted-foreground)]">
                {t("editMember.rolePreviewLabel", { role: t(`roles.${role}`) })}
              </p>
            </div>
            {selectedRoleDef && selectedRoleDef.permissions.length > 0 ? (
              <ul className="grid grid-cols-1 gap-1.5">
                {selectedRoleDef.permissions.map((permKey) => (
                  <li
                    key={permKey}
                    className="flex items-center gap-2 text-[13px] text-[var(--foreground)]"
                  >
                    <span
                      aria-hidden="true"
                      className="inline-block h-1.5 w-1.5 rounded-full bg-[var(--primary)]"
                    />
                    {permissionLabel(permKey)}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="font-serif text-[12.5px] italic text-[var(--muted-foreground)]">
                {t("editMember.rolePreviewEmpty")}
              </p>
            )}
          </div>

          <div>
            <label
              htmlFor="edit-member-title"
              className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
            >
              {t("titleLabel")}
            </label>
            <input
              id="edit-member-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={100}
              placeholder={t("titlePlaceholder")}
              className={inputBase}
            />
          </div>

          {mutation.isError && (
            <p
              role="alert"
              className="rounded-xl border border-[var(--primary-soft)] bg-[var(--primary-soft)] px-3 py-2 text-[13px] text-[var(--primary-deep)]"
            >
              {t("errors.generic")}
            </p>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={mutation.isPending}
            className="rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 py-2 text-[13px] font-semibold text-[var(--foreground)] transition-colors hover:border-[var(--border-strong)] hover:bg-[var(--background)] disabled:opacity-50"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={mutation.isPending || !hasChanges}
            className="inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("save")}
          </button>
        </div>
      </div>
    </div>
  )
}
