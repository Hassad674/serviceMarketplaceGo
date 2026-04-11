import { AcceptInvitationPage } from "@/features/team/components/accept-invitation-page"

// Thin route-level wrapper. Lives under (auth) so it inherits the
// public auth shell (no sidebar, no session guard). The token is a
// required URL segment — middleware.ts never marks this path as
// protected, so unauthenticated visitors land here directly from
// the email link.

export default async function Page({ params }: { params: Promise<{ token: string }> }) {
  const { token } = await params
  return <AcceptInvitationPage token={token} />
}
