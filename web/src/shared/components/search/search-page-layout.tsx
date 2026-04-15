"use client"

import { useMemo, useState } from "react"
import { Filter, Search, Users, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  toSearchDocument,
  type RawSearchDocumentLike,
} from "@/shared/lib/search/search-document-adapter"
import type {
  SearchDocument,
  SearchDocumentPersona,
} from "@/shared/lib/search/search-document"
import { SearchResultCard } from "./search-result-card"
import { SearchResultCardSkeleton } from "./search-result-card-skeleton"
import { SearchFilterSidebar } from "./search-filter-sidebar"
import {
  EMPTY_SEARCH_FILTERS,
  type SearchFilters,
} from "./search-filters"

// SearchPageLayout is the composition root shared by the three public
// listing pages (freelancers / agencies / referrers). It assembles:
//
//   - top bar with search input, sort dropdown, and filter toggle,
//   - left-side filter sidebar (sticky on desktop, drawer on mobile),
//   - results grid (3 cols desktop, 2 cols tablet, 1 col mobile),
//   - loading / empty / error states,
//   - load-more button.
//
// State is owned by the parent through a handful of render callbacks
// so the page can be server-rendered (title, description) with a
// client-side slot for the actual data fetching hook.

type SortKey = "relevance" | "rating" | "priceAsc" | "priceDesc" | "recent"

export type SearchLayoutStatus = "idle" | "loading" | "error"

export interface SearchPageLayoutProps {
  title: string
  subtitle?: string
  persona: SearchDocumentPersona
  documents: RawSearchDocumentLike[]
  status: SearchLayoutStatus
  hasMore: boolean
  isLoadingMore: boolean
  onLoadMore: () => void
  onRetry?: () => void
  query: string
  onQueryChange: (next: string) => void
}

export function SearchPageLayout(props: SearchPageLayoutProps) {
  const t = useTranslations("search")
  const [filters, setFilters] = useState<SearchFilters>(EMPTY_SEARCH_FILTERS)
  const [sort, setSort] = useState<SortKey>("relevance")
  const [drawerOpen, setDrawerOpen] = useState(false)

  const mappedDocuments: SearchDocument[] = useMemo(
    () => props.documents.map((raw) => toSearchDocument(raw, props.persona)),
    [props.documents, props.persona],
  )

  const handleApply = () => {
    // UI-only filter wiring today — the backend does not accept
    // these parameters yet, so clicking Apply just closes the mobile
    // drawer. When Typesense lands this becomes a query invalidation.
    // eslint-disable-next-line no-console
    console.debug("search filters applied", { filters, sort })
    setDrawerOpen(false)
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title={props.title} subtitle={props.subtitle} />
      <TopBar
        query={props.query}
        onQueryChange={props.onQueryChange}
        sort={sort}
        onSortChange={setSort}
        onOpenDrawer={() => setDrawerOpen(true)}
      />
      <div className="flex flex-col gap-6 lg:grid lg:grid-cols-[280px_1fr] lg:gap-8">
        <div className="hidden lg:block">
          <SearchFilterSidebar
            filters={filters}
            onChange={setFilters}
            onApply={handleApply}
            resultsCount={mappedDocuments.length}
          />
        </div>
        <div className="flex flex-col gap-6">
          <ResultsSection
            status={props.status}
            documents={mappedDocuments}
            hasMore={props.hasMore}
            isLoadingMore={props.isLoadingMore}
            onLoadMore={props.onLoadMore}
            onRetry={props.onRetry}
            onResetFilters={() => setFilters(EMPTY_SEARCH_FILTERS)}
            loadMoreLabel={t("loadMore")}
            loadingLabel={t("loading")}
          />
        </div>
      </div>
      <FilterDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        <SearchFilterSidebar
          filters={filters}
          onChange={setFilters}
          onApply={handleApply}
          resultsCount={mappedDocuments.length}
          className="border-0 shadow-none"
        />
      </FilterDrawer>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sections
// ---------------------------------------------------------------------------

function PageHeader({
  title,
  subtitle,
}: {
  title: string
  subtitle?: string
}) {
  return (
    <header className="flex flex-col gap-1">
      <h1 className="text-2xl font-bold tracking-tight text-foreground md:text-3xl">
        {title}
      </h1>
      {subtitle ? (
        <p className="text-sm text-muted-foreground">{subtitle}</p>
      ) : null}
    </header>
  )
}

function TopBar({
  query,
  onQueryChange,
  sort,
  onSortChange,
  onOpenDrawer,
}: {
  query: string
  onQueryChange: (next: string) => void
  sort: SortKey
  onSortChange: (next: SortKey) => void
  onOpenDrawer: () => void
}) {
  const t = useTranslations("search")
  const tSort = useTranslations("search.sort")
  return (
    <div className="flex flex-col gap-2 md:flex-row md:items-center">
      <label className="relative flex-1">
        <span className="sr-only">{t("searchPlaceholder")}</span>
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <input
          type="search"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder={t("searchPlaceholder")}
          className="h-11 w-full rounded-xl border border-border bg-card pl-9 pr-4 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
        />
      </label>
      <div className="flex items-center gap-2">
        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          <span className="hidden md:inline">{tSort("label")}</span>
          <select
            value={sort}
            onChange={(e) => onSortChange(e.target.value as SortKey)}
            className="h-11 rounded-xl border border-border bg-card px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
          >
            <option value="relevance">{tSort("relevance")}</option>
            <option value="rating">{tSort("rating")}</option>
            <option value="priceAsc">{tSort("priceAsc")}</option>
            <option value="priceDesc">{tSort("priceDesc")}</option>
            <option value="recent">{tSort("recent")}</option>
          </select>
        </label>
        <button
          type="button"
          onClick={onOpenDrawer}
          aria-label={t("showFilters")}
          className="inline-flex h-11 items-center gap-2 rounded-xl border border-border bg-card px-3 text-sm font-medium text-foreground shadow-xs hover:border-rose-200 lg:hidden"
        >
          <Filter className="h-4 w-4" aria-hidden />
          <span>{t("showFilters")}</span>
        </button>
      </div>
    </div>
  )
}

interface ResultsSectionProps {
  status: SearchLayoutStatus
  documents: SearchDocument[]
  hasMore: boolean
  isLoadingMore: boolean
  onLoadMore: () => void
  onRetry?: () => void
  onResetFilters: () => void
  loadMoreLabel: string
  loadingLabel: string
}

function ResultsSection(props: ResultsSectionProps) {
  const t = useTranslations("search")
  if (props.status === "loading") {
    return <LoadingGrid />
  }
  if (props.status === "error") {
    return (
      <ErrorState
        message={t("errorLoading")}
        retryLabel={t("loadMore")}
        onRetry={props.onRetry}
      />
    )
  }
  if (props.documents.length === 0) {
    return <EmptyState onReset={props.onResetFilters} />
  }
  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {props.documents.map((doc) => (
          <SearchResultCard key={doc.id} document={doc} />
        ))}
      </div>
      {props.hasMore ? (
        <div className="flex justify-center pt-2">
          <button
            type="button"
            onClick={props.onLoadMore}
            disabled={props.isLoadingMore}
            className="rounded-lg bg-rose-500 px-6 py-2.5 text-sm font-medium text-white transition-all duration-200 ease-out hover:bg-rose-600 hover:shadow-glow active:scale-[0.98] disabled:opacity-50"
          >
            {props.isLoadingMore ? props.loadingLabel : props.loadMoreLabel}
          </button>
        </div>
      ) : null}
    </>
  )
}

function LoadingGrid() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 9 }).map((_, i) => (
        <SearchResultCardSkeleton key={i} />
      ))}
    </div>
  )
}

function EmptyState({ onReset }: { onReset: () => void }) {
  const t = useTranslations("search.empty")
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border bg-card p-12 text-center">
      <Users className="h-10 w-10 text-muted-foreground" aria-hidden />
      <p className="text-base font-semibold text-foreground">{t("title")}</p>
      <p className="text-sm text-muted-foreground">{t("description")}</p>
      <button
        type="button"
        onClick={onReset}
        className="mt-2 rounded-lg border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
      >
        {t("cta")}
      </button>
    </div>
  )
}

function ErrorState({
  message,
  retryLabel,
  onRetry,
}: {
  message: string
  retryLabel: string
  onRetry?: () => void
}) {
  return (
    <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-center dark:border-red-500/30 dark:bg-red-500/10">
      <p className="text-sm text-red-700 dark:text-red-300">{message}</p>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="mt-3 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
        >
          {retryLabel}
        </button>
      ) : null}
    </div>
  )
}

function FilterDrawer({
  open,
  onClose,
  children,
}: {
  open: boolean
  onClose: () => void
  children: React.ReactNode
}) {
  const t = useTranslations("search")
  return (
    <div
      className={cn(
        "fixed inset-0 z-50 flex items-end bg-black/60 transition-opacity duration-200 lg:hidden",
        open ? "opacity-100" : "pointer-events-none opacity-0",
      )}
      role="dialog"
      aria-modal="true"
      aria-label={t("filters.title")}
      onClick={onClose}
    >
      <div
        className={cn(
          "relative flex max-h-[85vh] w-full flex-col overflow-y-auto rounded-t-2xl bg-card shadow-xl transition-transform duration-300 ease-out",
          open ? "translate-y-0" : "translate-y-full",
        )}
        onClick={(e) => e.stopPropagation()}
      >
        <button
          type="button"
          onClick={onClose}
          aria-label={t("hideFilters")}
          className="absolute right-4 top-4 inline-flex h-9 w-9 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors hover:text-foreground"
        >
          <X className="h-4 w-4" aria-hidden />
        </button>
        {children}
      </div>
    </div>
  )
}
