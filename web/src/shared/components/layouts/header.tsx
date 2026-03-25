"use client"

import { useState, useRef, useEffect } from "react"
import { Menu, Bell, Search, LogOut, User, ChevronDown } from "lucide-react"
import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

const ROLE_LABELS: Record<string, string> = {
  agency: "Agency",
  enterprise: "Enterprise",
  provider: "Provider",
}

const ROLE_COLORS: Record<string, string> = {
  agency: "bg-blue-50 text-blue-700",
  enterprise: "bg-purple-50 text-purple-700",
  provider: "bg-rose-50 text-rose-700",
}

type HeaderProps = {
  onMenuToggle?: () => void
}

export function Header({ onMenuToggle }: HeaderProps) {
  const { user, logout } = useAuth()
  const router = useRouter()
  const pathname = usePathname()
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [])

  function handleLogout() {
    setDropdownOpen(false)
    logout()
    router.push("/login")
  }

  const initials = user
    ? `${user.first_name.charAt(0)}${user.last_name.charAt(0)}`
    : "?"

  const profileHref = user?.role === "agency"
    ? "/dashboard/agency/profile"
    : user?.role === "provider"
      ? "/dashboard/provider/profile"
      : "/dashboard/enterprise"

  return (
    <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center gap-4 border-b border-gray-100/50 bg-white/80 px-4 backdrop-blur-xl sm:px-6">
      {/* Mobile menu */}
      <button
        onClick={onMenuToggle}
        className="rounded-xl p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 lg:hidden"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5" strokeWidth={1.5} />
      </button>

      {/* Search */}
      <div className="relative hidden flex-1 sm:block sm:max-w-sm">
        <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" strokeWidth={1.5} />
        <input
          type="text"
          placeholder="Search..."
          className={cn(
            "h-10 w-full rounded-full bg-gray-100/80 pl-10 pr-4 text-sm text-gray-900",
            "placeholder:text-gray-400 transition-all duration-200",
            "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
          )}
        />
      </div>

      <div className="ml-auto flex items-center gap-2">
        {/* Notifications */}
        <button
          className="relative rounded-xl p-2.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" strokeWidth={1.5} />
        </button>

        {/* User dropdown */}
        {user && (
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setDropdownOpen((prev) => !prev)}
              className="flex items-center gap-2.5 rounded-xl p-1.5 transition-all duration-200 hover:bg-gray-50"
            >
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-xs font-semibold text-white">
                {initials}
              </div>
              <ChevronDown
                className={cn(
                  "hidden h-4 w-4 text-gray-400 transition-transform duration-200 sm:block",
                  dropdownOpen && "rotate-180",
                )}
                strokeWidth={1.5}
              />
            </button>

            {/* Dropdown */}
            {dropdownOpen && (
              <div className="animate-scale-in absolute right-0 top-full z-50 mt-2 w-64 overflow-hidden rounded-2xl border border-gray-100/80 bg-white/90 shadow-lg backdrop-blur-xl">
                <div className="border-b border-gray-100 p-4">
                  <p className="text-sm font-semibold text-gray-900">{user.display_name}</p>
                  <p className="mt-0.5 text-xs text-gray-500">{user.email}</p>
                  <span
                    className={cn(
                      "mt-2 inline-block rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                      ROLE_COLORS[user.role] ?? "bg-gray-100 text-gray-600",
                    )}
                  >
                    {ROLE_LABELS[user.role] ?? user.role}
                  </span>
                </div>
                <div className="p-1.5">
                  <Link
                    href={profileHref}
                    onClick={() => setDropdownOpen(false)}
                    className="flex items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm text-gray-600 transition-colors hover:bg-gray-50 hover:text-gray-900"
                  >
                    <User className="h-4 w-4" strokeWidth={1.5} />
                    My Profile
                  </Link>
                  <div className="my-1 border-t border-gray-100" />
                  <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm text-gray-600 transition-colors hover:bg-rose-50 hover:text-rose-600"
                  >
                    <LogOut className="h-4 w-4" strokeWidth={1.5} />
                    Sign Out
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
