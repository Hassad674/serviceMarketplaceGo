import { getTranslations } from "next-intl/server"
import { Link } from "@i18n/navigation"

// not-found.tsx for the locale segment (PERF-W-03). Triggered when
// the router can't match a route OR a Server Component calls
// `notFound()`. Renders a localised 404 with a CTA back to the
// public listings — the most useful destination for an unknown URL
// on a marketplace SEO surface.
export default async function LocaleNotFound() {
  const t = await getTranslations("boundary")
  return (
    <div className="mx-auto flex min-h-[60vh] max-w-md flex-col items-center justify-center gap-4 p-6 text-center">
      <p
        aria-hidden
        className="text-6xl font-extrabold tracking-tight text-primary/80"
      >
        404
      </p>
      <h1 className="text-2xl font-bold">{t("notFoundTitle")}</h1>
      <p className="text-sm text-muted-foreground">
        {t("notFoundDescription")}
      </p>
      <Link
        href="/agencies"
        className="mt-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:opacity-90"
      >
        {t("notFoundCta")}
      </Link>
    </div>
  )
}
