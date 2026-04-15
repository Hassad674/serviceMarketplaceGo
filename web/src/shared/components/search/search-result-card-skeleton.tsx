// Skeleton placeholder that matches the exact dimensions of
// SearchResultCard so the grid never shifts when results land. Uses
// the project's .animate-shimmer class (never a spinner, per the
// design system rules) — all three listing pages render a grid of
// these while the search response is in flight.

import { cn } from "@/shared/lib/utils"

interface SearchResultCardSkeletonProps {
  className?: string
}

export function SearchResultCardSkeleton({
  className,
}: SearchResultCardSkeletonProps) {
  return (
    <div
      className={cn(
        "flex h-full flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-sm",
        className,
      )}
      aria-hidden
    >
      <div className="relative aspect-[4/5] w-full overflow-hidden bg-muted">
        <div className="absolute inset-0 animate-shimmer bg-gradient-to-r from-muted via-muted/60 to-muted bg-[length:200%_100%]" />
      </div>
      <div className="flex flex-1 flex-col gap-3 p-4">
        <ShimmerBar width="70%" height="h-5" />
        <ShimmerBar width="50%" height="h-4" />
        <ShimmerBar width="40%" height="h-3" />
        <ShimmerBar width="60%" height="h-4" />
        <div className="mt-auto flex gap-1.5 pt-2">
          <ShimmerBar width="48px" height="h-5" rounded="rounded-full" />
          <ShimmerBar width="56px" height="h-5" rounded="rounded-full" />
          <ShimmerBar width="40px" height="h-5" rounded="rounded-full" />
        </div>
      </div>
    </div>
  )
}

interface ShimmerBarProps {
  width: string
  height: string
  rounded?: string
}

function ShimmerBar({ width, height, rounded = "rounded" }: ShimmerBarProps) {
  return (
    <div
      className={cn(
        "relative overflow-hidden bg-muted",
        height,
        rounded,
      )}
      style={{ width }}
    >
      <div className="absolute inset-0 animate-shimmer bg-gradient-to-r from-muted via-muted/60 to-muted bg-[length:200%_100%]" />
    </div>
  )
}
