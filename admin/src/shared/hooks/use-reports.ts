import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listConversationReports,
  listUserReports,
  listJobReports,
  resolveReport,
} from "@/shared/api/reports-api"

export function useConversationReports(conversationId: string) {
  return useQuery({
    queryKey: ["admin", "conversations", conversationId, "reports"],
    queryFn: () => listConversationReports(conversationId),
    enabled: !!conversationId,
    staleTime: 30 * 1000,
  })
}

export function useUserReports(userId: string) {
  return useQuery({
    queryKey: ["admin", "users", userId, "reports"],
    queryFn: () => listUserReports(userId),
    enabled: !!userId,
    staleTime: 30 * 1000,
  })
}

export function useJobReports(jobId: string) {
  return useQuery({
    queryKey: ["admin", "jobs", jobId, "reports"],
    queryFn: () => listJobReports(jobId),
    enabled: !!jobId,
    staleTime: 30 * 1000,
  })
}

export function useResolveReport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (params: { reportId: string; status: "resolved" | "dismissed"; adminNote: string }) =>
      resolveReport(params.reportId, { status: params.status, admin_note: params.adminNote }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "conversations"] })
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] })
      queryClient.invalidateQueries({ queryKey: ["admin", "jobs"] })
    },
  })
}
