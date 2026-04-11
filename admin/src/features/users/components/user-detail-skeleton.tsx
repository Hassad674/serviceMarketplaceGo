import { Skeleton } from "@/shared/components/ui/skeleton"

// Rendered while the useUser query is pending. Matches the final
// layout (back button + title + two cards) so the page does not
// jump when the real data arrives.

export function UserDetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-10 w-64" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Skeleton className="h-64 rounded-xl" />
        <Skeleton className="h-64 rounded-xl lg:col-span-2" />
      </div>
    </div>
  )
}
