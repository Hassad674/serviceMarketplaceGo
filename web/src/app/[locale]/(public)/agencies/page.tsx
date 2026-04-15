import { SearchPage } from "@/features/provider/components/search-page"

// /agencies lists every organization of type `agency`. Thin route
// wrapper: SearchPage composes the shared SearchPageLayout and owns
// the data fetching via the provider feature's TanStack hook.
export default function AgenciesDirectoryPage() {
  return <SearchPage type="agency" />
}
