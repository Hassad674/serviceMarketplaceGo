"use client"

import { useCallback, useMemo, useState } from "react"
import { AlertTriangle, ChevronDown, Lock, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"
import {
  useRolePermissionsMatrix,
  useUpdateRolePermissions,
} from "../hooks/use-team"
import type {
  OrgRole,
  RolePermissionCell,
  RolePermissionsRow,
  UpdateRolePermissionsPayload,
} from "../types"
import {
  ConfirmSaveModal,
  PermissionRow,
  ReadOnlyBanner,
  RolePermissionsEditorSkeleton,
  StickySaveBar,
  capitalize,
  extractErrorMessage,
  groupPermissionsByGroup,
} from "./role-permissions-editor-parts"

// RolePermissionsEditor is the unified "Roles and permissions" section
// rendered on /team. It serves TWO audiences through a single UI:
//
//  - Owners:
//      * Pick a role tab (Admin / Member / Viewer — never Owner).
//      * Toggle each permission on or off. The visual state reflects
//        the cell's origin: default (no badge), granted-override
//        (green pill), revoked-override (red pill).
//      * A sticky save bar appears at the bottom as soon as a pending
//        change is made. Discard reverts to the server state.
//      * Save opens a confirmation modal that spells out how many
//        members will be logged out.
//      * On success, a toast reports the granted/revoked counts.
//
//  - Non-Owners (Admin / Member / Viewer, and the Owner during a
//    pending ownership transfer): the exact same grid, but every
//    toggle is disabled and the save bar / reset button are hidden.
//    A read-only banner at the top explains why. This lets every
//    member see their org's actual customizations — which is
//    valuable context even without the ability to edit.
//
// Non-overridable permissions (org.delete, team.transfer_ownership,
// wallet.withdraw, kyc.manage, team.manage_role_permissions) are
// filtered out of the grid and shown in a compact "Owner-only"
// footer section for everyone. They are still enforced by the
// backend — this is purely a visual simplification.

type RolePermissionsEditorProps = {
  orgID: string
  readOnly?: boolean
}

// Canonical editable roles. Owner is excluded on purpose — the Owner
// row is always locked at the backend level and would only confuse
// users who cannot distinguish "Owner" from "the rest".
const EDITABLE_ROLES: Array<Exclude<OrgRole, "owner">> = ["admin", "member", "viewer"]

export function RolePermissionsEditor({
  orgID,
  readOnly = false,
}: RolePermissionsEditorProps) {
  const t = useTranslations("team")
  const { data, isLoading, error } = useRolePermissionsMatrix(orgID)
  const mutation = useUpdateRolePermissions(orgID)

  const [selectedRole, setSelectedRole] = useState<Exclude<OrgRole, "owner">>("admin")
  const [pendingChanges, setPendingChanges] = useState<Record<string, boolean>>({})
  const [showConfirm, setShowConfirm] = useState(false)
  const [expanded, setExpanded] = useState(true)

  // Whenever the Owner switches roles, drop any pending changes from
  // the previous role — the save is per-role, not a global staged
  // batch, and mixing two roles in one save bar would be confusing.
  // Render-time tracking avoids the setState-in-effect cascade.
  const [lastSelectedRole, setLastSelectedRole] = useState(selectedRole)
  if (lastSelectedRole !== selectedRole) {
    setLastSelectedRole(selectedRole)
    setPendingChanges({})
  }

  // Read-only mode must never leak pending state into the UI. If the
  // caller flips `readOnly` mid-session (e.g. ownership transfer
  // accepted while the Owner is staring at the editor), drop any
  // in-flight changes so the save bar and confirm modal can never
  // reappear.
  const [lastReadOnly, setLastReadOnly] = useState(readOnly)
  if (lastReadOnly !== readOnly) {
    setLastReadOnly(readOnly)
    if (readOnly) {
      setPendingChanges({})
      setShowConfirm(false)
    }
  }

  const currentRoleRow = useMemo<RolePermissionsRow | undefined>(
    () => data?.roles.find((r) => r.role === selectedRole),
    [data, selectedRole],
  )

  // Partition the current role's cells into the editable grid (what
  // appears in the main list) and the non-overridable footer list
  // (hard-coded on the backend, always shown in a dedicated section).
  // `cell.locked === true` is the backend's authoritative signal for
  // non-overridable permissions on non-Owner rows.
  // `lockedCells` is intentionally kept in the memo result so the
  // partition stays explicit even if a future render needs it; we
  // currently only render `editableCells`.
  const { editableCells, lockedCells: _lockedCells } = useMemo(() => {
    if (!currentRoleRow) {
      return { editableCells: [], lockedCells: [] as RolePermissionCell[] }
    }
    const editable: RolePermissionCell[] = []
    const locked: RolePermissionCell[] = []
    for (const cell of currentRoleRow.permissions) {
      if (cell.locked) {
        locked.push(cell)
      } else {
        editable.push(cell)
      }
    }
    return { editableCells: editable, lockedCells: locked }
  }, [currentRoleRow])

  const pendingCount = Object.keys(pendingChanges).length

  const togglePermission = useCallback(
    (perm: RolePermissionCell) => {
      if (readOnly || perm.locked) return
      setPendingChanges((prev) => {
        const next = { ...prev }
        const serverGranted = perm.granted
        // If the new value equals the server state, drop the entry —
        // no point in keeping a "change" that cancels itself.
        const currentlyPending = perm.key in next
        const newValue = currentlyPending ? !next[perm.key] : !serverGranted
        if (newValue === serverGranted) {
          delete next[perm.key]
        } else {
          next[perm.key] = newValue
        }
        return next
      })
    },
    [readOnly],
  )

  const resetPending = useCallback(() => setPendingChanges({}), [])

  // Build the full payload for the save endpoint: start from the
  // effective state of every non-locked cell (either pending or
  // server), and hand the result off. The backend will collapse
  // any cell that equals the default back to "no override".
  const buildOverridesPayload = useCallback((): Record<string, boolean> => {
    if (!currentRoleRow) return {}
    const out: Record<string, boolean> = {}
    for (const cell of currentRoleRow.permissions) {
      if (cell.locked) continue
      const override =
        cell.key in pendingChanges ? pendingChanges[cell.key] : cell.granted
      out[cell.key] = override
    }
    return out
  }, [currentRoleRow, pendingChanges])

  const handleSave = useCallback(() => {
    if (readOnly || !currentRoleRow || pendingCount === 0) return
    const payload: UpdateRolePermissionsPayload = {
      role: selectedRole,
      overrides: buildOverridesPayload(),
    }
    mutation.mutate(payload, {
      onSuccess: (result) => {
        const grantedCount = result.granted_keys.length
        const revokedCount = result.revoked_keys.length
        toast.success(t("rolePermissions.saveSuccessTitle"), {
          description: t("rolePermissions.saveSuccessDescription", {
            granted: grantedCount,
            revoked: revokedCount,
            affected: result.affected_members,
          }),
        })
        setPendingChanges({})
        setShowConfirm(false)
      },
      onError: (err: unknown) => {
        toast.error(t("rolePermissions.saveErrorTitle"), {
          description: extractErrorMessage(err, t),
        })
        setShowConfirm(false)
      },
    })
  }, [
    readOnly,
    currentRoleRow,
    pendingCount,
    selectedRole,
    buildOverridesPayload,
    mutation,
    t,
  ])

  // ---------- Rendering ----------

  if (isLoading) {
    return (
      <section className="rounded-xl border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-5">
        <RolePermissionsEditorSkeleton />
      </section>
    )
  }

  if (error || !data) {
    return (
      <section className="rounded-xl border border-red-200 dark:border-red-500/30 bg-red-50/50 dark:bg-red-500/5 p-5">
        <div className="flex items-center gap-2 text-red-700 dark:text-red-300">
          <AlertTriangle className="h-4 w-4" />
          <p className="text-sm font-medium">{t("rolePermissions.loadError")}</p>
        </div>
      </section>
    )
  }

  // Owner-only permissions are stable across all roles (they are the
  // same set of keys flagged non-overridable on the backend). Prefer
  // the Admin row as the source of truth so the list is never empty;
  // fall back to the currently-selected row if Admin is missing.
  const ownerExclusiveRow =
    data.roles.find((r) => r.role === "admin") ?? currentRoleRow
  const ownerExclusiveCells =
    ownerExclusiveRow?.permissions.filter((c) => c.locked) ?? []

  const showSaveBar = !readOnly && pendingCount > 0

  return (
    <section className="rounded-xl border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800">
      <EditorHeader
        expanded={expanded}
        onToggle={() => setExpanded((v) => !v)}
        pendingCount={pendingCount}
        readOnly={readOnly}
      />

      {expanded && (
        <div className="border-t border-gray-200 dark:border-slate-700">
          {readOnly && <ReadOnlyBanner />}

          <RoleTabs
            selectedRole={selectedRole}
            onSelect={(role) => setSelectedRole(role)}
          />

          <div className="p-5">
            {currentRoleRow ? (
              <PermissionsList
                description={currentRoleRow.description}
                cells={editableCells}
                pendingChanges={pendingChanges}
                readOnly={readOnly}
                onToggle={togglePermission}
              />
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {t("rolePermissions.noPermissions")}
              </p>
            )}

            {ownerExclusiveCells.length > 0 && (
              <OwnerExclusiveSection cells={ownerExclusiveCells} />
            )}
          </div>

          {showSaveBar && (
            <StickySaveBar
              pendingCount={pendingCount}
              saving={mutation.isPending}
              onDiscard={resetPending}
              onSave={() => setShowConfirm(true)}
            />
          )}
        </div>
      )}

      {showConfirm && currentRoleRow && !readOnly && (
        <ConfirmSaveModal
          role={selectedRole}
          pendingCount={pendingCount}
          affectedMembers={undefined}
          onConfirm={handleSave}
          onCancel={() => setShowConfirm(false)}
          saving={mutation.isPending}
        />
      )}
    </section>
  )
}

// ---------------------------------------------------------------------------
// Sub-components (orchestration-local)
// ---------------------------------------------------------------------------

function EditorHeader({
  expanded,
  onToggle,
  pendingCount,
  readOnly,
}: {
  expanded: boolean
  onToggle: () => void
  pendingCount: number
  readOnly: boolean
}) {
  const t = useTranslations("team")
  const subtitleKey = readOnly
    ? "rolePermissions.subtitleReadOnly"
    : "rolePermissions.subtitle"
  return (
    <button
      type="button"
      onClick={onToggle}
      className="flex w-full items-center justify-between p-5 text-left"
      aria-expanded={expanded}
    >
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 items-center justify-center rounded-full bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-300">
          <Sparkles className="h-5 w-5" />
        </span>
        <div>
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">
            {t("rolePermissions.title")}
          </h2>
          <p className="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
            {t(subtitleKey)}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-3">
        {!readOnly && pendingCount > 0 && (
          <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
            {t("rolePermissions.pendingBadge", { count: pendingCount })}
          </span>
        )}
        <ChevronDown
          className={`h-5 w-5 text-gray-400 transition-transform ${expanded ? "rotate-180" : ""}`}
          aria-hidden
        />
      </div>
    </button>
  )
}

function RoleTabs({
  selectedRole,
  onSelect,
}: {
  selectedRole: Exclude<OrgRole, "owner">
  onSelect: (role: Exclude<OrgRole, "owner">) => void
}) {
  const t = useTranslations("team")
  return (
    <div
      className="flex border-b border-gray-200 dark:border-slate-700"
      role="tablist"
      aria-label={t("rolePermissions.tabsAria")}
    >
      {EDITABLE_ROLES.map((role) => {
        const active = role === selectedRole
        return (
          <button
            key={role}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onSelect(role)}
            className={`flex-1 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
              active
                ? "border-rose-500 text-rose-600 dark:text-rose-300"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            }`}
          >
            {t(`roles.${role}.label`)}
          </button>
        )
      })}
    </div>
  )
}

function PermissionsList({
  description,
  cells,
  pendingChanges,
  readOnly,
  onToggle,
}: {
  description: string
  cells: RolePermissionCell[]
  pendingChanges: Record<string, boolean>
  readOnly: boolean
  onToggle: (perm: RolePermissionCell) => void
}) {
  const t = useTranslations("team")
  // Group cells by the permission "group" field so the UI renders
  // sections like Jobs / Proposals / Messaging. Preserve the backend's
  // order since the domain's allPermissionsOrdered is stable.
  const groups = useMemo(() => groupPermissionsByGroup(cells), [cells])

  if (cells.length === 0) {
    return (
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {t("rolePermissions.noPermissions")}
      </p>
    )
  }

  return (
    <div className="space-y-6">
      <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>
      {groups.map(({ group, items }) => (
        <div key={group}>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {t(`permissionGroups.${group}`, { fallback: capitalize(group) })}
          </h3>
          <ul className="space-y-1">
            {items.map((cell) => {
              const pendingValue =
                cell.key in pendingChanges ? pendingChanges[cell.key] : undefined
              const effectiveGranted =
                pendingValue !== undefined ? pendingValue : cell.granted
              const modified = pendingValue !== undefined
              return (
                <PermissionRow
                  key={cell.key}
                  cell={cell}
                  effectiveGranted={effectiveGranted}
                  modified={modified}
                  disabled={readOnly}
                  onToggle={() => onToggle(cell)}
                />
              )
            })}
          </ul>
        </div>
      ))}
    </div>
  )
}

// OwnerExclusiveSection lists the non-overridable permissions in a
// compact informational block at the bottom of the editor. These
// cells are NEVER editable and do not depend on the read-only flag —
// they are shown the same way to every audience, including the Owner.
function OwnerExclusiveSection({ cells }: { cells: RolePermissionCell[] }) {
  const t = useTranslations("team")
  return (
    <div className="mt-8 rounded-lg border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-700 dark:bg-slate-900/40">
      <div className="flex items-start gap-2.5">
        <Lock
          className="mt-0.5 h-4 w-4 shrink-0 text-slate-500 dark:text-slate-400"
          aria-hidden
        />
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            {t("rolePermissions.ownerExclusiveTitle")}
          </h3>
          <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
            {t("rolePermissions.ownerExclusiveDescription")}
          </p>
          <ul className="mt-3 space-y-2">
            {cells.map((cell) => (
              <li key={cell.key} className="text-sm">
                <p className="font-medium text-slate-800 dark:text-slate-200">
                  {cell.label || cell.key}
                </p>
                {cell.description && (
                  <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                    {cell.description}
                  </p>
                )}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
