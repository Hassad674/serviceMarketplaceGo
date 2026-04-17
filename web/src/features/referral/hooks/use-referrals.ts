"use client"

import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query"

import {
  createReferral,
  getReferral,
  listAttributions,
  listCommissions,
  listIncomingReferrals,
  listMyReferrals,
  listNegotiations,
  respondToReferral,
  type ListReferralsFilter,
} from "../api/referral-api"
import type {
  CreateReferralInput,
  Referral,
  ReferralAttribution,
  ReferralCommission,
  ReferralListResponse,
  ReferralNegotiation,
  RespondReferralInput,
} from "../types"

// Query key factory — keep all referral keys under a single "referrals"
// namespace so the dashboard can invalidate the whole tree on any mutation.
export const referralKeys = {
  all: ["referrals"] as const,
  myList: (filter: ListReferralsFilter) =>
    [...referralKeys.all, "mine", filter] as const,
  incomingList: (filter: ListReferralsFilter) =>
    [...referralKeys.all, "incoming", filter] as const,
  detail: (id: string) => [...referralKeys.all, "detail", id] as const,
  negotiations: (id: string) =>
    [...referralKeys.all, "negotiations", id] as const,
  attributions: (id: string) =>
    [...referralKeys.all, "attributions", id] as const,
  commissions: (id: string) =>
    [...referralKeys.all, "commissions", id] as const,
}

// useMyReferrals fetches the apporteur dashboard list. Stale-while-revalidate
// with a 30-second freshness window so dashboard navigation feels instant
// while still picking up updates from other tabs / mobile.
export function useMyReferrals(filter: ListReferralsFilter = {}) {
  return useQuery<ReferralListResponse>({
    queryKey: referralKeys.myList(filter),
    queryFn: () => listMyReferrals(filter),
    staleTime: 30 * 1000,
  })
}

// useIncomingReferrals fetches intros where the current user is the
// provider or client party. Used by the incoming inbox shown on the
// dashboard alongside the apporteur's own intros.
export function useIncomingReferrals(filter: ListReferralsFilter = {}) {
  return useQuery<ReferralListResponse>({
    queryKey: referralKeys.incomingList(filter),
    queryFn: () => listIncomingReferrals(filter),
    staleTime: 30 * 1000,
  })
}

// useReferral fetches a single referral by id. Polls every 5 seconds while
// the row is in a pending state so the detail page reflects the other
// party's response without a manual refresh.
export function useReferral(id: string | undefined) {
  return useQuery<Referral>({
    queryKey: id ? referralKeys.detail(id) : ["referrals", "detail", "noop"],
    queryFn: () => getReferral(id!),
    enabled: Boolean(id),
    staleTime: 5 * 1000,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (!status) return false
      return status.startsWith("pending_") ? 5000 : false
    },
  })
}

export function useReferralNegotiations(id: string | undefined) {
  return useQuery<ReferralNegotiation[]>({
    queryKey: id
      ? referralKeys.negotiations(id)
      : ["referrals", "negotiations", "noop"],
    queryFn: () => listNegotiations(id!),
    enabled: Boolean(id),
    staleTime: 30 * 1000,
  })
}

// useReferralAttributions fetches the attributed proposals for a
// referral. Auto-refetched every 30 s while the referral is active so
// new milestone payments surface without a manual refresh.
export function useReferralAttributions(
  id: string | undefined,
  opts: { enabled?: boolean } = {},
) {
  const enabled = Boolean(id) && (opts.enabled ?? true)
  return useQuery<ReferralAttribution[]>({
    queryKey: id
      ? referralKeys.attributions(id)
      : ["referrals", "attributions", "noop"],
    queryFn: () => listAttributions(id!),
    enabled,
    staleTime: 30 * 1000,
  })
}

// useReferralCommissions fetches the per-milestone commission rows.
// Reserved for apporteur + provider; the backend returns 403 to the
// client, so components must not mount this hook for client viewers.
export function useReferralCommissions(
  id: string | undefined,
  opts: { enabled?: boolean } = {},
) {
  const enabled = Boolean(id) && (opts.enabled ?? true)
  return useQuery<ReferralCommission[]>({
    queryKey: id
      ? referralKeys.commissions(id)
      : ["referrals", "commissions", "noop"],
    queryFn: () => listCommissions(id!),
    enabled,
    staleTime: 30 * 1000,
  })
}

// useCreateReferral exposes the create mutation with cache invalidation
// on success. The dashboard list refreshes immediately.
export function useCreateReferral() {
  const queryClient = useQueryClient()
  return useMutation<Referral, Error, CreateReferralInput>({
    mutationFn: (input) => createReferral(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: referralKeys.all })
    },
  })
}

// useRespondToReferral handles every respond action (accept, reject,
// negotiate, cancel, terminate). The mutation invalidates the matching
// detail query and the dashboard lists so all surfaces stay in sync.
export function useRespondToReferral(id: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation<Referral, Error, RespondReferralInput>({
    mutationFn: (input) => {
      if (!id) throw new Error("referral id is required")
      return respondToReferral(id, input)
    },
    onSuccess: (data) => {
      if (id) {
        queryClient.setQueryData(referralKeys.detail(id), data)
        queryClient.invalidateQueries({ queryKey: referralKeys.negotiations(id) })
      }
      queryClient.invalidateQueries({ queryKey: referralKeys.all })
    },
  })
}
