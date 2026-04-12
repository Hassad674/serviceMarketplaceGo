"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import {
  AlertTriangle,
  Check,
  CheckCircle2,
  ChevronDown,
  Loader2,
  Lock,
  RotateCcw,
  Sparkles,
  X,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"
import {
  useRolePermissionsMatrix,
  useUpdateRolePermissions,
} from "../hooks/use-team"
import type {
  OrgRole,
  RolePermissionCell,
  RolePermissionCellState,
  RolePermissionsRow,
  UpdateRolePermissionsPayload,
} from "../types"

// RolePermissionsEditor is the Owner-only section rendered at the
// bottom of /team when the caller has PermTeamManageRolePermissions.
//
// UX flow:
//  - The Owner picks a role tab (Admin / Member / Viewer — never Owner).
//  - Each permission is a row with a toggle switch. The visual state
//    reflects the cell's origin: default (no badge), granted-override
//    (green pill), revoked-override (red pill), locked (lock icon,
//    disabled). Locked rows cannot be toggled regardless of click.
//  - A sticky save bar appears at the bottom as soon as the Owner
//    makes a pending change. Discard reverts to the server state.
//  - Save opens a confirmation modal that spells out how many members
//    will be logged out.
//  - On success, a toast reports the granted/revoked counts.

type RolePermissionsEditorProps = {
  orgID: string
}

// Canonical editable roles. Owner is excluded on purpose.
const EDITABLE_ROLES: Array<Exclude<OrgRole, "owner">> = ["admin", "member", "viewer"]

export function RolePermissionsEditor({ orgID }: RolePermissionsEditorProps) {
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
  useEffect(() => {
    setPendingChanges({})
  }, [selectedRole])

  const currentRoleRow = useMemo<RolePermissionsRow | undefined>(
    () => data?.roles.find((r) => r.role === selectedRole),
    [data, selectedRole],
  )

  const pendingCount = Object.keys(pendingChanges).length

  const togglePermission = useCallback(
    (perm: RolePermissionCell) => {
      if (perm.locked) return
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
    [],
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
    if (!currentRoleRow || pendingCount === 0) return
    const payload: UpdateRolePermissionsPayload = {
      role: selectedRole,
      overrides: buildOverridesPayload(),
    }
    mutation.mutate(payload, {
      onSuccess: (result) => {
        const grantedCount = result.granted_keys.length
        const revokedCount = result.revoked_keys.length
        toast.success(
          t("rolePermissions.saveSuccessTitle"),
          {
            description: t("rolePermissions.saveSuccessDescription", {
              granted: grantedCount,
              revoked: revokedCount,
              affected: result.affected_members,
            }),
          },
        )
        setPendingChanges({})
        setShowConfirm(false)
      },
      onError: (err: unknown) => {
        toast.error(
          t("rolePermissions.saveErrorTitle"),
          { description: extractErrorMessage(err, t) },
        )
        setShowConfirm(false)
      },
    })
  }, [currentRoleRow, pendingCount, selectedRole, buildOverridesPayload, mutation, t])

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

  const estimatedAffected = data.roles.find((r) => r.role === selectedRole)
    ? undefined
    : 0

  return (
    <section className="rounded-xl border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800">
      <EditorHeader
        expanded={expanded}
        onToggle={() => setExpanded((v) => !v)}
        pendingCount={pendingCount}
      />

      {expanded && (
        <div className="border-t border-gray-200 dark:border-slate-700">
          <RoleTabs
            selectedRole={selectedRole}
            onSelect={(role) => setSelectedRole(role)}
          />

          <div className="p-5">
            {currentRoleRow ? (
              <PermissionsList
                row={currentRoleRow}
                pendingChanges={pendingChanges}
                onToggle={togglePermission}
              />
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {t("rolePermissions.noPermissions")}
              </p>
            )}
          </div>

          {pendingCount > 0 && (
            <StickySaveBar
              pendingCount={pendingCount}
              saving={mutation.isPending}
              onDiscard={resetPending}
              onSave={() => setShowConfirm(true)}
            />
          )}
        </div>
      )}

      {showConfirm && currentRoleRow && (
        <ConfirmSaveModal
          role={selectedRole}
          pendingCount={pendingCount}
          affectedMembers={estimatedAffected}
          onConfirm={handleSave}
          onCancel={() => setShowConfirm(false)}
          saving={mutation.isPending}
        />
      )}
    </section>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function EditorHeader({
  expanded,
  onToggle,
  pendingCount,
}: {
  expanded: boolean
  onToggle: () => void
  pendingCount: number
}) {
  const t = useTranslations("team")
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
            {t("rolePermissions.subtitle")}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-3">
        {pendingCount > 0 && (
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
  row,
  pendingChanges,
  onToggle,
}: {
  row: RolePermissionsRow
  pendingChanges: Record<string, boolean>
  onToggle: (perm: RolePermissionCell) => void
}) {
  const t = useTranslations("team")
  // Group cells by the permission "group" field so the UI renders
  // sections like Jobs / Proposals / Messaging. Preserve the backend's
  // order since the domain's allPermissionsOrdered is stable.
  const groups = useMemo(() => groupPermissionsByGroup(row.permissions), [row.permissions])

  return (
    <div className="space-y-6">
      <p className="text-sm text-gray-500 dark:text-gray-400">{row.description}</p>
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

function PermissionRow({
  cell,
  effectiveGranted,
  modified,
  onToggle,
}: {
  cell: RolePermissionCell
  effectiveGranted: boolean
  modified: boolean
  onToggle: () => void
}) {
  const t = useTranslations("team")
  const state = resolveDisplayState(cell, effectiveGranted, modified)
  return (
    <li
      className={`flex items-start justify-between gap-3 rounded-lg border p-3 transition-colors ${
        cell.locked
          ? "border-gray-100 bg-gray-50/50 dark:border-slate-700 dark:bg-slate-900/40"
          : "border-gray-100 hover:border-gray-200 dark:border-slate-700 dark:hover:border-slate-600"
      }`}
    >
      <div className="flex min-w-0 flex-1 items-start gap-3">
        {cell.locked && (
          <Lock className="mt-1 h-4 w-4 shrink-0 text-gray-400" aria-hidden />
        )}
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <p className="truncate text-sm font-medium text-gray-900 dark:text-white">
              {cell.label || cell.key}
            </p>
            <StateBadge state={state} />
          </div>
          {cell.description && (
            <p className="mt-0.5 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
              {cell.description}
            </p>
          )}
          {cell.locked && (
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-500">
              {t("rolePermissions.lockedHint")}
            </p>
          )}
        </div>
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={effectiveGranted}
        aria-label={cell.label || cell.key}
        disabled={cell.locked}
        onClick={onToggle}
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${
          cell.locked
            ? "cursor-not-allowed bg-gray-200 dark:bg-slate-700"
            : effectiveGranted
              ? "bg-rose-500"
              : "bg-gray-300 dark:bg-slate-600"
        }`}
      >
        <span
          className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
            effectiveGranted ? "translate-x-5" : "translate-x-0.5"
          }`}
        />
      </button>
    </li>
  )
}

function StateBadge({ state }: { state: RolePermissionCellState }) {
  const t = useTranslations("team")
  if (state === "granted_override") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">
        <CheckCircle2 className="h-3 w-3" />
        {t("rolePermissions.states.grantedOverride")}
      </span>
    )
  }
  if (state === "revoked_override") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-[11px] font-medium text-red-700 dark:bg-red-500/15 dark:text-red-300">
        <X className="h-3 w-3" />
        {t("rolePermissions.states.revokedOverride")}
      </span>
    )
  }
  if (state === "locked") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-slate-700 dark:text-gray-300">
        <Lock className="h-3 w-3" />
        {t("rolePermissions.states.locked")}
      </span>
    )
  }
  return null
}

function StickySaveBar({
  pendingCount,
  saving,
  onDiscard,
  onSave,
}: {
  pendingCount: number
  saving: boolean
  onDiscard: () => void
  onSave: () => void
}) {
  const t = useTranslations("team")
  return (
    <div
      className="sticky bottom-0 flex items-center justify-between gap-3 border-t border-gray-200 bg-white px-5 py-3 dark:border-slate-700 dark:bg-slate-800"
      role="region"
      aria-label={t("rolePermissions.saveBarAria")}
    >
      <p className="text-sm text-gray-700 dark:text-gray-200">
        {t("rolePermissions.pendingBadge", { count: pendingCount })}
      </p>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onDiscard}
          disabled={saving}
          className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-slate-600 dark:text-gray-200 dark:hover:bg-slate-700"
        >
          <RotateCcw className="h-4 w-4" />
          {t("rolePermissions.discard")}
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={saving}
          className="inline-flex items-center gap-1.5 rounded-lg bg-rose-500 px-3.5 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Check className="h-4 w-4" />
          )}
          {t("rolePermissions.save")}
        </button>
      </div>
    </div>
  )
}

function ConfirmSaveModal({
  role,
  pendingCount,
  affectedMembers,
  onConfirm,
  onCancel,
  saving,
}: {
  role: Exclude<OrgRole, "owner">
  pendingCount: number
  affectedMembers: number | undefined
  onConfirm: () => void
  onCancel: () => void
  saving: boolean
}) {
  const t = useTranslations("team")
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="role-perms-confirm-title"
    >
      <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl dark:bg-slate-800">
        <div className="flex items-start gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300">
            <AlertTriangle className="h-5 w-5" />
          </span>
          <div>
            <h3
              id="role-perms-confirm-title"
              className="text-base font-semibold text-gray-900 dark:text-white"
            >
              {t("rolePermissions.confirmTitle", { role: t(`roles.${role}.label`) })}
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t("rolePermissions.confirmDescription", {
                count: pendingCount,
                affected: affectedMembers ?? "?",
              })}
            </p>
          </div>
        </div>
        <div className="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={saving}
            className="inline-flex items-center rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-slate-600 dark:text-gray-200 dark:hover:bg-slate-700"
          >
            {t("rolePermissions.cancel")}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={saving}
            className="inline-flex items-center gap-1.5 rounded-lg bg-rose-500 px-3.5 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Check className="h-4 w-4" />
            )}
            {t("rolePermissions.confirmButton")}
          </button>
        </div>
      </div>
    </div>
  )
}

function RolePermissionsEditorSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <div className="h-10 w-10 animate-pulse rounded-full bg-gray-200 dark:bg-slate-700" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-32 animate-pulse rounded bg-gray-200 dark:bg-slate-700" />
          <div className="h-3 w-48 animate-pulse rounded bg-gray-100 dark:bg-slate-700/60" />
        </div>
      </div>
      <div className="h-10 animate-pulse rounded-lg bg-gray-100 dark:bg-slate-700/60" />
      <div className="space-y-2">
        {[0, 1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="h-12 animate-pulse rounded-lg bg-gray-100 dark:bg-slate-700/60"
          />
        ))}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// groupPermissionsByGroup reorganizes a flat cell list into an ordered
// sequence of groups without disturbing the relative order inside
// each group. The domain returns allPermissionsOrdered in a stable
// sequence, so this preserves it.
function groupPermissionsByGroup(
  cells: RolePermissionCell[],
): Array<{ group: string; items: RolePermissionCell[] }> {
  const order: string[] = []
  const buckets: Record<string, RolePermissionCell[]> = {}
  for (const cell of cells) {
    if (!(cell.group in buckets)) {
      buckets[cell.group] = []
      order.push(cell.group)
    }
    buckets[cell.group].push(cell)
  }
  return order.map((group) => ({ group, items: buckets[group] }))
}

// resolveDisplayState adjusts the server-reported state to match the
// UI's pending-change-aware view. A pending change on a default cell
// becomes granted_override / revoked_override even before the Owner
// saves, so the badge appears immediately.
function resolveDisplayState(
  cell: RolePermissionCell,
  effectiveGranted: boolean,
  modified: boolean,
): RolePermissionCellState {
  if (cell.locked) return "locked"
  if (!modified) return cell.state
  // Modified: the display state reflects the NEW value vs the default.
  // We derive default from the server cell: default_granted / default_revoked
  // map directly; granted_override means default was revoked; revoked_override
  // means default was granted.
  const defaultGranted =
    cell.state === "default_granted" ||
    cell.state === "revoked_override"
  if (effectiveGranted === defaultGranted) {
    return defaultGranted ? "default_granted" : "default_revoked"
  }
  return effectiveGranted ? "granted_override" : "revoked_override"
}

function capitalize(s: string): string {
  if (!s) return s
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function extractErrorMessage(err: unknown, t: (key: string) => string): string {
  if (err && typeof err === "object" && "message" in err) {
    const msg = (err as { message: unknown }).message
    if (typeof msg === "string" && msg.length > 0) return msg
  }
  return t("rolePermissions.saveErrorGeneric")
}
