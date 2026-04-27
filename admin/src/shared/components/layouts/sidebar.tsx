import { NavLink } from "react-router-dom"
import { LayoutDashboard, Users, MessageSquare, Briefcase, Star, Image, Shield, Scale, FileText, LogOut } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { useAuth } from "@/shared/hooks/use-auth"
import { useModerationCount } from "@/shared/hooks/use-moderation-count"

type NavItem = {
  to: string
  label: string
  icon: typeof LayoutDashboard
  badge?: number
}

export function Sidebar() {
  const { logout } = useAuth()
  const { data: countData } = useModerationCount()

  const pendingCount = countData?.count ?? 0

  const navigation: NavItem[] = [
    { to: "/", label: "Tableau de bord", icon: LayoutDashboard },
    { to: "/moderation", label: "Mod\u00E9ration", icon: Shield, badge: pendingCount },
    { to: "/users", label: "Utilisateurs", icon: Users },
    { to: "/conversations", label: "Conversations", icon: MessageSquare },
    { to: "/jobs", label: "Offres", icon: Briefcase },
    { to: "/reviews", label: "Avis", icon: Star },
    { to: "/media", label: "M\u00E9dias", icon: Image },
    { to: "/disputes", label: "Litiges", icon: Scale },
    { to: "/invoices", label: "Factures", icon: FileText },
  ]

  return (
    <aside className="flex w-[280px] flex-col border-r border-gray-100/50 bg-white/80 backdrop-blur-xl">
      <div className="flex h-16 items-center border-b border-gray-100/50 px-6">
        <h1 className="text-lg font-bold bg-gradient-to-r from-rose-500 to-rose-600 bg-clip-text text-transparent">
          Marketplace Admin
        </h1>
      </div>

      <nav className="flex-1 space-y-1 px-3 py-4">
        {navigation.map(({ to, label, icon: Icon, badge }) => (
          <NavLink
            key={to}
            to={to}
            end={to === "/"}
            className={({ isActive }) =>
              cn(
                "relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200 ease-out",
                isActive
                  ? "bg-rose-50 text-rose-600"
                  : "text-muted-foreground hover:bg-gray-50 hover:text-foreground",
              )
            }
          >
            {({ isActive }) => (
              <>
                {isActive && (
                  <span className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-r-full bg-rose-500" />
                )}
                <Icon className="h-4 w-4" />
                <span className="flex-1">{label}</span>
                {badge != null && badge > 0 && (
                  <span className="inline-flex min-w-[20px] items-center justify-center rounded-full bg-destructive px-1.5 py-0.5 text-[10px] font-semibold text-white">
                    {badge > 99 ? "99+" : badge}
                  </span>
                )}
              </>
            )}
          </NavLink>
        ))}
      </nav>

      <div className="border-t border-gray-100/50 p-3">
        <button
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:bg-gray-50 hover:text-foreground"
        >
          <LogOut className="h-4 w-4" />
          {`D\u00E9connexion`}
        </button>
      </div>
    </aside>
  )
}
