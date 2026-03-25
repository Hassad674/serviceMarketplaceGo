"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import {
  LayoutDashboard,
  MessageSquare,
  UserCircle,
  Search,
  Briefcase,
  Package,
  Receipt,
  Users,
  Building2,
  Building,
  Settings,
  Handshake,
  PlusCircle,
  FolderOpen,
  FileText,
  LogOut,
  ArrowRightLeft,
  X,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

type NavItem = {
  label: string
  href: string
  icon: React.ElementType
}

// ---------------------------------------------------------------------------
// Navigation configurations per role / mode
// ---------------------------------------------------------------------------

const agencyNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/agency", icon: LayoutDashboard },
  { label: "Messagerie", href: "/dashboard/agency/messaging", icon: MessageSquare },
  { label: "Mon profil", href: "/dashboard/agency/profile", icon: UserCircle },
  { label: "Trouver des clients", href: "#", icon: Search },
  { label: "Gérer mes missions", href: "/dashboard/agency/missions", icon: Briefcase },
  { label: "Offres clés en main", href: "#", icon: Package },
  { label: "Gestion financière", href: "/dashboard/agency/invoices", icon: Receipt },
  { label: "Mon équipe", href: "/dashboard/agency/team", icon: Users },
  { label: "Mon entreprise", href: "#", icon: Building2 },
  { label: "Mon compte", href: "#", icon: Settings },
]

const providerNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/provider", icon: LayoutDashboard },
  { label: "Messagerie", href: "/dashboard/provider/messaging", icon: MessageSquare },
  { label: "Mon profil", href: "/dashboard/provider/profile", icon: UserCircle },
  { label: "Trouver des clients", href: "#", icon: Search },
  { label: "Gérer mes missions", href: "/dashboard/provider/missions", icon: Briefcase },
  { label: "Offres clés en main", href: "#", icon: Package },
  { label: "Gestion financière", href: "/dashboard/provider/invoices", icon: Receipt },
  { label: "Mon entreprise", href: "#", icon: Building2 },
  { label: "Mon compte", href: "#", icon: Settings },
]

const referrerNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/referrer", icon: LayoutDashboard },
  { label: "Mes apports d'affaires", href: "#", icon: Handshake },
  { label: "Créer un apport d'affaire", href: "#", icon: PlusCircle },
  { label: "Messagerie", href: "#", icon: MessageSquare },
  { label: "Entreprises", href: "#", icon: Building },
  { label: "Profil Apporteur", href: "#", icon: UserCircle },
  { label: "Gestion financière", href: "#", icon: Receipt },
  { label: "Chercher un freelance", href: "#", icon: Search },
  { label: "Mon compte", href: "#", icon: Settings },
]

const enterpriseNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/enterprise", icon: LayoutDashboard },
  { label: "Messagerie", href: "/dashboard/enterprise/messaging", icon: MessageSquare },
  { label: "Chercher un prestataire", href: "#", icon: Search },
  { label: "Chercher un apporteur", href: "#", icon: Handshake },
  { label: "Offres clés en main", href: "#", icon: Package },
  { label: "Gérer mes projets", href: "/dashboard/enterprise/projects", icon: FolderOpen },
  { label: "Mes projets déposés", href: "#", icon: FileText },
  { label: "Mes factures", href: "/dashboard/enterprise/invoices", icon: Receipt },
  { label: "Mon équipe", href: "#", icon: Users },
  { label: "Mon compte", href: "#", icon: Settings },
]

const ROLE_TITLES: Record<string, string> = {
  agency: "AgenceB2B",
  enterprise: "EnterpriseB2B",
  provider: "ProviderB2B",
  referrer: "ApporteurB2B",
}

// ---------------------------------------------------------------------------
// Helper: select the right nav config
// ---------------------------------------------------------------------------

function getNavConfig(role: string, isReferrerMode: boolean): NavItem[] {
  if (role === "provider" && isReferrerMode) return referrerNav
  if (role === "provider") return providerNav
  if (role === "agency") return agencyNav
  if (role === "enterprise") return enterpriseNav
  return enterpriseNav
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

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
  const sidebarTitle = isReferrerMode ? ROLE_TITLES.referrer : (ROLE_TITLES[role] ?? "Marketplace")

  function handleLogout() {
    logout()
    router.push("/login")
  }

  return (
    <>
      {/* Mobile overlay */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/40 lg:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-sidebar-border bg-sidebar lg:static lg:z-auto",
          "transition-transform duration-200 lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full",
        )}
      >
        {/* Header: logo + close */}
        <div className="flex h-16 items-center justify-between border-b border-sidebar-border px-6">
          <Link
            href="/"
            className="text-lg font-semibold text-sidebar-foreground"
          >
            {sidebarTitle}
          </Link>
          <button
            onClick={onClose}
            className="rounded-md p-1 text-sidebar-foreground hover:bg-sidebar-accent/50 lg:hidden"
            aria-label="Fermer le menu"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Role switch button (provider only) */}
        {role === "provider" && (
          <div className="px-3 pt-4">
            <SwitchButton isReferrerMode={isReferrerMode} />
          </div>
        )}

        {/* Navigation items */}
        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
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
        <div className="border-t border-sidebar-border p-3">
          <button
            onClick={handleLogout}
            className={cn(
              "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium",
              "text-sidebar-foreground transition-colors hover:bg-sidebar-accent/50",
            )}
          >
            <LogOut className="h-5 w-5 shrink-0" />
            Se déconnecter
          </button>
        </div>
      </aside>
    </>
  )
}

// ---------------------------------------------------------------------------
// Sub-components (kept in the same file, internal only)
// ---------------------------------------------------------------------------

function SwitchButton({ isReferrerMode }: { isReferrerMode: boolean }) {
  if (isReferrerMode) {
    return (
      <Link
        href="/dashboard/provider"
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2.5",
          "text-sm font-medium transition-colors",
          "bg-emerald-500/10 text-emerald-600 hover:bg-emerald-500/20",
        )}
      >
        <ArrowRightLeft className="h-4 w-4" />
        Dashboard Freelance
      </Link>
    )
  }

  return (
    <Link
      href="/dashboard/referrer"
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2.5",
        "text-sm font-medium transition-colors",
        "bg-primary/10 text-primary hover:bg-primary/20",
      )}
    >
      <ArrowRightLeft className="h-4 w-4" />
      Apporteur d&apos;affaire
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
        "relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors",
        isActive
          ? "bg-primary/10 font-medium text-primary"
          : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground",
      )}
    >
      {/* Active dot indicator */}
      {isActive && (
        <span className="absolute left-0 h-6 w-1 rounded-r-full bg-primary" />
      )}
      <item.icon className="h-5 w-5 shrink-0" />
      {item.label}
    </Link>
  )
}
