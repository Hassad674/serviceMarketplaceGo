// Query-key factory for the invoicing feature. Every query key lives
// under the `["invoicing"]` prefix so a single `invalidateQueries`
// call can refresh the whole feature when needed (e.g. after a
// successful billing-profile save invalidates everything that
// depends on completeness).

export const invoicingQueryKey = {
  all: ["invoicing"] as const,
  profile: () => [...invoicingQueryKey.all, "billing-profile"] as const,
  invoices: (cursor: string | null) =>
    [...invoicingQueryKey.all, "invoices", cursor] as const,
  currentMonth: () => [...invoicingQueryKey.all, "current-month"] as const,
} as const
