import { Navigate, Outlet } from "react-router-dom"
import { useAuth } from "@/shared/hooks/use-auth"
import { Sidebar } from "./sidebar"
import { Header } from "./header"

export function AdminLayout() {
  const { isAuthenticated } = useAuth()

  if (!isAuthenticated) return <Navigate to="/login" replace />

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto bg-muted/30 p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
