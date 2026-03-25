"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import {
  LayoutDashboard,
  UserCircle,
  LogOut,
  ArrowRightLeft,
  X,
  Sparkles,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

type NavItem = {
  label: string
  href: string
  icon: React.ElementType
}

const agencyNav: NavItem[] = [
  { label: "Dashboard", href: "/dashboard/agency", icon: LayoutDashboard },
  { label: "My Profile", href: "/dashboard/agency/profile", icon: UserCircle },
]

const providerNav: NavItem[] = [
  { label: "Dashboard", href: "/dashboard/provider", icon: LayoutDashboard },
  { label: "My Profile", href: "/dashboard/provider/profile", icon: UserCircle },
]

const referrerNav: NavItem[] = [
  { label: "Dashboard", href: "/dashboard/referrer", icon: LayoutDashboard },
  { label: "Referrer Profile", href: "/dashboard/provider/referral", icon: UserCircle },
]

const enterpriseNav: NavItem[] = [
  { label: "Dashboard", href: "/dashboard/enterprise", icon: LayoutDashboard },
]

const ROLE_LABELS: Record<string, string> = {
  agency: "Agency",
  enterprise: "Enterprise",
  provider: "Provider",
  referrer: "Referrer",
}

const ROLE_COLORS: Record<string, string> = {
  agency: "bg-blue-50 text-blue-700",
  enterprise: "bg-purple-50 text-purple-700",
  provider: "bg-rose-50 text-rose-700",
  referrer: "bg-amber-50 text-amber-700",
}

function getNavConfig(role: string, isReferrerMode: boolean): NavItem[] {
  if (role === "provider" && isReferrerMode) return referrerNav
  if (role === "provider") return providerNav
  if (role === "agency") return agencyNav
  if (role === "enterprise") return enterpriseNav
  return enterpriseNav
}

type SidebarProps = {
  open?: boolean
  onClose?: () => void
}

export function Sidebar({ open, onClose }: SidebarProps) {
  const pathname = usePathname()
  const router = useRouter()
  const { user, logout } = useAuth()

  const role = user?.role ?? "enterprise"
  const isReferrerMode = pathname.startsWith("/dashboard/referrer")
  const items = getNavConfig(role, isReferrerMode)
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
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col",
          "bg-white/80 backdrop-blur-xl border-r border-gray-100/50",
          "lg:static lg:z-auto",
          "transition-transform duration-300 ease-out lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full",
        )}
      >
        {/* Logo */}
        <div className="flex h-16 items-center justify-between px-6">
          <Link href="/" className="flex items-center gap-2">
            <span className="bg-gradient-to-r from-rose-500 to-purple-600 bg-clip-text text-xl font-bold tracking-tight text-transparent">
              Atelier
            </span>
          </Link>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 lg:hidden"
            aria-label="Close menu"
          >
            <X className="h-4 w-4" strokeWidth={1.5} />
          </button>
        </div>

        {/* User info */}
        <div className="mx-4 mb-2 rounded-xl bg-gray-50/80 p-3">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-rose-500 to-purple-600 text-sm font-semibold text-white">
              {initials}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-gray-900">
                {user?.display_name ?? "User"}
              </p>
              <span
                className={cn(
                  "mt-0.5 inline-block rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                  ROLE_COLORS[displayRole] ?? "bg-gray-100 text-gray-600",
                )}
              >
                {ROLE_LABELS[displayRole] ?? displayRole}
              </span>
            </div>
          </div>
        </div>

        {/* Role switch (provider only) */}
        {role === "provider" && (
          <div className="px-4 pb-2">
            <ReferrerSwitch isReferrerMode={isReferrerMode} />
          </div>
        )}

        {/* Navigation */}
        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-2">
          {items.map((item) => (
            <NavLink
              key={item.label}
              item={item}
              pathname={pathname}
              onClick={onClose}
            />
          ))}
        </nav>

        {/* Logout */}
        <div className="border-t border-gray-100/80 p-3">
          <button
            onClick={handleLogout}
            className={cn(
              "flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm",
              "text-gray-500 transition-all duration-200 hover:bg-gray-50 hover:text-gray-700",
            )}
          >
            <LogOut className="h-5 w-5" strokeWidth={1.5} />
            Sign Out
          </button>
        </div>
      </aside>
    </>
  )
}

function ReferrerSwitch({ isReferrerMode }: { isReferrerMode: boolean }) {
  if (isReferrerMode) {
    return (
      <Link
        href="/dashboard/provider"
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-xl px-3 py-2.5",
          "text-sm font-medium transition-all duration-200",
          "bg-emerald-50 text-emerald-700 hover:bg-emerald-100",
        )}
      >
        <ArrowRightLeft className="h-4 w-4" strokeWidth={1.5} />
        Freelance Mode
      </Link>
    )
  }

  return (
    <Link
      href="/dashboard/referrer"
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-xl px-3 py-2.5",
        "text-sm font-medium text-white transition-all duration-200",
        "gradient-referrer hover:opacity-90 hover:shadow-md",
      )}
    >
      <Sparkles className="h-4 w-4" strokeWidth={1.5} />
      Business Referrer
    </Link>
  )
}

function NavLink({
  item,
  pathname,
  onClick,
}: {
  item: NavItem
  pathname: string
  onClick?: () => void
}) {
  const isActive =
    item.href !== "#" &&
    (pathname === item.href || pathname.startsWith(item.href + "/"))

  return (
    <Link
      href={item.href}
      onClick={onClick}
      className={cn(
        "relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition-all duration-200",
        isActive
          ? "bg-rose-50 font-medium text-rose-600"
          : "text-gray-500 hover:bg-gray-50 hover:text-gray-900",
      )}
    >
      {/* Active indicator pill */}
      {isActive && (
        <span className="absolute left-0 top-1/2 h-6 w-[3px] -translate-y-1/2 rounded-full bg-rose-500" />
      )}
      <item.icon className="h-5 w-5 shrink-0" strokeWidth={1.5} />
      {item.label}
    </Link>
  )
}
