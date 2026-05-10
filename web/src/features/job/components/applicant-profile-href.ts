// applicant-profile-href centralizes the persona-aware link the
// "View profile" CTA uses on the candidates list. The route prefix
// branches on the persisted applicant_kind so:
//   * agency        -> /agencies/<orgId>
//   * referrer      -> /referrers/<orgId>
//   * freelance     -> /freelancers/<orgId>
// Falls back to /freelancers/<orgId> for unknown kinds — same behaviour
// as the legacy hardcoded path so we never regress on a stale cache row.
//
// IMPORTANT: the org id is read from `profile.organization_id`, never
// from `application.applicant_id`. The DTO field `applicant_id` is the
// applicant's user id (audit trail), NOT the organization id — passing
// it to /freelancers/<id> 404s because the public profile route looks
// up `organizations.id`. Mobile already uses the org id; this helper
// brings web back in line.

import type { ApplicantKind } from "../types"

export function applicantProfileHref(
  applicantKind: ApplicantKind | string,
  organizationId: string,
): string {
  switch (applicantKind) {
    case "agency":
      return `/agencies/${organizationId}`
    case "referrer":
      return `/referrers/${organizationId}`
    case "freelance":
      return `/freelancers/${organizationId}`
    default:
      // Legacy/unknown kinds fall through to freelance — matches the
      // pre-fix behaviour so we never 404 harder than before.
      return `/freelancers/${organizationId}`
  }
}
