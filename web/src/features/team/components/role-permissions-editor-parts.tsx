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

// Soleil v2 — extracted presentational sub-components for the role
// permissions editor. Pure props-in/JSX-out helpers with their own
// i18n lookups. No business logic.

/* -------------------------------------------------------------------------- */
/* Read-only banner                                                            */
/* -------------------------------------------------------------------------- */

export function ReadOnlyBanner() {
  const t = useTranslations("team")
  return (
    <div
      className="flex items-start gap-2.5 border-b border-[var(--border)] bg-[var(--background)] px-5 py-3"
      role="note"
    >
      <Eye
        className="mt-0.5 h-4 w-4 shrink-0 text-[var(--muted-foreground)]"
        strokeWidth={1.8}
        aria-hidden="true"
      />
      <p className="font-serif text-[13px] italic text-[var(--muted-foreground)]">
        {t("rolePermissions.readOnlyBanner")}
      </p>
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* State badge                                                                 */
/* -------------------------------------------------------------------------- */

export function StateBadge({ state }: { state: RolePermissionCellState }) {
  const t = useTranslations("team")
  if (state === "granted_override") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-[var(--success-soft)] px-2 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--success)]">
        <CheckCircle2 className="h-3 w-3" strokeWidth={2} />
        {t("rolePermissions.states.grantedOverride")}
      </span>
    )
  }
  if (state === "revoked_override") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-[var(--primary-soft)] px-2 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--primary-deep)]">
        <X className="h-3 w-3" strokeWidth={2} />
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

export function PermissionRow({
  cell,
  effectiveGranted,
  modified,
  disabled,
  onToggle,
}: PermissionRowProps) {
  const state = resolveDisplayState(cell, effectiveGranted, modified)
  return (
    <li className="flex items-start justify-between gap-3 rounded-xl border border-[var(--border)] bg-[var(--surface)] p-3 transition-colors hover:border-[var(--border-strong)]">
      <div className="flex min-w-0 flex-1 items-start gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="truncate text-[13.5px] font-medium text-[var(--foreground)]">
              {cell.label || cell.key}
            </p>
            <StateBadge state={state} />
          </div>
          {cell.description && (
            <p className="mt-0.5 line-clamp-2 text-[12px] text-[var(--muted-foreground)]">
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
                  ? "bg-[var(--primary-soft)]"
                  : "bg-[var(--border)]"
              }`
            : effectiveGranted
              ? "bg-[var(--primary)]"
              : "bg-[var(--border-strong)]"
        }`}
      >
        <span
          className={`inline-block h-5 w-5 transform rounded-full bg-[var(--surface)] shadow transition-transform ${
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

export function StickySaveBar({
  pendingCount,
  saving,
  onDiscard,
  onSave,
}: StickySaveBarProps) {
  const t = useTranslations("team")
  return (
    <div
      className="sticky bottom-0 flex flex-wrap items-center justify-between gap-3 border-t border-[var(--border)] bg-[var(--surface)] px-5 py-3"
      role="region"
      aria-label={t("rolePermissions.saveBarAria")}
    >
      <p className="text-[13px] font-medium text-[var(--foreground)]">
        {t("rolePermissions.pendingBadge", { count: pendingCount })}
      </p>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onDiscard}
          disabled={saving}
          className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border)] bg-[var(--surface)] px-3 py-1.5 text-[12.5px] font-semibold text-[var(--foreground)] transition-colors hover:border-[var(--border-strong)] hover:bg-[var(--background)] disabled:opacity-50"
        >
          <RotateCcw className="h-3.5 w-3.5" strokeWidth={1.8} />
          {t("rolePermissions.discard")}
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={saving}
          className="inline-flex items-center gap-1.5 rounded-full bg-[var(--primary)] px-3.5 py-1.5 text-[12.5px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
        >
          {saving ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Check className="h-3.5 w-3.5" strokeWidth={2} />
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
      className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(42,31,21,0.45)] p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-labelledby="role-perms-confirm-title"
    >
      <div className="animate-scale-in w-full max-w-md rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-6 shadow-[var(--shadow-card-strong)]">
        <div className="flex items-start gap-3">
          <span
            aria-hidden="true"
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--amber-soft)] text-[var(--warning)]"
          >
            <AlertTriangle className="h-5 w-5" strokeWidth={1.8} />
          </span>
          <div className="min-w-0">
            <h3
              id="role-perms-confirm-title"
              className="font-serif text-[18px] font-medium tracking-[-0.01em] text-[var(--foreground)]"
            >
              {t("rolePermissions.confirmTitle", { role: t(`roles.${role}.label`) })}
            </h3>
            <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
              {t("rolePermissions.confirmDescription", {
                count: pendingCount,
                affected: affectedMembers ?? "?",
              })}
            </p>
          </div>
        </div>
        <div className="mt-6 flex flex-wrap items-center justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={saving}
            className="inline-flex items-center rounded-full border border-[var(--border)] bg-[var(--surface)] px-3.5 py-2 text-[13px] font-semibold text-[var(--foreground)] transition-colors hover:border-[var(--border-strong)] hover:bg-[var(--background)] disabled:opacity-50"
          >
            {t("rolePermissions.cancel")}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={saving}
            className="inline-flex items-center gap-1.5 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Check className="h-4 w-4" strokeWidth={2} />
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

export function RolePermissionsEditorSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <div className="h-11 w-11 animate-pulse rounded-full bg-[var(--border)]" />
        <div className="flex-1 space-y-2">
          <div className="h-3 w-32 animate-pulse rounded-full bg-[var(--border)]" />
          <div className="h-3 w-48 animate-pulse rounded-full bg-[var(--border)]" />
        </div>
      </div>
      <div className="h-10 animate-pulse rounded-xl bg-[var(--border)]" />
      <div className="space-y-2">
        {[0, 1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="h-12 animate-pulse rounded-xl bg-[var(--border)]"
          />
        ))}
      </div>
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* Helpers                                                                     */
/* -------------------------------------------------------------------------- */

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

export function resolveDisplayState(
  cell: RolePermissionCell,
  effectiveGranted: boolean,
  modified: boolean,
): RolePermissionCellState {
  if (cell.locked) return "locked"
  if (!modified) return cell.state
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
