"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  listOpenJobs,
  applyToJob,
  withdrawApplication,
  listJobApplications,
  listMyApplications,
  contactApplicant,
  hasApplied,
} from "../api/job-application-api"
import type { OpenJobListFilters } from "../types"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

function openJobsKey(filters?: OpenJobListFilters, cursor?: string) {
  return ["jobs", "open", filters, cursor] as const
}

function myApplicationsKey(uid: string | undefined, cursor?: string) {
  return ["user", uid, "applications", cursor] as const
}

function jobApplicationsKey(jobId: string, cursor?: string) {
  return ["jobs", jobId, "applications", cursor] as const
}

function hasAppliedKey(uid: string | undefined, jobId: string) {
  return ["user", uid, "hasApplied", jobId] as const
}

export function useOpenJobs(filters?: OpenJobListFilters, cursor?: string) {
  return useQuery({
    queryKey: openJobsKey(filters, cursor),
    queryFn: () => listOpenJobs(filters, cursor),
    staleTime: 30 * 1000,
  })
}

export function useApplyToJob() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ jobId, message, videoUrl }: { jobId: string; message: string; videoUrl?: string }) =>
      applyToJob(jobId, { message, video_url: videoUrl }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: hasAppliedKey(uid, variables.jobId) })
      queryClient.invalidateQueries({ queryKey: ["jobs", "open"] })
      queryClient.invalidateQueries({ queryKey: ["user", uid, "applications"] })
      queryClient.invalidateQueries({ queryKey: ["credits"] })
    },
  })
}

export function useWithdrawApplication() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (applicationId: string) => withdrawApplication(applicationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["user", uid, "applications"] })
      queryClient.invalidateQueries({ queryKey: ["jobs", "open"] })
    },
  })
}

export function useJobApplications(jobId: string, cursor?: string) {
  return useQuery({
    queryKey: jobApplicationsKey(jobId, cursor),
    queryFn: () => listJobApplications(jobId, cursor),
    staleTime: 30 * 1000,
  })
}

export function useMyApplications(cursor?: string) {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: myApplicationsKey(uid, cursor),
    queryFn: () => listMyApplications(cursor),
    staleTime: 30 * 1000,
  })
}

export function useContactApplicant() {
  return useMutation({
    mutationFn: ({ jobId, applicantId }: { jobId: string; applicantId: string }) =>
      contactApplicant(jobId, applicantId),
  })
}

export function useHasApplied(jobId: string) {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: hasAppliedKey(uid, jobId),
    queryFn: () => hasApplied(jobId),
    staleTime: 60 * 1000,
  })
}
