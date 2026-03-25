import { DashboardShell } from "@/shared/components/layouts/dashboard-shell"

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return <DashboardShell>{children}</DashboardShell>
}
