import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listAdminJobs,
  getAdminJob,
  deleteAdminJob,
  listAdminJobApplications,
  deleteAdminJobApplication,
} from "../api/jobs-api"
import type { JobFilters, ApplicationFilters } from "../types"

export function jobsQueryKey(filters: JobFilters) {
  return ["admin", "jobs", filters] as const
}

export function useAdminJobs(filters: JobFilters) {
  return useQuery({
    queryKey: jobsQueryKey(filters),
    queryFn: () => listAdminJobs(filters),
    staleTime: 30 * 1000,
  })
}

export function jobQueryKey(id: string) {
  return ["admin", "jobs", id] as const
}

export function useAdminJob(id: string) {
  return useQuery({
    queryKey: jobQueryKey(id),
    queryFn: () => getAdminJob(id),
    enabled: !!id,
  })
}

export function useDeleteJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAdminJob(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "jobs"] })
    },
  })
}

export function applicationsQueryKey(filters: ApplicationFilters) {
  return ["admin", "job-applications", filters] as const
}

export function useAdminJobApplications(filters: ApplicationFilters) {
  return useQuery({
    queryKey: applicationsQueryKey(filters),
    queryFn: () => listAdminJobApplications(filters),
    staleTime: 30 * 1000,
  })
}

export function useDeleteJobApplication() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAdminJobApplication(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "job-applications"] })
      queryClient.invalidateQueries({ queryKey: ["admin", "jobs"] })
    },
  })
}
