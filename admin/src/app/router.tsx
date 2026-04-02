import { BrowserRouter, Routes, Route } from "react-router-dom"
import { AdminLayout } from "@/shared/components/layouts/admin-layout"
import { LoginPage } from "@/features/auth/components/login-form"
import { DashboardPage } from "@/features/dashboard/components/dashboard-page"
import { UsersPage } from "@/features/users/components/users-page"
import { UserDetailPage } from "@/features/users/components/user-detail-page"

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        <Route element={<AdminLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/users/:id" element={<UserDetailPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
