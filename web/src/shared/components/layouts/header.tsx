"use client"

import { useState, useRef, useEffect } from "react"
import { Menu, Search, LogOut, User, ChevronDown } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { useUser, useLogout } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { ThemeToggle } from "@/shared/components/theme-toggle"
import { NotificationBell } from "@/features/notification/components/notification-bell"
import { cn } from "@/shared/lib/utils"

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

  // Defensive: some accounts (legacy data or partial fetch) may have
  // empty or undefined name fields. Fall back gracefully so the header
  // never crashes the entire dashboard over a missing initial.
  const firstInitial = user?.first_name?.charAt(0) ?? ""
  const lastInitial = user?.last_name?.charAt(0) ?? ""
  const initials = (firstInitial + lastInitial).toUpperCase() || "?"

  const profileHref = (user?.role === "provider" && isReferrerMode) ? "/referral" : "/profile"

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-gray-100/50 bg-white/80 px-4 backdrop-blur-xl dark:border-slate-700/50 dark:bg-slate-900/80 sm:px-5">
      {/* Mobile menu */}
      <button
        onClick={onMenuToggle}
        className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200 lg:hidden"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5" strokeWidth={1.5} />
      </button>

      {/* Search */}
      <div className="relative hidden flex-1 sm:block sm:max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-slate-400" strokeWidth={1.5} />
        <input
          type="text"
          placeholder={tCommon("search")}
          className={cn(
            "h-9 w-full rounded-full bg-gray-100/80 pl-9 pr-4 text-sm text-gray-900",
            "placeholder:text-gray-400 transition-all duration-200",
            "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
            "dark:bg-slate-800/80 dark:text-slate-100 dark:placeholder:text-slate-400",
            "dark:focus:bg-slate-800 dark:focus:ring-rose-400/20",
          )}
        />
      </div>

      <div className="ml-auto flex items-center gap-1.5">
        {/* Theme toggle */}
        <ThemeToggle className="rounded-lg border-0 bg-transparent shadow-none hover:shadow-none hover:bg-gray-100 dark:hover:bg-slate-700" />

        {/* Notifications */}
        <NotificationBell />

        {/* User dropdown */}
        {user && (
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setDropdownOpen((prev) => !prev)}
              className="flex items-center gap-2 rounded-lg p-1.5 transition-all duration-200 hover:bg-gray-50 dark:hover:bg-slate-800"
            >
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-xs font-semibold text-white">
                {initials}
              </div>
              <ChevronDown
                className={cn(
                  "hidden h-3.5 w-3.5 text-gray-400 transition-transform duration-200 sm:block",
                  dropdownOpen && "rotate-180",
                )}
                strokeWidth={1.5}
              />
            </button>

            {/* Dropdown */}
            {dropdownOpen && (
              <div className="animate-scale-in absolute right-0 top-full z-50 mt-1.5 w-60 overflow-hidden rounded-xl border border-gray-100/80 bg-white/90 shadow-lg backdrop-blur-xl dark:border-slate-700/80 dark:bg-slate-800/90">
                <div className="border-b border-gray-100 p-3 dark:border-slate-700">
                  <p className="text-sm font-semibold text-gray-900 dark:text-slate-100">{user.display_name}</p>
                  <p className="mt-0.5 text-xs text-gray-500 dark:text-slate-400">{user.email}</p>
                  <span
                    className={cn(
                      "mt-1.5 inline-block rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                      ROLE_COLORS[displayRole] ?? "bg-gray-100 text-gray-600",
                    )}
                  >
                    {ROLE_LABEL_KEYS[displayRole] ? tSidebar(ROLE_LABEL_KEYS[displayRole]) : displayRole}
                  </span>
                </div>
                <div className="p-1">
                  <Link
                    href={profileHref}
                    onClick={() => setDropdownOpen(false)}
                    className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-50 hover:text-gray-900 dark:text-slate-300 dark:hover:bg-slate-700 dark:hover:text-slate-100"
                  >
                    <User className="h-4 w-4" strokeWidth={1.5} />
                    {tSidebar("myProfile")}
                  </Link>
                  <div className="my-0.5 border-t border-gray-100 dark:border-slate-700" />
                  <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-gray-600 transition-colors hover:bg-rose-50 hover:text-rose-600 dark:text-slate-300 dark:hover:bg-rose-900/30 dark:hover:text-rose-400"
                  >
                    <LogOut className="h-4 w-4" strokeWidth={1.5} />
                    {tCommon("signOut")}
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </header>
  )
}
