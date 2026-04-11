import { TeamPage } from "@/features/team/components/team-page"

// Thin route-level wrapper. Real composition + data fetching live in
// the feature component so this file stays minimal (the CLAUDE.md
// convention: app/ is routing only, features own the logic).

export default function Page() {
  return <TeamPage />
}
