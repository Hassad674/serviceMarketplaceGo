"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createJob, updateJob, listMyJobs, closeJob, reopenJob, deleteJob, markApplicationsViewed, getCredits } from "../api/job-api"
import type { CreateJobData } from "../api/job-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function jobsQueryKey(uid: string | undefined) {
  return ["user", uid, "jobs"] as const
}

/** @deprecated Use jobsQueryKey(uid) instead */
export const JOBS_QUERY_KEY = ["jobs"]

export function useCreateJob() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (data: CreateJobData) => createJob(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jobsQueryKey(uid) })
    },
  })
}

export function useUpdateJob(jobId: string) {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (data: CreateJobData) => updateJob(jobId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jobsQueryKey(uid) })
      queryClient.invalidateQueries({ queryKey: ["jobs", jobId] })
    },
  })
}

export function useMyJobs(cursor?: string) {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: [...jobsQueryKey(uid), "mine", cursor],
    queryFn: () => listMyJobs(cursor),
    staleTime: 30 * 1000,
  })
}

export function useCloseJob() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => closeJob(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jobsQueryKey(uid) })
    },
  })
}

export function useReopenJob() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => reopenJob(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jobsQueryKey(uid) })
    },
  })
}

export function useDeleteJob() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (id: string) => deleteJob(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jobsQueryKey(uid) })
    },
  })
}

export function useMarkApplicationsViewed() {
  return useMutation({
    mutationFn: (id: string) => markApplicationsViewed(id),
  })
}

export function useCredits() {
  return useQuery({
    queryKey: ["credits"],
    queryFn: () => getCredits(),
    staleTime: 30 * 1000,
  })
}
