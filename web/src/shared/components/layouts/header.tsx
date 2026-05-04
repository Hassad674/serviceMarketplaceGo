"use client"

import { useState, useRef, useEffect } from "react"
import { Menu, Search, LogOut, User, ChevronDown } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { useUser, useLogout } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { ThemeToggle } from "@/shared/components/theme-toggle"
import { NotificationBell } from "@/features/notification/components/notification-bell"
import { SubscriptionBadge } from "@/features/subscription/components/subscription-badge"
import { UpgradeModal } from "@/features/subscription/components/upgrade-modal"
import { ManageModal } from "@/features/subscription/components/manage-modal"
import { useSubscription } from "@/features/subscription/hooks/use-subscription"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
import { Portrait } from "@/shared/components/ui/portrait"

const ROLE_LABEL_KEYS: Record<string, string> = {
  agency: "roleAgency",
  enterprise: "roleEnterprise",
  provider: "roleProvider",
  referrer: "roleReferrer",
}

// Soleil-aware role badge tones — same scale as the sidebar.
const ROLE_BADGE_TONES: Record<string, string> = {
  agency: "bg-primary-soft text-primary-deep",
  enterprise: "bg-pink-soft text-primary-deep",
  provider: "bg-success-soft text-success",
  referrer: "bg-amber-soft text-foreground",
}

const ROLE_PORTRAIT_ID: Record<string, number> = {
  agency: 0,
  enterprise: 4,
  provider: 1,
  referrer: 3,
}

type HeaderProps = {
  onMenuToggle?: () => void
}

export function Header({ onMenuToggle }: HeaderProps) {
  const { data: user } = useUser()
  const logout = useLogout()
  const { isReferrerMode } = useWorkspace()
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const tCommon = useTranslations("common")
  const tSidebar = useTranslations("sidebar")
  const displayRole = (user?.role === "provider" && isReferrerMode) ? "referrer" : (user?.role ?? "")
  const portraitId = ROLE_PORTRAIT_ID[displayRole] ?? 0

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [])

  async function handleLogout() {
    setDropdownOpen(false)
    await logout()
  }

  const profileHref = (user?.role === "provider" && isReferrerMode) ? "/referral" : "/profile"

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-border bg-card px-4 sm:px-5">
      {/* Mobile menu */}
      <Button variant="ghost" size="auto"
        onClick={onMenuToggle}
        className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-background hover:text-foreground lg:hidden"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5" strokeWidth={1.5} />
      </Button>

      {/* Search — Soleil pill */}
      <div className="relative hidden flex-1 sm:block sm:max-w-sm">
        <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" strokeWidth={1.5} />
        <input
          type="text"
          placeholder={tCommon("search")}
          className={cn(
            "h-10 w-full rounded-full border border-border bg-background pl-10 pr-4 text-sm text-foreground",
            "placeholder:text-muted-foreground transition-all duration-150",
            "focus:border-primary focus:bg-card focus:outline-none focus:ring-4 focus:ring-primary/15",
          )}
        />
      </div>

      <div className="ml-auto flex items-center gap-1.5">
        {/* Premium badge — provider-only (hidden for enterprise/referrer) */}
        <SubscriptionSlot role={user?.role} />

        {/* Theme toggle */}
        <ThemeToggle className="rounded-lg border-0 bg-transparent shadow-none hover:shadow-none hover:bg-background" />

        {/* Notifications */}
        <NotificationBell />

        {/* User dropdown */}
        {user && (
          <div className="relative" ref={dropdownRef}>
            <Button variant="ghost" size="auto"
              onClick={() => setDropdownOpen((prev) => !prev)}
              className="flex items-center gap-2 rounded-full p-1 transition-all duration-200 hover:bg-background"
            >
              <Portrait id={portraitId} size={32} alt="" />
              <ChevronDown
                className={cn(
                  "hidden h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 sm:block",
                  dropdownOpen && "rotate-180",
                )}
                strokeWidth={1.5}
              />
            </Button>

            {/* Dropdown */}
            {dropdownOpen && (
              <div className="animate-scale-in absolute right-0 top-full z-50 mt-1.5 w-60 overflow-hidden rounded-xl border border-border bg-card shadow-[var(--shadow-card)]">
                <div className="border-b border-border p-3">
                  <p className="text-sm font-semibold text-foreground">{user.display_name}</p>
                  <p className="mt-0.5 text-xs text-muted-foreground">{user.email}</p>
                  <span
                    className={cn(
                      "mt-1.5 inline-block rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                      ROLE_BADGE_TONES[displayRole] ?? "bg-muted text-muted-foreground",
                    )}
                  >
                    {ROLE_LABEL_KEYS[displayRole] ? tSidebar(ROLE_LABEL_KEYS[displayRole]) : displayRole}
                  </span>
                </div>
                <div className="p-1">
                  <Link
                    href={profileHref}
                    onClick={() => setDropdownOpen(false)}
                    className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-background hover:text-foreground"
                  >
                    <User className="h-4 w-4" strokeWidth={1.5} />
                    {tSidebar("myProfile")}
                  </Link>
                  <div className="my-0.5 border-t border-border" />
                  <Button variant="ghost" size="auto"
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-primary-soft hover:text-primary-deep"
                  >
                    <LogOut className="h-4 w-4" strokeWidth={1.5} />
                    {tCommon("signOut")}
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </header>
  )
}

/**
 * Local composition that decides which subscription modal (upgrade
 * vs manage) the badge should open. Kept in the header file so the
 * shared layout stays the single source of truth for navbar chrome.
 *
 * Gating:
 *   - provider → freelance plan, badge visible
 *   - agency   → agency plan, badge visible
 *   - enterprise / no user → badge hidden (returns null)
 */
function SubscriptionSlot({ role }: { role?: string }) {
  const [showUpgrade, setShowUpgrade] = useState(false)
  const [showManage, setShowManage] = useState(false)
  const { data: subscription } = useSubscription()

  if (role !== "provider" && role !== "agency") return null

  const plan: "freelance" | "agency" = role === "agency" ? "agency" : "freelance"
  const handleClick = () => {
    if (subscription) setShowManage(true)
    else setShowUpgrade(true)
  }

  return (
    <>
      <SubscriptionBadge onClick={handleClick} />
      <UpgradeModal
        open={showUpgrade}
        role={plan}
        onClose={() => setShowUpgrade(false)}
      />
      <ManageModal
        open={showManage}
        onClose={() => setShowManage(false)}
      />
    </>
  )
}
