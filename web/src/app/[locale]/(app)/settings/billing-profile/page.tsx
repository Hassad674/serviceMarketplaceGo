import type { Metadata } from "next"
import { BillingProfilePageClient } from "./page-client"

export const metadata: Metadata = {
  title: "Profil de facturation",
}

export default function BillingProfilePage() {
  return <BillingProfilePageClient />
}
