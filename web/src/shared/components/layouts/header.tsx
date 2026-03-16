"use client"

import { useState, useRef, useEffect } from "react"
import { Menu, Bell, Search, LogOut, ChevronDown } from "lucide-react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

const roleLabels: Record<string, string> = {
  agency: "Agence",
  enterprise: "Entreprise",
  provider: "Prestataire",
}

type HeaderProps = {
  onMenuToggle?: () => void
}

export function Header({ onMenuToggle }: HeaderProps) {
  const { user, logout } = useAuth()
  const router = useRouter()
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

  return (
    <header className="flex h-16 shrink-0 items-center gap-4 border-b border-border bg-background px-4 sm:px-6">
      {/* Mobile menu toggle */}
      <button
        onClick={onMenuToggle}
        className="rounded-lg p-2 text-muted-foreground hover:bg-accent lg:hidden"
        aria-label="Ouvrir le menu"
      >
        <Menu className="h-5 w-5" />
      </button>

      {/* Search bar */}
      <div className="relative hidden flex-1 sm:block sm:max-w-md">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          type="text"
          placeholder="Rechercher..."
          className={cn(
            "h-9 w-full rounded-lg border border-input bg-background pl-9 pr-4 text-sm",
            "placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring",
          )}
        />
      </div>

      <div className="ml-auto flex items-center gap-3">
        {/* Notifications */}
        <button
          className="relative rounded-lg p-2 text-muted-foreground hover:bg-accent"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" />
        </button>

        {/* User dropdown */}
        {user && (
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setDropdownOpen((prev) => !prev)}
              className="flex items-center gap-2 rounded-lg p-1.5 transition-colors hover:bg-accent"
            >
              <div className="hidden text-right text-sm sm:block">
                <p className="font-medium text-foreground">{user.display_name}</p>
                <p className="text-xs text-muted-foreground">
                  {roleLabels[user.role] ?? user.role}
                </p>
              </div>
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground">
                {user.first_name.charAt(0)}
                {user.last_name.charAt(0)}
              </div>
              <ChevronDown className="hidden h-4 w-4 text-muted-foreground sm:block" />
            </button>

            {/* Dropdown menu */}
            {dropdownOpen && (
              <div className="absolute right-0 top-full z-50 mt-1 w-48 rounded-lg border border-border bg-popover py-1 shadow-lg">
                <div className="border-b border-border px-3 py-2 sm:hidden">
                  <p className="text-sm font-medium text-foreground">{user.display_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {roleLabels[user.role] ?? user.role}
                  </p>
                </div>
                <button
                  onClick={handleLogout}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent"
                >
                  <LogOut className="h-4 w-4" />
                  Se deconnecter
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </header>
  )
}
