// Soleil v2 — Skeleton placeholder for /team while session + members
// + invitations are still loading. Mirrors the editorial header,
// section eyebrow and member-card layout so the page does not shift
// when data arrives.

export function TeamPageSkeleton() {
  return (
    <div className="space-y-7 animate-pulse">
      <div className="space-y-2 px-1">
        <div className="h-3 w-24 rounded-full bg-[var(--border)]" />
        <div className="h-9 w-3/4 rounded-xl bg-[var(--border)]" />
        <div className="h-3 w-2/3 rounded-full bg-[var(--border)]" />
      </div>
      <div className="space-y-2">
        <div className="h-3 w-20 rounded-full bg-[var(--border)]" />
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="h-[68px] rounded-2xl border border-[var(--border)] bg-[var(--surface)]"
          />
        ))}
      </div>
    </div>
  )
}
