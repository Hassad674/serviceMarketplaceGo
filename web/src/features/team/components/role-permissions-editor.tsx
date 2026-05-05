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

// Soleil v2 — Roles & permissions editor. The unified section serves
// two audiences via one UI: Owners can toggle, non-Owners (and the
// Owner during a pending ownership transfer) see the same grid in
// read-only mode. Non-overridable permissions are filtered out of the
// editable grid and shown in a compact "Owner-only" footer for
// everyone. Backend enforces Owner-only writes (R17).

type RolePermissionsEditorProps = {
  orgID: string
  readOnly?: boolean
}

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

  const [lastSelectedRole, setLastSelectedRole] = useState(selectedRole)
  if (lastSelectedRole !== selectedRole) {
    setLastSelectedRole(selectedRole)
    setPendingChanges({})
  }

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
      <section className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-5 shadow-[var(--shadow-card)]">
        <RolePermissionsEditorSkeleton />
      </section>
    )
  }

  if (error || !data) {
    return (
      <section className="rounded-2xl border border-[var(--primary-soft)] bg-[var(--primary-soft)]/40 p-5">
        <div className="flex items-center gap-2 text-[var(--primary-deep)]">
          <AlertTriangle className="h-4 w-4" strokeWidth={1.8} />
          <p className="text-[13px] font-semibold">{t("rolePermissions.loadError")}</p>
        </div>
      </section>
    )
  }

  const ownerExclusiveRow =
    data.roles.find((r) => r.role === "admin") ?? currentRoleRow
  const ownerExclusiveCells =
    ownerExclusiveRow?.permissions.filter((c) => c.locked) ?? []

  const showSaveBar = !readOnly && pendingCount > 0

  return (
    <section className="overflow-hidden rounded-2xl border border-[var(--border)] bg-[var(--surface)] shadow-[var(--shadow-card)]">
      <EditorHeader
        expanded={expanded}
        onToggle={() => setExpanded((v) => !v)}
        pendingCount={pendingCount}
        readOnly={readOnly}
      />

      {expanded && (
        <div className="border-t border-[var(--border)]">
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
              <p className="font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
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
      className="flex w-full items-center justify-between gap-3 p-5 text-left transition-colors hover:bg-[var(--background)]"
      aria-expanded={expanded}
    >
      <div className="flex items-center gap-3">
        <span
          aria-hidden="true"
          className="flex h-11 w-11 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
        >
          <Sparkles className="h-5 w-5" strokeWidth={1.8} />
        </span>
        <div>
          <h2 className="font-serif text-[18px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
            {t("rolePermissions.title")}
          </h2>
          <p className="mt-0.5 font-serif text-[13px] italic text-[var(--muted-foreground)]">
            {t(subtitleKey)}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-3">
        {!readOnly && pendingCount > 0 && (
          <span className="inline-flex items-center gap-1 rounded-full bg-[var(--amber-soft)] px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--warning)]">
            {t("rolePermissions.pendingBadge", { count: pendingCount })}
          </span>
        )}
        <ChevronDown
          className={`h-5 w-5 text-[var(--muted-foreground)] transition-transform ${
            expanded ? "rotate-180" : ""
          }`}
          aria-hidden="true"
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
      className="flex border-b border-[var(--border)]"
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
            className={`flex-1 border-b-2 px-4 py-3 text-[13px] font-semibold transition-colors ${
              active
                ? "border-[var(--primary)] text-[var(--primary)]"
                : "border-transparent text-[var(--muted-foreground)] hover:bg-[var(--background)] hover:text-[var(--foreground)]"
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
  const groups = useMemo(() => groupPermissionsByGroup(cells), [cells])

  if (cells.length === 0) {
    return (
      <p className="font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
        {t("rolePermissions.noPermissions")}
      </p>
    )
  }

  return (
    <div className="space-y-6">
      <p className="font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
        {description}
      </p>
      {groups.map(({ group, items }) => (
        <div key={group}>
          <h3 className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.06em] text-[var(--subtle-foreground)]">
            {t(`permissionGroups.${group}`, { fallback: capitalize(group) })}
          </h3>
          <ul className="space-y-1.5">
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

// OwnerExclusiveSection — non-overridable permissions in a compact info
// block at the bottom of the editor. Always shown the same way to
// every audience, including the Owner.
function OwnerExclusiveSection({ cells }: { cells: RolePermissionCell[] }) {
  const t = useTranslations("team")
  return (
    <div className="mt-8 rounded-xl border border-[var(--border)] bg-[var(--background)] p-4">
      <div className="flex items-start gap-2.5">
        <Lock
          className="mt-0.5 h-4 w-4 shrink-0 text-[var(--muted-foreground)]"
          strokeWidth={1.8}
          aria-hidden="true"
        />
        <div className="min-w-0 flex-1">
          <h3 className="font-serif text-[14.5px] font-medium text-[var(--foreground)]">
            {t("rolePermissions.ownerExclusiveTitle")}
          </h3>
          <p className="mt-0.5 font-serif text-[12.5px] italic text-[var(--muted-foreground)]">
            {t("rolePermissions.ownerExclusiveDescription")}
          </p>
          <ul className="mt-3 space-y-2">
            {cells.map((cell) => (
              <li key={cell.key} className="text-[13px]">
                <p className="font-medium text-[var(--foreground)]">
                  {cell.label || cell.key}
                </p>
                {cell.description && (
                  <p className="mt-0.5 text-[12px] text-[var(--muted-foreground)]">
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
