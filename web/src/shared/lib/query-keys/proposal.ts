/**
 * Shared proposal query keys.
 *
 * `proposalQueryKey` is used cross-feature: the `messaging` feature's
 * WebSocket layer invalidates proposal queries when a system message
 * announces a proposal status change. Lifted out of
 * `features/proposal/hooks/use-proposals` so messaging does not have
 * to import from the proposal feature directly.
 *
 * The proposal feature also imports from here.
 */

export function projectsQueryKey(uid: string | undefined) {
  return ["user", uid, "projects"] as const
}

export function proposalQueryKey(uid: string | undefined) {
  return ["user", uid, "proposal"] as const
}
