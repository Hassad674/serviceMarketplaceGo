import { redirect } from "@i18n/navigation"

// /cgu — permanent redirect to the full, legally-reviewed CGU document
// served at /legal/cgu. The standalone /cgu page used to be an empty
// placeholder shell; the authoritative content lives at /legal/cgu
// (locale-aware: /en/legal/terms). Redirecting avoids a duplicate,
// content-less URL and keeps a single source of truth.
export default async function CguPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  redirect({ href: "/legal/cgu", locale })
}
