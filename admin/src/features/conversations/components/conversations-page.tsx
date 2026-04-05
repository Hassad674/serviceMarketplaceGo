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

function resolveFilter(reportedConversations: boolean, reportedMessages: boolean): string {
  if (reportedConversations && reportedMessages) return "reported"
  if (reportedConversations) return "reported_conversations"
  if (reportedMessages) return "reported_messages"
  return "all"
}

export function ConversationsPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [sort, setSort] = useState("recent")
  const [reportedConversations, setReportedConversations] = useState(false)
  const [reportedMessages, setReportedMessages] = useState(false)

  const filter = resolveFilter(reportedConversations, reportedMessages)
  const { data, isLoading, error } = useConversations({ page, sort, filter })

  const conversations = data?.data ?? []
  const total = data?.total ?? 0
  const totalPages = data?.total_pages ?? 0

  function handleRowClick(conversation: AdminConversation) {
    navigate(`/conversations/${conversation.id}`)
  }

  function handleSortChange(newSort: string) {
    setSort(newSort)
    setPage(1)
  }

  function resetPagination() {
    setPage(1)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Conversations"
        description={total > 0 ? `${total} conversation${total > 1 ? "s" : ""} au total` : undefined}
      />

      <ConversationsFilters
        sort={sort}
        reportedConversations={reportedConversations}
        reportedMessages={reportedMessages}
        onSortChange={handleSortChange}
        onReportedConversationsChange={(v) => { setReportedConversations(v); resetPagination() }}
        onReportedMessagesChange={(v) => { setReportedMessages(v); resetPagination() }}
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
            currentPage={page}
            totalPages={totalPages}
            totalCount={total}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  )
}
