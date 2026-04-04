import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { MessageSquare } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { useConversations } from "../hooks/use-conversations"
import { conversationsColumns } from "./conversations-columns"
import { ConversationsFilters } from "./conversations-filters"
import type { AdminConversation } from "../types"

export function ConversationsPage() {
  const navigate = useNavigate()
  const [cursor, setCursor] = useState("")
  const [cursors, setCursors] = useState<string[]>([])
  const [sort, setSort] = useState("recent")
  const [filter, setFilter] = useState("all")

  const { data, isLoading, error } = useConversations({ cursor, sort, filter })

  const conversations = data?.data ?? []
  const hasMore = data?.has_more ?? false
  const total = data?.total ?? 0

  function handleNextPage() {
    if (data?.next_cursor) {
      setCursors((prev) => [...prev, cursor])
      setCursor(data.next_cursor)
    }
  }

  function handlePreviousPage() {
    const prev = cursors[cursors.length - 1] ?? ""
    setCursors((c) => c.slice(0, -1))
    setCursor(prev)
  }

  function handleRowClick(conversation: AdminConversation) {
    navigate(`/conversations/${conversation.id}`)
  }

  function handleSortChange(newSort: string) {
    setSort(newSort)
    setCursor("")
    setCursors([])
  }

  function handleFilterChange(newFilter: string) {
    setFilter(newFilter)
    setCursor("")
    setCursors([])
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Conversations"
        description={total > 0 ? `${total} conversation${total > 1 ? "s" : ""} au total` : undefined}
      />

      <ConversationsFilters
        sort={sort}
        filter={filter}
        onSortChange={handleSortChange}
        onFilterChange={handleFilterChange}
      />

      {isLoading && <TableSkeleton rows={8} cols={5} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des conversations
        </div>
      )}

      {!isLoading && !error && conversations.length === 0 && (
        <EmptyState
          icon={MessageSquare}
          title="Aucune conversation"
          description="Aucune conversation n'a encore ete creee sur la plateforme."
        />
      )}

      {!isLoading && !error && conversations.length > 0 && (
        <>
          <DataTable
            columns={conversationsColumns}
            data={conversations}
            onRowClick={handleRowClick}
          />
          <DataTablePagination
            hasMore={hasMore}
            onNextPage={handleNextPage}
            onPreviousPage={handlePreviousPage}
            hasPreviousPage={cursors.length > 0}
            totalCount={total}
          />
        </>
      )}
    </div>
  )
}
