"use client"

import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import {
  LayoutDashboard,
  MessageSquare,
  User,
  Briefcase,
  Users,
  Receipt,
  Building2,
  Settings,
  Search,
  FolderKanban,
  Package,
  Handshake,
  X,
  LogOut,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

type NavItem = { label: string; href: string; icon: React.ElementType }

const agencyNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/agency", icon: LayoutDashboard },
  { label: "Messagerie", href: "/dashboard/agency/messaging", icon: MessageSquare },
  { label: "Mon profil", href: "/dashboard/agency/profile", icon: User },
  { label: "Missions", href: "/dashboard/agency/missions", icon: Briefcase },
  { label: "Equipe", href: "/dashboard/agency/team", icon: Users },
  { label: "Facturation", href: "/dashboard/agency/invoices", icon: Receipt },
  { label: "Mon entreprise", href: "/dashboard/agency/company", icon: Building2 },
  { label: "Mon compte", href: "/dashboard/agency/settings", icon: Settings },
]

const enterpriseNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/enterprise", icon: LayoutDashboard },
  { label: "Messagerie", href: "/dashboard/enterprise/messaging", icon: MessageSquare },
  { label: "Mes projets", href: "/dashboard/enterprise/projects", icon: FolderKanban },
  { label: "Rechercher", href: "/dashboard/enterprise/search", icon: Search },
  { label: "Facturation", href: "/dashboard/enterprise/invoices", icon: Receipt },
  { label: "Mon entreprise", href: "/dashboard/enterprise/company", icon: Building2 },
  { label: "Mon compte", href: "/dashboard/enterprise/settings", icon: Settings },
]

const providerNav: NavItem[] = [
  { label: "Tableau de bord", href: "/dashboard/provider", icon: LayoutDashboard },
  { label: "Messagerie", href: "/dashboard/provider/messaging", icon: MessageSquare },
  { label: "Mon profil", href: "/dashboard/provider/profile", icon: User },
  { label: "Trouver des clients", href: "/dashboard/provider/search", icon: Search },
  { label: "Missions", href: "/dashboard/provider/missions", icon: Briefcase },
  { label: "Offres cles en main", href: "/dashboard/provider/offers", icon: Package },
  { label: "Facturation", href: "/dashboard/provider/invoices", icon: Receipt },
  { label: "Mon entreprise", href: "/dashboard/provider/company", icon: Building2 },
  { label: "Mon compte", href: "/dashboard/provider/settings", icon: Settings },
]

const navByRole: Record<string, NavItem[]> = {
  agency: agencyNav,
  enterprise: enterpriseNav,
  provider: providerNav,
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
  const items = navByRole[role] ?? enterpriseNav

  function handleLogout() {
    logout()
    router.push("/login")
  }

  return (
    <>
      {/* Mobile overlay */}
      {open && (
        <div className="fixed inset-0 z-40 bg-black/40 lg:hidden" onClick={onClose} />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-sidebar-border bg-sidebar lg:static lg:z-auto",
          "transition-transform duration-200 lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full",
        )}
      >
        {/* Logo */}
        <div className="flex h-16 items-center justify-between border-b border-sidebar-border px-6">
          <Link href="/" className="text-lg font-semibold text-sidebar-foreground">
            Marketplace
          </Link>
          <button
            onClick={onClose}
            className="lg:hidden text-sidebar-foreground"
            aria-label="Fermer le menu"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Referrer toggle for providers */}
        {role === "provider" && (
          <Link
            href="/dashboard/provider"
            className={cn(
              "mx-4 mt-4 flex items-center justify-center gap-2 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors",
              user?.referrer_enabled
                ? "bg-primary/10 text-primary"
                : "border border-dashed border-sidebar-border text-sidebar-foreground hover:bg-sidebar-accent/50",
            )}
          >
            <Handshake className="h-4 w-4" />
            {user?.referrer_enabled
              ? "Mode Apporteur d'affaires"
              : "Devenir apporteur d'affaires"}
          </Link>
        )}

        {/* Navigation */}
        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {items.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(item.href + "/")
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-sidebar-foreground hover:bg-sidebar-accent/50",
                )}
              >
                <item.icon className="h-4 w-4 shrink-0" />
                {item.label}
              </Link>
            )
          })}
        </nav>

        {/* Logout button */}
        <div className="border-t border-sidebar-border p-3">
          <button
            onClick={handleLogout}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-accent/50"
          >
            <LogOut className="h-4 w-4 shrink-0" />
            Se deconnecter
          </button>
        </div>
      </aside>
    </>
  )
}
