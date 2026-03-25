"use client"

import { useState, useEffect, useCallback } from "react"
import {
  LayoutDashboard,
  UserCircle,
  LogOut,
  ArrowRightLeft,
  X,
  Sparkles,
  ChevronLeft,
  ChevronRight,
  Search,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, usePathname, useRouter } from "@i18n/navigation"
import { useSearchParams } from "next/navigation"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

type NavItem = {
  labelKey: string
  href: string
  icon: React.ElementType
  exact?: boolean
  roles: string[]
  referrerOnly?: boolean
  hideInReferrerMode?: boolean
}

const NAV_ITEMS: NavItem[] = [
  { labelKey: "dashboard", href: "/dashboard", icon: LayoutDashboard, exact: true, roles: ["agency", "provider", "enterprise"] },
  { labelKey: "myProfile", href: "/profile", icon: UserCircle, roles: ["agency", "provider"], hideInReferrerMode: true },
  { labelKey: "referrerProfile", href: "/referral", icon: UserCircle, roles: ["provider"], referrerOnly: true },
  { labelKey: "findFreelancers", href: "/search?type=freelancer", icon: Search, roles: ["agency", "enterprise", "provider"] },
  { labelKey: "findAgencies", href: "/search?type=agency", icon: Search, roles: ["enterprise"] },
  { labelKey: "findReferrers", href: "/search?type=referrer", icon: Search, roles: ["agency", "enterprise"] },
]

const ROLE_LABEL_KEYS: Record<string, string> = {
  agency: "roleAgency",
  enterprise: "roleEnterprise",
  provider: "roleProvider",
  referrer: "roleReferrer",
}

const ROLE_COLORS: Record<string, string> = {
  agency: "bg-blue-50 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400",
  enterprise: "bg-purple-50 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400",
  provider: "bg-rose-50 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400",
  referrer: "bg-amber-50 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400",
}

function getFilteredNav(role: string, isReferrerMode: boolean): NavItem[] {
  return NAV_ITEMS.filter((item) => {
    if (!item.roles.includes(role)) return false
    if (item.referrerOnly && !isReferrerMode) return false
    if (item.hideInReferrerMode && isReferrerMode) return false
    return true
  })
}

const STORAGE_KEY = "sidebar-collapsed"

type SidebarProps = {
  open?: boolean
  onClose?: () => void
  collapsed?: boolean
  onToggleCollapse?: () => void
}

export function Sidebar({ open, onClose, collapsed = false, onToggleCollapse }: SidebarProps) {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const router = useRouter()
  const { user, logout } = useAuth()
  const t = useTranslations("sidebar")
  const tCommon = useTranslations("common")

  // Read role from cookie first (instant, no hydration delay), fallback to Zustand
  const role = user?.role ?? (() => {
    if (typeof document === "undefined") return ""
    const match = document.cookie.match(/user_role=(\w+)/)
    return match?.[1] ?? ""
  })()
  const isReferrerMode = pathname === "/referral" || searchParams.get("mode") === "referrer"
  const items = getFilteredNav(role, isReferrerMode)
  const displayRole = isReferrerMode ? "referrer" : role

  function handleLogout() {
    logout()
    router.push("/login")
  }

  const initials = user
    ? `${user.first_name.charAt(0)}${user.last_name.charAt(0)}`
    : "?"

  return (
    <>
      {/* Mobile glass overlay */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/20 backdrop-blur-sm lg:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex flex-col",
          "bg-white/80 dark:bg-gray-900/90 backdrop-blur-xl border-r border-gray-100/50 dark:border-gray-800/50",
          "lg:static lg:z-auto",
          "transition-all duration-300 ease-out lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full",
          collapsed ? "w-[72px]" : "w-[280px]",
          "lg:flex",
        )}
      >
        {/* Logo */}
        <div className="flex h-14 items-center justify-between px-4">
          <Link href="/" className="flex items-center gap-2 overflow-hidden">
            {collapsed ? (
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-r from-rose-500 to-purple-600 text-sm font-bold text-white">
                M
              </span>
            ) : (
              <span className="bg-gradient-to-r from-rose-500 to-purple-600 bg-clip-text text-lg font-bold tracking-tight text-transparent">
                Atelier
              </span>
            )}
          </Link>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-gray-400 dark:text-gray-500 transition-colors hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-600 dark:hover:text-gray-300 lg:hidden"
            aria-label="Close menu"
          >
            <X className="h-4 w-4" strokeWidth={1.5} />
          </button>
        </div>

        {/* User info */}
        <div className={cn("mx-3 mb-2 rounded-xl bg-gray-50/80 dark:bg-gray-800/50", collapsed ? "p-2" : "p-3")}>
          <div className={cn("flex items-center", collapsed ? "justify-center" : "gap-3")}>
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-xs font-semibold text-white">
              {initials}
            </div>
            {!collapsed && (
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">
                  {user?.display_name ?? "User"}
                </p>
                <span
                  className={cn(
                    "mt-0.5 inline-block rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                    ROLE_COLORS[displayRole] ?? "bg-gray-100 text-gray-600",
                  )}
                >
                  {ROLE_LABEL_KEYS[displayRole] ? t(ROLE_LABEL_KEYS[displayRole]) : displayRole}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Role switch (provider only) */}
        {role === "provider" && (
          <div className={cn("pb-2", collapsed ? "px-3" : "px-4")}>
            <ReferrerSwitch isReferrerMode={isReferrerMode} collapsed={collapsed} />
          </div>
        )}

        {/* Navigation */}
        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-2">
          {items.map((item) => (
            <NavLink
              key={item.labelKey}
              item={item}
              label={t(item.labelKey)}
              pathname={pathname}
              searchParams={searchParams}
              onClick={onClose}
              collapsed={collapsed}
            />
          ))}
        </nav>

        {/* Collapse toggle (desktop only) */}
        <div className="hidden border-t border-gray-100/80 dark:border-gray-800 p-2 lg:block">
          <button
            onClick={onToggleCollapse}
            className={cn(
              "flex w-full items-center rounded-lg px-3 py-2 text-sm",
              "text-gray-400 dark:text-gray-500 transition-all duration-200 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-600 dark:hover:text-gray-300",
              collapsed ? "justify-center" : "gap-3",
            )}
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" strokeWidth={1.5} />
            ) : (
              <>
                <ChevronLeft className="h-4 w-4" strokeWidth={1.5} />
                <span>{t("collapse")}</span>
              </>
            )}
          </button>
        </div>

        {/* Logout */}
        <div className="border-t border-gray-100/80 dark:border-gray-800 p-2">
          <button
            onClick={handleLogout}
            className={cn(
              "flex w-full items-center rounded-lg px-3 py-2 text-sm",
              "text-gray-500 dark:text-gray-400 transition-all duration-200 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-700 dark:hover:text-gray-200",
              collapsed ? "justify-center" : "gap-3",
            )}
            aria-label={tCommon("signOut")}
          >
            <LogOut className="h-[18px] w-[18px] shrink-0" strokeWidth={1.5} />
            {!collapsed && <span>{tCommon("signOut")}</span>}
          </button>
        </div>
      </aside>
    </>
  )
}

function ReferrerSwitch({
  isReferrerMode,
  collapsed,
}: {
  isReferrerMode: boolean
  collapsed: boolean
}) {
  const t = useTranslations("sidebar")

  if (collapsed) {
    const href = isReferrerMode ? "/dashboard" : "/dashboard?mode=referrer"
    const dotColor = isReferrerMode ? "bg-emerald-500" : "bg-amber-500"
    return (
      <Link
        href={href}
        className="flex items-center justify-center rounded-lg p-2 transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
        aria-label={isReferrerMode ? t("freelanceDashboard") : t("businessReferrer")}
      >
        <span className={cn("h-3 w-3 rounded-full", dotColor)} />
      </Link>
    )
  }

  if (isReferrerMode) {
    return (
      <Link
        href="/dashboard"
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2",
          "text-sm font-medium transition-all duration-200",
          "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-500/15 dark:text-emerald-400 dark:hover:bg-emerald-500/25",
        )}
      >
        <ArrowRightLeft className="h-4 w-4" strokeWidth={1.5} />
        {t("freelanceDashboard")}
      </Link>
    )
  }

  return (
    <Link
      href="/dashboard?mode=referrer"
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2",
        "text-sm font-medium text-white transition-all duration-200",
        "gradient-referrer hover:opacity-90 hover:shadow-md",
      )}
    >
      <Sparkles className="h-4 w-4" strokeWidth={1.5} />
      {t("businessReferrer")}
    </Link>
  )
}

function NavLink({
  item,
  label,
  pathname,
  searchParams,
  onClick,
  collapsed,
}: {
  item: NavItem
  label: string
  pathname: string
  searchParams: URLSearchParams
  onClick?: () => void
  collapsed: boolean
}) {
  // For hrefs with query params (e.g. /search?type=freelancer), compare path + query
  const [hrefPath, hrefQuery] = item.href.split("?")
  const currentQuery = searchParams.toString()

  let isActive = false
  if (item.exact) {
    isActive = pathname === hrefPath && (!hrefQuery || currentQuery.includes(hrefQuery))
  } else {
    const pathMatches = pathname === hrefPath || pathname.startsWith(hrefPath + "/")
    const queryMatches = !hrefQuery || currentQuery.includes(hrefQuery)
    isActive = pathMatches && queryMatches
  }

  return (
    <Link
      href={item.href}
      onClick={onClick}
      title={collapsed ? label : undefined}
      className={cn(
        "relative flex items-center rounded-lg py-2 text-sm transition-all duration-200",
        collapsed ? "justify-center px-2" : "gap-3 px-3",
        isActive
          ? "bg-rose-50 dark:bg-rose-500/10 font-medium text-rose-600 dark:text-rose-400"
          : "text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white",
      )}
    >
      {/* Active indicator pill */}
      {isActive && !collapsed && (
        <span className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-full bg-rose-500" />
      )}
      {isActive && collapsed && (
        <span className="absolute left-1 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-full bg-rose-500" />
      )}
      <item.icon className="h-[18px] w-[18px] shrink-0" strokeWidth={1.5} />
      {!collapsed && <span>{label}</span>}
    </Link>
  )
}

export { STORAGE_KEY as SIDEBAR_STORAGE_KEY }
