// Query key factory for the subscription feature. Every query key
// lives under the `['subscription']` prefix so consumers can
// invalidate the whole feature in a single call if needed.

export const subscriptionQueryKey = {
  all: ["subscription"] as const,
  me: () => [...subscriptionQueryKey.all, "me"] as const,
  stats: () => [...subscriptionQueryKey.all, "stats"] as const,
} as const
