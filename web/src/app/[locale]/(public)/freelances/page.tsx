import { getTranslations } from "next-intl/server"
import Link from "next/link"
import { Search, Sparkles } from "lucide-react"

// M-12 — Public freelances directory page (Soleil v2 visual port).
//
// The repo's actual freelance search engine lives at /freelancers
// (Typesense-backed SearchPage). This route is the placeholder
// editorial landing — same name as the brief, same role: hero +
// soft CTA pointing at the live search. We ship the editorial
// chrome (Fraunces hero with italic-corail accent, decorative
// search-bar mock, calm empty state) without inventing any new
// data hooks or schemas.
//
// FLAGGED: an actual freelance directory is already shipped at
// `/freelancers` (Typesense + RSC seed). This `/freelances` page is
// kept as a discovery / landing page only — no list, no filters.
// Active filters/results would require new hooks/schemas which the
// brief explicitly forbids.

export default async function FreelancesDirectoryPage() {
  const t = await getTranslations("freelancesSearch_m12")

  return (
    <div className="mx-auto w-full max-w-4xl space-y-10">
      {/* Editorial hero — eyebrow + Fraunces title with italic-corail
          accent + tabac italic subtitle. Anatomy matches W-22 / W-24. */}
      <header className="px-1">
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-[var(--primary)]">
          {t("eyebrow")}
        </p>
        <h1 className="mt-2 font-serif text-[34px] font-medium leading-[1.05] tracking-[-0.025em] text-[var(--foreground)] sm:text-[40px]">
          {t("titleLead")}{" "}
          <span className="italic text-[var(--primary)]">
            {t("titleAccent")}
          </span>
          .
        </h1>
        <p className="mt-3 max-w-2xl font-serif text-[15px] italic text-[var(--muted-foreground)]">
          {t("subtitle")}
        </p>
      </header>

      {/* Decorative search bar — full pill, ivoire bg, corail focus
          aura. Behaves as a link to the real search engine to keep
          the surface honest (no fake input). */}
      <Link
        href="/freelancers"
        className="group flex items-center gap-3 rounded-full border border-[var(--border)] bg-[var(--surface)] px-5 py-3.5 shadow-[var(--shadow-card)] transition-colors hover:border-[var(--primary)]"
      >
        <Search
          className="h-4 w-4 shrink-0 text-[var(--muted-foreground)] transition-colors group-hover:text-[var(--primary)]"
          strokeWidth={1.8}
        />
        <span className="flex-1 truncate font-serif text-[14px] italic text-[var(--muted-foreground)]">
          {t("searchPlaceholder")}
        </span>
        <span className="hidden shrink-0 items-center gap-1.5 rounded-full bg-[var(--primary-soft)] px-3 py-1 text-[12px] font-semibold text-[var(--primary-deep)] sm:inline-flex">
          {t("openSearch")}
        </span>
      </Link>

      {/* Filter pills mock — rounded-full, ivoire-soft. Pure visual
          chrome that points to the real filter sheet on /freelancers. */}
      <div className="flex flex-wrap gap-2">
        {[
          t("filters.expertise"),
          t("filters.location"),
          t("filters.rate"),
          t("filters.availability"),
        ].map((label) => (
          <Link
            key={label}
            href="/freelancers"
            className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border)] bg-[var(--background)] px-3.5 py-1.5 text-[12.5px] font-medium text-[var(--muted-foreground)] transition-colors hover:border-[var(--primary)] hover:text-[var(--primary)]"
          >
            {label}
          </Link>
        ))}
      </div>

      {/* Calm empty/CTA card — points the visitor to the real
          directory. We do NOT render fake card stubs (rule §3 of
          design/rules.md: skip when data isn't available). */}
      <div className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-10 text-center shadow-[var(--shadow-card)]">
        <span
          aria-hidden="true"
          className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
        >
          <Sparkles className="h-6 w-6" strokeWidth={1.6} />
        </span>
        <h2 className="mt-4 font-serif text-[22px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
          {t("emptyTitle")}
        </h2>
        <p className="mt-2 mx-auto max-w-md font-serif text-[14px] italic text-[var(--muted-foreground)]">
          {t("emptyDescription")}
        </p>
        <Link
          href="/freelancers"
          className="mt-5 inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-5 py-2.5 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)]"
        >
          {t("emptyCta")}
        </Link>
      </div>
    </div>
  )
}
