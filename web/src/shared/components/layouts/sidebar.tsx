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

import { Button } from "@/shared/components/ui/button"
import { Portrait } from "@/shared/components/ui/portrait"

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

// Soleil-aware role badge tones. All four roles share the same warm
// palette — they're distinguishable by saturation, not by hue family,
// keeping the marketplace identity coherent.
const ROLE_BADGE_TONES: Record<string, string> = {
  agency: "bg-primary-soft text-primary-deep",
  enterprise: "bg-pink-soft text-primary-deep",
  provider: "bg-success-soft text-success",
  referrer: "bg-amber-soft text-foreground",
}

// Deterministic Portrait id by role — every avatar in the sidebar
// stays in the warm Soleil palette but each role gets a distinct tone.
const ROLE_PORTRAIT_ID: Record<string, number> = {
  agency: 0,    // corail
  enterprise: 4, // lilas
  provider: 1,  // vert olive
  referrer: 3,  // ambre
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
  const portraitId = ROLE_PORTRAIT_ID[displayRole] ?? 0

  async function handleLogout() {
    await logout()
  }

  return (
    <>
      {/* Mobile overlay */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-foreground/20 backdrop-blur-sm lg:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex flex-col",
          "bg-card border-r border-border",
          "lg:static lg:z-auto",
          "transition-all duration-300 ease-out lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full",
          collapsed ? "w-[72px]" : "w-[256px]",
          "lg:flex",
        )}
      >
        {/* Brand */}
        <div className="flex h-14 items-center justify-between px-4">
          <Link href="/" className="flex items-center gap-2.5 overflow-hidden">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary font-serif text-base font-semibold italic text-primary-foreground">
              a
            </span>
            {!collapsed && (
              <span className="font-serif text-xl font-medium tracking-tight text-foreground">
                Atelier
              </span>
            )}
          </Link>
          <Button variant="ghost" size="auto"
            onClick={onClose}
            className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-primary-soft hover:text-foreground lg:hidden"
            aria-label="Close menu"
          >
            <X className="h-4 w-4" strokeWidth={1.5} />
          </Button>
        </div>

        {/* User info */}
        <div className={cn("mx-3 mb-2 rounded-xl bg-background", collapsed ? "p-2" : "p-3")}>
          <div className={cn("flex items-center", collapsed ? "justify-center" : "gap-3")}>
            <Portrait id={portraitId} size={36} alt="" />
            {!collapsed && (
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-foreground">
                  {user?.display_name ?? "User"}
                </p>
                <span
                  className={cn(
                    "mt-0.5 inline-block rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                    ROLE_BADGE_TONES[displayRole] ?? "bg-muted text-muted-foreground",
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
        <div className="hidden border-t border-border p-2 lg:block">
          <Button variant="ghost" size="auto"
            onClick={onToggleCollapse}
            className={cn(
              "flex w-full items-center rounded-lg px-3 py-2 text-sm",
              "text-muted-foreground transition-all duration-200 hover:bg-background hover:text-foreground",
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
          </Button>
        </div>

        {/* Logout */}
        <div className="border-t border-border p-2">
          <Button variant="ghost" size="auto"
            onClick={handleLogout}
            className={cn(
              "flex w-full items-center rounded-lg px-3 py-2 text-sm",
              "text-muted-foreground transition-all duration-200 hover:bg-background hover:text-foreground",
              collapsed ? "justify-center" : "gap-3",
            )}
            aria-label={tCommon("signOut")}
          >
            <LogOut className="h-[18px] w-[18px] shrink-0" strokeWidth={1.5} />
            {!collapsed && <span>{tCommon("signOut")}</span>}
          </Button>
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
    // Soleil-aware dot marker — corail when active referrer, sapin when freelance.
    const dotColor = isReferrerMode ? "bg-primary" : "bg-success"
    return (
      <Button variant="ghost" size="auto"
        onClick={onToggle}
        className="flex w-full items-center justify-center rounded-lg p-2 transition-colors hover:bg-background"
        aria-label={isReferrerMode ? t("freelanceDashboard") : t("businessReferrer")}
      >
        <span className={cn("h-3 w-3 rounded-full", dotColor)} />
      </Button>
    )
  }

  if (isReferrerMode) {
    return (
      <Button variant="ghost" size="auto"
        onClick={onToggle}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-full px-3 py-2",
          "text-sm font-medium transition-all duration-200",
          "bg-success-soft text-success hover:opacity-80",
        )}
      >
        <ArrowRightLeft className="h-4 w-4" strokeWidth={1.5} />
        {t("freelanceDashboard")}
      </Button>
    )
  }

  return (
    <Button variant="ghost" size="auto"
      onClick={onToggle}
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-full px-3 py-2",
        "text-sm font-medium text-foreground transition-all duration-200",
        "gradient-coral hover:opacity-90",
      )}
    >
      <Sparkles className="h-4 w-4" strokeWidth={1.5} />
      {t("businessReferrer")}
    </Button>
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
          ? "bg-primary-soft font-medium text-primary"
          : "text-muted-foreground hover:bg-background hover:text-foreground",
      )}
    >
      {/* Active indicator pill */}
      {isActive && !collapsed && (
        <span className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-full bg-primary" />
      )}
      {isActive && collapsed && (
        <span className="absolute left-1 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-full bg-primary" />
      )}
      <item.icon className="h-[18px] w-[18px] shrink-0" strokeWidth={1.5} />
      {!collapsed && <span className="flex-1">{label}</span>}
      {badge > 0 && (
        <span
          className={cn(
            "flex h-5 min-w-5 items-center justify-center rounded-full bg-primary px-1.5 text-[10px] font-bold text-primary-foreground",
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
