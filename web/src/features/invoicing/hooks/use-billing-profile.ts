// Billing-profile hooks live in `@/shared/hooks/billing-profile/use-billing-profile`
// (P9 — wallet renders the completion gate). Re-exported here for back-compat.
export {
  useBillingProfile,
  useUpdateBillingProfile,
  useSyncBillingProfile,
  useValidateVAT,
} from "@/shared/hooks/billing-profile/use-billing-profile"
