import { ClientProfilePage } from "@/features/client-profile/components/client-profile-page"

// Private editable client profile. Thin route wrapper — all state,
// permissions and data fetching live in the feature component.
export default function ClientProfileRoutePage() {
  return <ClientProfilePage />
}
