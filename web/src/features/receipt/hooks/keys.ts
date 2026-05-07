/**
 * Query-key factory for the receipt feature. Every query key lives
 * under the `["receipt"]` prefix so a single `invalidateQueries`
 * call can refresh the whole feature when needed (for example after
 * a payment-side mutation that emits a new receipt).
 */
export const receiptQueryKey = {
  all: ["receipt"] as const,
  list: (cursor: string | null) =>
    [...receiptQueryKey.all, "list", cursor] as const,
  detail: (id: string) => [...receiptQueryKey.all, "detail", id] as const,
} as const
