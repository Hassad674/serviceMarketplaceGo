// Shared invoicing query keys (P9 — billing-profile and current-month
// hooks live in `shared/`, so the keys live here too to stay in sync
// with the legacy `invoicingQueryKey` factory in
// `features/invoicing/hooks/keys`).

export const sharedInvoicingQueryKey = {
  all: ["invoicing"] as const,
  profile: () => [...sharedInvoicingQueryKey.all, "billing-profile"] as const,
  currentMonth: () =>
    [...sharedInvoicingQueryKey.all, "current-month"] as const,
} as const
