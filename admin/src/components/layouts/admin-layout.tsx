import { Navigate, Outlet, NavLink } from "react-router-dom"
import { useAuth } from "@/hooks/use-auth.tsx"
import { cn } from "@/lib/utils.ts"
import {
  LayoutDashboard,
  Users,
  Briefcase,
  Building2,
  ClipboardList,
  FolderKanban,
  FileText,
  ShieldAlert,
  Settings,
  LogOut,
} from "lucide-react"

const navigation = [
  { to: "/", label: "Tableau de bord", icon: LayoutDashboard },
  { to: "/users", label: "Utilisateurs", icon: Users },
  { to: "/providers", label: "Prestataires", icon: Briefcase },
  { to: "/enterprises", label: "Entreprises", icon: Building2 },
  { to: "/missions", label: "Missions", icon: ClipboardList },
  { to: "/projects", label: "Projets", icon: FolderKanban },
  { to: "/invoices", label: "Factures", icon: FileText },
  { to: "/reports", label: "Signalements", icon: ShieldAlert },
  { to: "/settings", label: "Paramètres", icon: Settings },
]

export default function AdminLayout() {
  const { isAuthenticated, logout } = useAuth()

  if (!isAuthenticated) return <Navigate to="/login" replace />

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="flex w-64 flex-col border-r border-border bg-sidebar">
        <div className="flex h-16 items-center border-b border-border px-6">
          <h1 className="text-lg font-bold text-primary">Marketplace Admin</h1>
        </div>

        <nav className="flex-1 space-y-1 px-3 py-4">
          {navigation.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === "/"}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground",
                )
              }
            >
              <Icon className="h-4 w-4" />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="border-t border-border p-3">
          <button
            onClick={logout}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <LogOut className="h-4 w-4" />
            Déconnexion
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto bg-muted/30 p-8">
        <Outlet />
      </main>
    </div>
  )
}
