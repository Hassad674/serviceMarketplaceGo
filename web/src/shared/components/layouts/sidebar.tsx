"use client"

import { useState, useEffect } from "react"
import {
  LayoutDashboard,
  UserCircle,
  Building2,
  LogOut,
  ArrowRightLeft,
  X,
  Sparkles,
  ChevronLeft,
  ChevronRight,
  Search,
  MessageSquare,
  FolderOpen,
  Briefcase,
  CreditCard,
  Wallet,
  Settings,
  FileText,
  Users2,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, usePathname, useRouter } from "@i18n/navigation"
import { useUser, useLogout, useOrganization } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { useUnreadCount } from "@/shared/hooks/use-unread-count"
import { cn } from "@/shared/lib/utils"

type NavItem = {
  labelKey: string
  href: string
  icon: React.ElementType
  exact?: boolean
  roles: string[]
  // Optional secondary gate on the current organization's `type`.
  // When present the entry is visible only if `organization.type`
  // matches one of the listed values — used for surfaces that are
  // scoped to an org-level persona (e.g. the client profile page is
  // hidden for `provider_personal` orgs).
  orgTypes?: string[]
}

// Freelance mode nav
const FREELANCE_NAV: NavItem[] = [
  { labelKey: "dashboard", href: "/dashboard", icon: LayoutDashboard, exact: true, roles: ["agency", "provider", "enterprise"] },
  { labelKey: "messages", href: "/messages", icon: MessageSquare, roles: ["agency", "provider", "enterprise"] },
  { labelKey: "projects", href: "/projects", icon: FolderOpen, roles: ["agency", "provider", "enterprise"] },
  { labelKey: "jobs", href: "/jobs", icon: Briefcase, roles: ["enterprise", "agency"] },
  { labelKey: "opportunities", href: "/opportunities", icon: Briefcase, roles: ["provider", "agency"] },
  { labelKey: "myApplications", href: "/my-applications", icon: FileText, roles: ["provider", "agency"] },
  { labelKey: "team", href: "/team", icon: Users2, roles: ["agency", "enterprise"] },
  // `providerProfile` replaces the legacy "myProfile" label. Agencies
  // now see both this entry AND the new client-profile entry below,
  // so the label has to be unambiguous about which side of the
  // marketplace identity it edits.
  { labelKey: "providerProfile", href: "/profile", icon: UserCircle, roles: ["agency", "provider"] },
  // Client profile: agencies and enterprises only. Solo providers
  // (org.type === "provider_personal") don't have a client-facing
  // identity and the entry must stay hidden for them.
  { labelKey: "clientProfile", href: "/client-profile", icon: Building2, roles: ["agency", "enterprise"], orgTypes: ["agency", "enterprise"] },
  { labelKey: "paymentInfo", href: "/payment-info", icon: CreditCard, roles: ["agency", "provider"] },
  { labelKey: "wallet", href: "/wallet", icon: Wallet, roles: ["agency", "provider"] },
  // Invoicing surfaces — visible to providers & agencies only. Enterprises
  // pay AGAINST these factures via Stripe but they don't need a self-serve
  // invoice list yet (Phase 7 ships only the operator-side experience).
  { labelKey: "invoices", href: "/invoices", icon: FileText, roles: ["agency", "provider"] },
  { labelKey: "findFreelancers", href: "/search?type=freelancer", icon: Search, roles: ["agency", "enterprise"] },
  { labelKey: "findAgencies", href: "/search?type=agency", icon: Search, roles: ["enterprise"] },
  { labelKey: "findReferrers", href: "/search?type=referrer", icon: Search, roles: ["agency", "enterprise"] },
  { labelKey: "accountSettings", href: "/account", icon: Settings, roles: ["agency", "provider", "enterprise"] },
]

// Referrer mode nav — no ?mode=referrer needed; cookie tracks the workspace
const REFERRER_NAV: NavItem[] = [
  { labelKey: "dashboard", href: "/dashboard", icon: LayoutDashboard, exact: true, roles: ["provider"] },
  { labelKey: "messages", href: "/messages", icon: MessageSquare, roles: ["provider"] },
  // /referrals (plural) is the deals dashboard for the apport d'affaires
  // feature. Distinct from /referral (singular) which stays the public
  // referrer profile editor below — both labels are intentionally close.
  { labelKey: "referralDeals", href: "/referrals", icon: Sparkles, roles: ["provider"] },
  { labelKey: "referrerProfile", href: "/referral", icon: UserCircle, roles: ["provider"] },
  // Wallet is where apporteurs see their commission income. Without
  // this entry they'd have to switch back to freelance mode to check
  // how much they've earned — which is the bug this line fixes.
  { labelKey: "paymentInfo", href: "/payment-info", icon: CreditCard, roles: ["provider"] },
  { labelKey: "wallet", href: "/wallet", icon: Wallet, roles: ["provider"] },
  { labelKey: "findFreelancers", href: "/search?type=freelancer", icon: Search, roles: ["provider"] },
  { labelKey: "accountSettings", href: "/account", icon: Settings, roles: ["provider"] },
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

function getFilteredNav(
  role: string,
  orgType: string | undefined,
  isReferrerMode: boolean,
): NavItem[] {
  const filterByOrgType = (item: NavItem) =>
    !item.orgTypes || (orgType ? item.orgTypes.includes(orgType) : false)
  if (role === "provider" && isReferrerMode) {
    return REFERRER_NAV.filter(
      (item) => item.roles.includes(role) && filterByOrgType(item),
    )
  }
  return FREELANCE_NAV.filter(
    (item) => item.roles.includes(role) && filterByOrgType(item),
  )
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
  const router = useRouter()
  const { data: user } = useUser()
  const { data: org } = useOrganization()
  const logout = useLogout()
  const { isReferrerMode, setReferrerMode, switchToReferrer, switchToFreelance } = useWorkspace()
  const { data: unreadData } = useUnreadCount()
  const t = useTranslations("sidebar")
  const tCommon = useTranslations("common")

  const role = user?.role ?? ""
  // Sync workspace to referrer when visiting the /referral page (mount-only)
  useEffect(() => {
    if (pathname === "/referral") {
      setReferrerMode(true)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname])

  const items = getFilteredNav(role, org?.type, isReferrerMode)
  const displayRole = isReferrerMode ? "referrer" : role

  async function handleLogout() {
    await logout()
  }

  // Defensive: some accounts (legacy data or partial fetch) may have
  // empty or undefined name fields. Fall back gracefully so the sidebar
  // never crashes the entire dashboard over a missing initial.
  const firstInitial = user?.first_name?.charAt(0) ?? ""
  const lastInitial = user?.last_name?.charAt(0) ?? ""
  const initials = (firstInitial + lastInitial).toUpperCase() || "?"

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
            <ReferrerSwitch
              isReferrerMode={isReferrerMode}
              collapsed={collapsed}
              onToggle={() => {
                if (!isReferrerMode) {
                  const targetPath = switchToReferrer()
                  router.push(targetPath)
                } else {
                  const targetPath = switchToFreelance()
                  router.push(targetPath)
                }
              }}
            />
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
              onClick={onClose}
              collapsed={collapsed}
              badge={item.labelKey === "messages" ? (unreadData?.count ?? 0) : 0}
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
  onToggle,
}: {
  isReferrerMode: boolean
  collapsed: boolean
  onToggle: () => void
}) {
  const t = useTranslations("sidebar")

  if (collapsed) {
    const dotColor = isReferrerMode ? "bg-emerald-500" : "bg-amber-500"
    return (
      <button
        onClick={onToggle}
        className="flex w-full items-center justify-center rounded-lg p-2 transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
        aria-label={isReferrerMode ? t("freelanceDashboard") : t("businessReferrer")}
      >
        <span className={cn("h-3 w-3 rounded-full", dotColor)} />
      </button>
    )
  }

  if (isReferrerMode) {
    return (
      <button
        onClick={onToggle}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2",
          "text-sm font-medium transition-all duration-200",
          "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-500/15 dark:text-emerald-400 dark:hover:bg-emerald-500/25",
        )}
      >
        <ArrowRightLeft className="h-4 w-4" strokeWidth={1.5} />
        {t("freelanceDashboard")}
      </button>
    )
  }

  return (
    <button
      onClick={onToggle}
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2",
        "text-sm font-medium text-white transition-all duration-200",
        "gradient-referrer hover:opacity-90 hover:shadow-md",
      )}
    >
      <Sparkles className="h-4 w-4" strokeWidth={1.5} />
      {t("businessReferrer")}
    </button>
  )
}

function NavLink({
  item,
  label,
  pathname,
  onClick,
  collapsed,
  badge = 0,
}: {
  item: NavItem
  label: string
  pathname: string
  onClick?: () => void
  collapsed: boolean
  badge?: number
}) {
  // Track browser search params reactively.
  // pathname alone is not enough — /search?type=freelancer → /search?type=agency
  // has the same pathname but different search params.
  // Use href from the Link component's click to detect the full URL change.
  const [currentSearch, setCurrentSearch] = useState("")

  useEffect(() => {
    const updateSearch = () => setCurrentSearch(window.location.search)
    updateSearch()

    // Listen for popstate (back/forward) and custom event for client-side nav
    window.addEventListener("popstate", updateSearch)

    // MutationObserver on the URL is not possible, so use a short interval
    // that only runs while the component is mounted (cleans up on unmount)
    const interval = setInterval(updateSearch, 300)

    return () => {
      window.removeEventListener("popstate", updateSearch)
      clearInterval(interval)
    }
  }, [pathname])

  const [hrefPath, hrefQuery] = item.href.split("?")
  const hrefParams = new URLSearchParams(hrefQuery ?? "")
  const currentParams = new URLSearchParams(currentSearch)

  const pathMatches = item.exact
    ? pathname === hrefPath
    : pathname === hrefPath || pathname.startsWith(hrefPath + "/")

  // If href has query params, ALL must match in current URL
  const queryMatches = !hrefQuery || Array.from(hrefParams.entries()).every(
    ([key, value]) => currentParams.get(key) === value,
  )

  const isActive = pathMatches && queryMatches

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
      {!collapsed && <span className="flex-1">{label}</span>}
      {badge > 0 && (
        <span
          className={cn(
            "flex h-5 min-w-5 items-center justify-center rounded-full bg-rose-500 px-1.5 text-[10px] font-bold text-white",
            collapsed && "absolute -right-0.5 -top-0.5 h-4 min-w-4 px-1",
          )}
        >
          {badge > 99 ? "99+" : badge}
        </span>
      )}
    </Link>
  )
}

export { STORAGE_KEY as SIDEBAR_STORAGE_KEY }
