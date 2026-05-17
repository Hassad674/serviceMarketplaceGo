import { redirect } from "@i18n/navigation"

// /cgv — permanent redirect to the full, legally-reviewed CGV document
// served at /legal/cgv. The standalone /cgv page used to be an empty
// placeholder shell; the authoritative content lives at /legal/cgv
// (locale-aware: /en/legal/sales-terms). Redirecting avoids a
// duplicate, content-less URL and keeps a single source of truth.
export default async function CgvPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  redirect({ href: "/legal/cgv", locale })
}
