import { SearchPage } from "@/features/provider/components/search-page"

// /freelancers lists every organization of type `provider_personal`.
// The page is a thin route wrapper: data fetching + composition both
// live in the provider feature's SearchPage component, which renders
// the shared SearchPageLayout internally.
export default function FreelancersDirectoryPage() {
  return <SearchPage type="freelancer" />
}
