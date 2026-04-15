import { SearchPage } from "@/features/provider/components/search-page"

// /referrers lists every organization of type `provider_personal` that
// has the `referrer_enabled` toggle on. Thin route wrapper; SearchPage
// composes the shared SearchPageLayout and fetches through the
// provider feature's existing TanStack hook.
export default function ReferrersDirectoryPage() {
  return <SearchPage type="referrer" />
}
