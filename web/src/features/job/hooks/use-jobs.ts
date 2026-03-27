"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createJob, listMyJobs, closeJob } from "../api/job-api"
import type { CreateJobData } from "../api/job-api"

export const JOBS_QUERY_KEY = ["jobs"]

export function useCreateJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateJobData) => createJob(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: JOBS_QUERY_KEY })
    },
  })
}

export function useMyJobs(cursor?: string) {
  return useQuery({
    queryKey: [...JOBS_QUERY_KEY, "mine", cursor],
    queryFn: () => listMyJobs(cursor),
    staleTime: 30 * 1000,
  })
}

export function useCloseJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => closeJob(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: JOBS_QUERY_KEY })
    },
  })
}
