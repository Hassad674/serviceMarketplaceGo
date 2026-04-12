"use client"

import { useCallback, useMemo, useState } from "react"
import {
  Check,
  ChevronDown,
  Crown,
  Eye,
  Info,
  Shield,
  ShieldCheck,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { useRoleDefinitions } from "../hooks/use-team"
import type { OrgRole, RoleDefinition, RoleDefinitionPermission } from "../types"

// Collapsible "About roles" panel rendered on the team page. Lists
// every role as an individually expandable accordion card, each
// showing its description and grouped permissions when opened.
//
// Pattern: outer section toggle reveals the role catalogue, then
// each role card has its own expand/collapse. Multiple cards can be
// open simultaneously so users can compare roles side-by-side.
//
// Smooth height animation uses the CSS grid-rows trick
// (grid-template-rows: 0fr -> 1fr) with overflow: hidden on the
// inner wrapper, combined with a transition on grid-template-rows.

const ROLE_ICONS: Record<OrgRole, typeof Shield> = {
  owner: Crown,
  admin: ShieldCheck,
  member: Shield,
  viewer: Eye,
}

// Color tokens applied to the role icon circle and accent border
// on hover/expanded states. Owner = amber, Admin = violet,
// Member = blue, Viewer = slate.
const ROLE_COLORS: Record<OrgRole, {
  badge: string
  border: string
  countBg: string
}> = {
  owner: {
    badge: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300",
    border: "hover:border-amber-200 dark:hover:border-amber-500/30",
    countBg: "bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300",
  },
  admin: {
    badge: "bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-300",
    border: "hover:border-violet-200 dark:hover:border-violet-500/30",
    countBg: "bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300",
  },
  member: {
    badge: "bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-300",
    border: "hover:border-blue-200 dark:hover:border-blue-500/30",
    countBg: "bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300",
  },
  viewer: {
    badge: "bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
    border: "hover:border-slate-300 dark:hover:border-slate-500",
    countBg: "bg-slate-100 text-slate-600 dark:bg-slate-600 dark:text-slate-300",
  },
}

const FALLBACK_COLORS = ROLE_COLORS.viewer

export function AboutRolesPanel() {
  const t = useTranslations("team")
  const { data, isLoading } = useRoleDefinitions()
  const [sectionOpen, setSectionOpen] = useState(false)

  // Track which role cards are expanded. Multiple can be open.
  const [openRoles, setOpenRoles] = useState<Set<OrgRole>>(new Set())

  const toggleRole = useCallback((key: OrgRole) => {
    setOpenRoles((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }, [])

  const permissionByKey = useMemo<Map<string, RoleDefinitionPermission>>(() => {
    const map = new Map<string, RoleDefinitionPermission>()
    for (const p of data?.permissions ?? []) {
      map.set(p.key, p)
    }
    return map
  }, [data])

  return (
    <section className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
      {/* Section-level toggle */}
      <button
        type="button"
        onClick={() => setSectionOpen((v) => !v)}
        aria-expanded={sectionOpen}
        aria-controls="about-roles-content"
        className="flex w-full items-center justify-between gap-3 px-5 py-4 text-left rounded-xl hover:bg-slate-50 dark:hover:bg-slate-700/40 focus:outline-none focus:ring-2 focus:ring-rose-500/30 transition-colors"
      >
        <div className="flex items-center gap-3">
          <span className="flex h-9 w-9 items-center justify-center rounded-full bg-rose-50 dark:bg-rose-500/10 text-rose-600 dark:text-rose-400">
            <Info className="h-4 w-4" aria-hidden="true" />
          </span>
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              {t("aboutRoles.title")}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              {t("aboutRoles.subtitle")}
            </p>
          </div>
        </div>
        <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
          {sectionOpen ? t("aboutRoles.collapse") : t("aboutRoles.expand")}
          <ChevronDown
            className={`h-4 w-4 transition-transform duration-300 ${sectionOpen ? "rotate-180" : ""}`}
            aria-hidden="true"
          />
        </span>
      </button>

      {/* Animated section content via grid-rows trick */}
      <div
        id="about-roles-content"
        className="grid transition-[grid-template-rows] duration-300 ease-out"
        style={{ gridTemplateRows: sectionOpen ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">
          <div className="border-t border-slate-100 dark:border-slate-700 p-5">
            {isLoading || !data ? (
              <div className="space-y-3">
                {[0, 1, 2, 3].map((i) => (
                  <div
                    key={i}
                    className="h-[72px] rounded-xl border border-slate-100 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 animate-shimmer"
                  />
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                {data.roles.map((role) => (
                  <RoleAccordionCard
                    key={role.key}
                    role={role}
                    permissionByKey={permissionByKey}
                    isOpen={openRoles.has(role.key)}
                    onToggle={() => toggleRole(role.key)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  )
}

/* ------------------------------------------------------------------ */
/* Role accordion card                                                 */
/* ------------------------------------------------------------------ */

interface RoleAccordionCardProps {
  role: RoleDefinition
  permissionByKey: Map<string, RoleDefinitionPermission>
  isOpen: boolean
  onToggle: () => void
}

function RoleAccordionCard({
  role,
  permissionByKey,
  isOpen,
  onToggle,
}: RoleAccordionCardProps) {
  const t = useTranslations("team")
  const Icon = ROLE_ICONS[role.key] ?? Shield
  const colors = ROLE_COLORS[role.key] ?? FALLBACK_COLORS
  const cardId = `role-card-${role.key}`
  const panelId = `role-panel-${role.key}`

  const grouped = useMemo(() => {
    const map = new Map<string, RoleDefinitionPermission[]>()
    for (const key of role.permissions) {
      const meta = permissionByKey.get(key)
      if (!meta) continue
      const list = map.get(meta.group) ?? []
      list.push(meta)
      map.set(meta.group, list)
    }
    return map
  }, [role.permissions, permissionByKey])

  const localizedLabel = safeTranslate(t, `roles.${role.key}`, role.label)
  const localizedDescription = safeTranslate(
    t,
    `roleDescriptions.${role.key}`,
    role.description,
  )

  return (
    <article
      className={[
        "rounded-xl border bg-white dark:bg-slate-900/40",
        "transition-all duration-200 ease-out",
        isOpen
          ? "border-slate-200 dark:border-slate-600 shadow-md"
          : `border-slate-200 dark:border-slate-700 shadow-sm ${colors.border}`,
      ].join(" ")}
    >
      {/* Card header — always visible, acts as accordion trigger */}
      <button
        type="button"
        id={cardId}
        onClick={onToggle}
        aria-expanded={isOpen}
        aria-controls={panelId}
        className="flex w-full items-center gap-3 px-4 py-3.5 text-left rounded-xl focus:outline-none focus:ring-2 focus:ring-rose-500/30 transition-colors group"
      >
        <span
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full transition-all duration-200 ${colors.badge}`}
        >
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
              {localizedLabel}
            </h3>
            <span
              className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${colors.countBg}`}
            >
              {role.permissions.length}
            </span>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-1">
            {localizedDescription}
          </p>
        </div>

        <ChevronDown
          className={[
            "h-4 w-4 shrink-0 text-slate-400 dark:text-slate-500",
            "transition-transform duration-300 ease-out",
            isOpen ? "rotate-180" : "",
          ].join(" ")}
          aria-hidden="true"
        />
      </button>

      {/* Expandable permissions panel — CSS grid-rows animation */}
      <div
        id={panelId}
        role="region"
        aria-labelledby={cardId}
        className="grid transition-[grid-template-rows] duration-300 ease-out"
        style={{ gridTemplateRows: isOpen ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">
          <div className="px-4 pb-4 pt-1">
            {/* Full description when expanded */}
            <p className="text-xs text-slate-600 dark:text-slate-300 leading-relaxed mb-3">
              {localizedDescription}
            </p>

            {/* Permissions grouped by category */}
            <div className="space-y-3">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
                {t("aboutRoles.permissionsLabel")}
              </p>
              {grouped.size === 0 ? (
                <p className="text-xs text-slate-400 italic">—</p>
              ) : (
                <div className="space-y-2.5">
                  {Array.from(grouped.entries()).map(([group, perms]) => (
                    <PermissionGroup
                      key={group}
                      group={group}
                      permissions={perms}
                    />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </article>
  )
}

/* ------------------------------------------------------------------ */
/* Permission group                                                    */
/* ------------------------------------------------------------------ */

interface PermissionGroupProps {
  group: string
  permissions: RoleDefinitionPermission[]
}

function PermissionGroup({ group, permissions }: PermissionGroupProps) {
  const t = useTranslations("team")
  const localizedGroup = safeTranslate(t, `permissionGroups.${group}`, group)

  return (
    <div className="rounded-lg bg-slate-50 dark:bg-slate-800/60 px-3 py-2.5">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-rose-600 dark:text-rose-400 mb-1.5">
        {localizedGroup}
      </p>
      <ul className="space-y-1">
        {permissions.map((p) => (
          <li
            key={p.key}
            title={p.description}
            className="flex items-center gap-2 text-xs text-slate-700 dark:text-slate-300"
          >
            <Check
              className="h-3.5 w-3.5 shrink-0 text-emerald-500 dark:text-emerald-400"
              aria-hidden="true"
            />
            {p.label}
          </li>
        ))}
      </ul>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Utility                                                             */
/* ------------------------------------------------------------------ */

// safeTranslate looks up an i18n key and falls back to the inline
// English string if next-intl raises a missing-key error.
function safeTranslate(
  t: ReturnType<typeof useTranslations>,
  key: string,
  fallback: string,
): string {
  try {
    const value = t(key)
    if (!value || value === key) {
      return fallback
    }
    return value
  } catch {
    return fallback
  }
}
