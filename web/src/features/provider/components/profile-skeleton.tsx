export function ProfileSkeleton() {
  return (
    <div className="space-y-6 animate-pulse" aria-label="Loading profile">
      {/* Header skeleton */}
      <div className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <div className="flex flex-col sm:flex-row items-start gap-6">
          <div className="w-24 h-24 rounded-full bg-muted shrink-0" />
          <div className="flex-1 space-y-3">
            <div className="h-6 w-48 bg-muted rounded" />
            <div className="h-4 w-64 bg-muted rounded" />
            <div className="h-4 w-32 bg-muted rounded" />
          </div>
          <div className="h-4 w-24 bg-muted rounded shrink-0" />
        </div>
      </div>

      {/* Video skeleton */}
      <div className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <div className="h-5 w-48 bg-muted rounded mb-4" />
        <div className="aspect-video bg-muted rounded-lg" />
      </div>

      {/* History skeleton */}
      <div className="bg-card border border-border rounded-xl p-6 shadow-sm">
        <div className="h-5 w-56 bg-muted rounded mb-4" />
        <div className="flex flex-col items-center py-10">
          <div className="w-14 h-14 rounded-full bg-muted mb-3" />
          <div className="h-4 w-40 bg-muted rounded mb-2" />
          <div className="h-4 w-64 bg-muted rounded" />
        </div>
      </div>
    </div>
  )
}
