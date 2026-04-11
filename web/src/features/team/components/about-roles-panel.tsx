"use client"

import { useMemo, useState } from "react"
import { ChevronDown, ChevronUp, Crown, Info, Shield, ShieldCheck, Eye } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRoleDefinitions } from "../hooks/use-team"
import type { OrgRole, RoleDefinition, RoleDefinitionPermission } from "../types"

// Collapsible "About roles" panel rendered on the team page. Lists
// every role with its description and the permissions it grants,
// grouped by resource family. Lives directly in the team page (not
// behind a modal) so users can scan it without breaking their flow.
//
// Responsive: stacks to a single column on mobile (< 768px) and
// shows a 2-column role grid on desktop. The expand/collapse button
// is the only interaction — keyboard accessible, focus indicator
// preserved.

const ROLE_ICONS: Record<OrgRole, typeof Shield> = {
  owner: Crown,
  admin: ShieldCheck,
  member: Shield,
  viewer: Eye,
}

// Color tokens applied to the role icon circle. Owner stays amber
// (matches the crown badge in the members list); other roles use
// the design system's neutral slate so the panel feels calm.
const ROLE_BADGE_CLASSES: Record<OrgRole, string> = {
  owner: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300",
  admin: "bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-300",
  member: "bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-300",
  viewer: "bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
}

export function AboutRolesPanel() {
  const t = useTranslations("team")
  const { data, isLoading } = useRoleDefinitions()
  const [expanded, setExpanded] = useState(false)

  // Build a quick lookup so each role's permission keys can be
  // resolved against the catalogue without re-iterating on every
  // render.
  const permissionByKey = useMemo<Map<string, RoleDefinitionPermission>>(() => {
    const map = new Map<string, RoleDefinitionPermission>()
    for (const p of data?.permissions ?? []) {
      map.set(p.key, p)
    }
    return map
  }, [data])

  return (
    <section className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
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
          {expanded ? t("aboutRoles.collapse") : t("aboutRoles.expand")}
          {expanded ? (
            <ChevronUp className="h-4 w-4" aria-hidden="true" />
          ) : (
            <ChevronDown className="h-4 w-4" aria-hidden="true" />
          )}
        </span>
      </button>

      {expanded && (
        <div
          id="about-roles-content"
          className="border-t border-slate-100 dark:border-slate-700 p-5 animate-fade-in"
        >
          {isLoading || !data ? (
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-32 rounded-xl border border-slate-100 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 animate-shimmer"
                />
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              {data.roles.map((role) => (
                <RoleCard
                  key={role.key}
                  role={role}
                  permissionByKey={permissionByKey}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  )
}

type RoleCardProps = {
  role: RoleDefinition
  permissionByKey: Map<string, RoleDefinitionPermission>
}

function RoleCard({ role, permissionByKey }: RoleCardProps) {
  const t = useTranslations("team")
  const Icon = ROLE_ICONS[role.key] ?? Shield
  const badgeClass = ROLE_BADGE_CLASSES[role.key] ?? ROLE_BADGE_CLASSES.viewer

  // Group permissions by their resource family so the card reads
  // like a small table of contents instead of a flat 20-item list.
  const grouped = useMemo(() => {
    const map = new Map<string, RoleDefinitionPermission[]>()
    for (const key of role.permissions) {
      const meta = permissionByKey.get(key)
      if (!meta) {
        continue
      }
      const list = map.get(meta.group) ?? []
      list.push(meta)
      map.set(meta.group, list)
    }
    return map
  }, [role.permissions, permissionByKey])

  // Translate role label/description with i18n; backend keeps the
  // English string as a fallback so a freshly-deployed permission
  // shows up immediately even before the frontend ships its catalogue
  // update.
  const localizedLabel = safeTranslate(t, `roles.${role.key}`, role.label)
  const localizedDescription = safeTranslate(
    t,
    `roleDescriptions.${role.key}`,
    role.description,
  )

  return (
    <article className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900/40 p-4 flex flex-col gap-3">
      <header className="flex items-center gap-3">
        <span className={`flex h-10 w-10 items-center justify-center rounded-full ${badgeClass}`}>
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
        <div>
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">{localizedLabel}</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">{role.permissions.length} permissions</p>
        </div>
      </header>
      <p className="text-xs text-slate-600 dark:text-slate-300 leading-relaxed">{localizedDescription}</p>
      <div>
        <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400 mb-2">
          {t("aboutRoles.permissionsLabel")}
        </p>
        {grouped.size === 0 ? (
          <p className="text-xs text-slate-400">—</p>
        ) : (
          <ul className="space-y-2">
            {Array.from(grouped.entries()).map(([group, perms]) => (
              <li key={group}>
                <p className="text-[11px] font-semibold uppercase tracking-wide text-rose-600 dark:text-rose-400 mb-1">
                  {safeTranslate(t, `permissionGroups.${group}`, group)}
                </p>
                <ul className="ml-3 list-disc space-y-0.5 text-xs text-slate-700 dark:text-slate-300">
                  {perms.map((p) => (
                    <li key={p.key} title={p.description}>
                      {p.label}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        )}
      </div>
    </article>
  )
}

// safeTranslate looks up an i18n key and falls back to the inline
// English string if next-intl raises a missing-key error. This is
// the only safe way to render a backend-supplied label that may be
// ahead of the frontend's catalogue.
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
