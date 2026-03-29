import { Suspense } from "react"
import { AccountSettingsPage } from "@/features/account/components/account-settings-page"

function AccountPageSkeleton() {
  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-6 h-8 w-48 animate-pulse rounded-lg bg-slate-200 dark:bg-slate-700" />
      <div className="flex flex-col gap-6 lg:flex-row">
        <div className="h-40 w-full animate-pulse rounded-xl bg-slate-200 lg:w-56 dark:bg-slate-700" />
        <div className="h-96 flex-1 animate-pulse rounded-xl bg-slate-200 dark:bg-slate-700" />
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
