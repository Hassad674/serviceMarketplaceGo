"use client"

import {
  AlertTriangle,
  Check,
  CheckCircle2,
  Eye,
  Loader2,
  RotateCcw,
  X,
} from "lucide-react"
import { useTranslations } from "next-intl"
import type {
  OrgRole,
  RolePermissionCell,
  RolePermissionCellState,
} from "../types"

// Extracted presentational sub-components for role-permissions-editor.
//
// Kept in a separate file to honour the 600-lines-per-file rule while
// preserving the editor's orchestration logic in a single main file.
// None of these components hold business logic — they are pure props-in,
// JSX-out helpers with their own i18n lookups.

/* -------------------------------------------------------------------------- */
/* Read-only banner                                                            */
/* -------------------------------------------------------------------------- */

// ReadOnlyBanner is rendered at the top of the editor for members who
// do not hold PermTeamManageRolePermissions. It makes it obvious that
// the toggles are disabled on purpose and points to the Owner as the
// only role that can change the matrix.
export function ReadOnlyBanner() {
  const t = useTranslations("team")
  return (
    <div
      className="flex items-start gap-2.5 border-b border-gray-200 bg-slate-50 px-5 py-3 dark:border-slate-700 dark:bg-slate-900/40"
      role="note"
    >
      <Eye
        className="mt-0.5 h-4 w-4 shrink-0 text-slate-500 dark:text-slate-400"
        aria-hidden
      />
      <p className="text-sm text-slate-600 dark:text-slate-300">
        {t("rolePermissions.readOnlyBanner")}
      </p>
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* State badge                                                                 */
/* -------------------------------------------------------------------------- */

// StateBadge renders the per-cell origin pill (default/override). The
// "locked" case never reaches this component in the new editor flow
// because non-overridable permissions are filtered out of the grid,
// but we keep the branch for type-safety and future-proofing.
export function StateBadge({ state }: { state: RolePermissionCellState }) {
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
  return null
}

/* -------------------------------------------------------------------------- */
/* Permission row                                                              */
/* -------------------------------------------------------------------------- */

type PermissionRowProps = {
  cell: RolePermissionCell
  effectiveGranted: boolean
  modified: boolean
  disabled: boolean
  onToggle: () => void
}

// PermissionRow is the single-row unit of the matrix grid. The toggle
// is fully disabled when `disabled` is true — this covers both the
// read-only mode (non-Owner) and any future "cannot interact" states.
export function PermissionRow({
  cell,
  effectiveGranted,
  modified,
  disabled,
  onToggle,
}: PermissionRowProps) {
  const state = resolveDisplayState(cell, effectiveGranted, modified)
  return (
    <li className="flex items-start justify-between gap-3 rounded-lg border border-gray-100 p-3 transition-colors hover:border-gray-200 dark:border-slate-700 dark:hover:border-slate-600">
      <div className="flex min-w-0 flex-1 items-start gap-3">
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
        </div>
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={effectiveGranted}
        aria-label={cell.label || cell.key}
        disabled={disabled}
        onClick={onToggle}
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${
          disabled
            ? `cursor-not-allowed ${
                effectiveGranted
                  ? "bg-rose-300 dark:bg-rose-500/40"
                  : "bg-gray-200 dark:bg-slate-700"
              }`
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

/* -------------------------------------------------------------------------- */
/* Sticky save bar                                                             */
/* -------------------------------------------------------------------------- */

type StickySaveBarProps = {
  pendingCount: number
  saving: boolean
  onDiscard: () => void
  onSave: () => void
}

// StickySaveBar appears once the Owner has staged at least one
// pending change. Hidden entirely in read-only mode.
export function StickySaveBar({
  pendingCount,
  saving,
  onDiscard,
  onSave,
}: StickySaveBarProps) {
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

/* -------------------------------------------------------------------------- */
/* Confirm save modal                                                          */
/* -------------------------------------------------------------------------- */

type ConfirmSaveModalProps = {
  role: Exclude<OrgRole, "owner">
  pendingCount: number
  affectedMembers: number | undefined
  onConfirm: () => void
  onCancel: () => void
  saving: boolean
}

// ConfirmSaveModal is the final guard before a destructive save: it
// spells out how many members will be logged out when the Owner
// confirms. Only reachable from the save bar — which itself is only
// visible in edit mode.
export function ConfirmSaveModal({
  role,
  pendingCount,
  affectedMembers,
  onConfirm,
  onCancel,
  saving,
}: ConfirmSaveModalProps) {
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

/* -------------------------------------------------------------------------- */
/* Skeleton                                                                    */
/* -------------------------------------------------------------------------- */

// RolePermissionsEditorSkeleton mirrors the overall layout so the
// editor panel does not shift when data loads in.
export function RolePermissionsEditorSkeleton() {
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

/* -------------------------------------------------------------------------- */
/* Helpers                                                                     */
/* -------------------------------------------------------------------------- */

// groupPermissionsByGroup reorganizes a flat cell list into an ordered
// sequence of groups without disturbing the relative order inside
// each group. The domain returns allPermissionsOrdered in a stable
// sequence, so this preserves it.
export function groupPermissionsByGroup(
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
export function resolveDisplayState(
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
    cell.state === "default_granted" || cell.state === "revoked_override"
  if (effectiveGranted === defaultGranted) {
    return defaultGranted ? "default_granted" : "default_revoked"
  }
  return effectiveGranted ? "granted_override" : "revoked_override"
}

export function extractErrorMessage(
  err: unknown,
  t: (key: string) => string,
): string {
  if (err && typeof err === "object" && "message" in err) {
    const msg = (err as { message: unknown }).message
    if (typeof msg === "string" && msg.length > 0) return msg
  }
  return t("rolePermissions.saveErrorGeneric")
}

export function capitalize(s: string): string {
  if (!s) return s
  return s.charAt(0).toUpperCase() + s.slice(1)
}
