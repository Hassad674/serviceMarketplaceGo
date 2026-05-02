import { Navigate, Outlet } from "react-router-dom"
import { useAuth } from "@/shared/hooks/use-auth"
import { useAdminWS } from "@/shared/hooks/use-admin-ws"
import { Sidebar } from "./sidebar"
import { Header } from "./header"
import { RouteSkeleton } from "@/shared/components/ui/route-skeleton"

export function AdminLayout() {
  const { isAuthenticated, isHydrating } = useAuth()

  // While the boot-time cookie probe is in flight (SEC-FINAL-07
  // memory-only token model — see auth-store.ts), avoid redirecting
  // to /login for users who actually have a valid session cookie.
  // The flash from "loading" to authenticated UI is shorter than the
  // /login round-trip would be.
  if (isHydrating) return <RouteSkeleton />

  if (!isAuthenticated) return <Navigate to="/login" replace />

  return <AuthenticatedLayout />
}

function AuthenticatedLayout() {
  useAdminWS()

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto bg-gray-50/50 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
