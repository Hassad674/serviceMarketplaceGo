import { cn } from "@/shared/lib/utils"

// SkeletonBlock is the shared shimmer primitive used by route-level
// loading.tsx files (PERF-W-03 + QUAL-W-01). It renders a single
// muted div with the design-system `.animate-shimmer` overlay so the
// page-level skeletons stay consistent across (auth), (public),
// (app) and slow per-route loaders.
//
// We deliberately keep the API to two props (className + as) so the
// JSX nesting in the consuming loading boundaries stays under 3
// levels even when a skeleton is composed of multiple blocks.

interface SkeletonBlockProps {
  className?: string
}

export function SkeletonBlock({ className }: SkeletonBlockProps) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-md bg-muted",
        className,
      )}
      aria-hidden
    >
      <div className="absolute inset-0 animate-shimmer bg-gradient-to-r from-muted via-muted/60 to-muted bg-[length:200%_100%]" />
    </div>
  )
}
