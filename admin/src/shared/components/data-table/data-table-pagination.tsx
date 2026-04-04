import { ChevronLeft, ChevronRight } from "lucide-react"
import { cn } from "@/shared/lib/utils"

type DataTablePaginationProps = {
  currentPage: number
  totalPages: number
  totalCount: number
  onPageChange: (page: number) => void
}

export function DataTablePagination({
  currentPage,
  totalPages,
  totalCount,
  onPageChange,
}: DataTablePaginationProps) {
  const pages = buildPageNumbers(currentPage, totalPages)

  return (
    <div className="flex items-center justify-between px-4 py-3">
      <div className="text-sm text-muted-foreground">
        {totalCount} r{"\u00E9"}sultat{totalCount > 1 ? "s" : ""}
      </div>
      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={() => onPageChange(currentPage - 1)}
          disabled={currentPage <= 1}
          className={cn(
            "inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm font-medium transition-all duration-200 ease-out",
            currentPage <= 1
              ? "cursor-not-allowed text-muted-foreground/50"
              : "text-muted-foreground hover:bg-muted",
          )}
        >
          <ChevronLeft className="h-4 w-4" />
          Pr{"\u00E9"}c{"\u00E9"}dent
        </button>

        {pages.map((entry, idx) =>
          entry === "..." ? (
            <span
              key={`ellipsis-${idx}`}
              className="px-2 py-1.5 text-sm text-muted-foreground"
            >
              ...
            </span>
          ) : (
            <button
              key={entry}
              type="button"
              onClick={() => onPageChange(entry as number)}
              className={cn(
                "min-w-[36px] rounded-lg px-2.5 py-1.5 text-sm font-medium transition-all duration-200 ease-out",
                entry === currentPage
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-muted",
              )}
            >
              {entry}
            </button>
          ),
        )}

        <button
          type="button"
          onClick={() => onPageChange(currentPage + 1)}
          disabled={currentPage >= totalPages}
          className={cn(
            "inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm font-medium transition-all duration-200 ease-out",
            currentPage >= totalPages
              ? "cursor-not-allowed text-muted-foreground/50"
              : "text-muted-foreground hover:bg-muted",
          )}
        >
          Suivant
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}

function buildPageNumbers(current: number, total: number): (number | "...")[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }

  const pages: (number | "...")[] = [1]

  if (current > 3) {
    pages.push("...")
  }

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  if (current < total - 2) {
    pages.push("...")
  }

  pages.push(total)

  return pages
}
