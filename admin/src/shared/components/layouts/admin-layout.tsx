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
        <main className="flex-1 overflow-y-auto bg-gray-50/50 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
