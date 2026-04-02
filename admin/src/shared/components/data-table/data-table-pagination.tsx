import { ChevronLeft, ChevronRight } from "lucide-react"
import { Button } from "@/shared/components/ui/button"

type DataTablePaginationProps = {
  hasMore: boolean
  onNextPage: () => void
  onPreviousPage?: () => void
  hasPreviousPage?: boolean
  totalCount?: number
}

export function DataTablePagination({
  hasMore,
  onNextPage,
  onPreviousPage,
  hasPreviousPage,
  totalCount,
}: DataTablePaginationProps) {
  return (
    <div className="flex items-center justify-between px-4 py-3">
      <div className="text-sm text-muted-foreground">
        {totalCount !== undefined && `${totalCount} résultat${totalCount > 1 ? "s" : ""}`}
      </div>
      <div className="flex items-center gap-2">
        {onPreviousPage && (
          <Button
            variant="outline"
            size="sm"
            onClick={onPreviousPage}
            disabled={!hasPreviousPage}
          >
            <ChevronLeft className="h-4 w-4" />
            Précédent
          </Button>
        )}
        <Button
          variant="outline"
          size="sm"
          onClick={onNextPage}
          disabled={!hasMore}
        >
          Suivant
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
