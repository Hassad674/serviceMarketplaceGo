import { Suspense } from "react"
import { AccountSettingsPage } from "@/features/account/components/account-settings-page"

/**
 * Soleil v2 skeleton — mirrors the live layout so the loading state
 * doesn't shift content (LCP/CLS budget). Editorial title + 240px
 * tabs column + flexible content column.
 */
function AccountPageSkeleton() {
  return (
    <div className="mx-auto max-w-[1100px]">
      <div className="mb-7 h-9 w-60 animate-pulse rounded-lg bg-border" />
      <div className="flex flex-col gap-7 lg:grid lg:grid-cols-[240px_1fr] lg:items-start">
        <div className="h-44 w-full animate-pulse rounded-2xl border border-border bg-card" />
        <div className="h-96 w-full animate-pulse rounded-2xl border border-border bg-card" />
      </div>
    </div>
  )
}

export default function AccountPage() {
  return (
    <Suspense fallback={<AccountPageSkeleton />}>
      <AccountSettingsPage />
    </Suspense>
  )
}
