"use client"

import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query"

import {
  createReferral,
  listAttributions,
  listCommissions,
  listIncomingReferrals,
  listMyReferrals,
  listNegotiations,
  type ListReferralsFilter,
} from "../api/referral-api"
import type {
  CreateReferralInput,
  Referral,
  ReferralAttribution,
  ReferralCommission,
  ReferralListResponse,
  ReferralNegotiation,
} from "../types"

// `useReferral` and `useRespondToReferral` (P9 — shared with messaging
// for the inline system message) live in
// `@/shared/hooks/referral/use-referral`. Re-exported here so existing
// intra-feature imports keep working.
export {
  useReferral,
  useRespondToReferral,
} from "@/shared/hooks/referral/use-referral"

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

// `useReferral` lives in `@/shared/hooks/referral/use-referral` (P9 —
// re-exported at the top of this file).

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

// `useRespondToReferral` lives in `@/shared/hooks/referral/use-referral`
// (P9 — re-exported at the top of this file).
