export function TeamPageSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="h-20 rounded-xl bg-gray-100 dark:bg-slate-800" />
      <div className="h-10 rounded-xl bg-gray-100 dark:bg-slate-800" />
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-16 rounded-xl bg-gray-100 dark:bg-slate-800" />
        ))}
      </div>
    </div>
  )
}
