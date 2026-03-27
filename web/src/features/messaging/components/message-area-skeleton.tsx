import { cn } from "@/shared/lib/utils"

export function MessageAreaSkeleton() {
  return (
    <div className="flex-1 overflow-hidden px-5 py-4">
      <div className="mx-auto flex max-w-4xl flex-col gap-3">
        {[1, 2, 3, 4, 5].map((i) => (
          <div
            key={i}
            className={cn(
              "flex",
              i % 2 === 0 ? "justify-end" : "justify-start",
            )}
          >
            <div
              className={cn(
                "animate-pulse rounded-2xl px-4 py-2.5",
                i % 2 === 0 ? "bg-rose-200 dark:bg-rose-500/20" : "bg-gray-200 dark:bg-gray-700",
              )}
              style={{ width: `${40 + (i * 10) % 35}%`, height: "48px" }}
            />
          </div>
        ))}
      </div>
    </div>
  )
}
