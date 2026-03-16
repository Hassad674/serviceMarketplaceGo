import { BrowserRouter, Routes, Route } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { AuthProvider } from "@/hooks/use-auth.tsx"
import AdminLayout from "@/components/layouts/admin-layout.tsx"
import LoginPage from "@/pages/Login.tsx"
import DashboardPage from "@/pages/Dashboard.tsx"
import UsersPage from "@/pages/Users.tsx"
import ProvidersPage from "@/pages/Providers.tsx"
import EnterprisesPage from "@/pages/Enterprises.tsx"
import MissionsPage from "@/pages/Missions.tsx"
import ProjectsPage from "@/pages/Projects.tsx"
import InvoicesPage from "@/pages/Invoices.tsx"
import ReportsPage from "@/pages/Reports.tsx"
import SettingsPage from "@/pages/Settings.tsx"

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            {/* Public */}
            <Route path="/login" element={<LoginPage />} />

            {/* Protected — wrapped in AdminLayout (handles auth guard) */}
            <Route element={<AdminLayout />}>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/users" element={<UsersPage />} />
              <Route path="/providers" element={<ProvidersPage />} />
              <Route path="/enterprises" element={<EnterprisesPage />} />
              <Route path="/missions" element={<MissionsPage />} />
              <Route path="/projects" element={<ProjectsPage />} />
              <Route path="/invoices" element={<InvoicesPage />} />
              <Route path="/reports" element={<ReportsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Route>
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
