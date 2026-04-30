import { Skeleton } from "./skeleton"

// RouteSkeleton is the Suspense fallback rendered between the
// AdminLayout's Outlet and a lazy route's resolved chunk
// (ADMIN-PERF-01). Mirrors the typical admin page layout — header,
// filters, table — so the layout doesn't shift when the route's
// real content lands.
export function RouteSkeleton() {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label="Loading admin section"
      className="space-y-6"
    >
      <span className="sr-only">Loading…</span>
      <Skeleton className="h-9 w-1/3" />
      <div className="flex gap-3">
        <Skeleton className="h-10 flex-1" />
        <Skeleton className="h-10 w-32" />
      </div>
      <div className="space-y-2">
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    </div>
  )
}
