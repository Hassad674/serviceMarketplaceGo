// Thin shim re-exporting the shared expertise catalog. Historically
// this constant lived inside the provider feature; the split-profile
// refactor moved it to shared/lib/profile so the new freelance and
// referrer features can consume it without crossing feature
// boundaries. The shim is preserved so agency pages still compile.
export {
  EXPERTISE_DOMAIN_KEYS,
  isExpertiseDomainKey,
  getMaxExpertiseForOrgType,
  orgTypeSupportsExpertise,
  type ExpertiseDomainKey,
} from "@/shared/lib/profile/expertise"
