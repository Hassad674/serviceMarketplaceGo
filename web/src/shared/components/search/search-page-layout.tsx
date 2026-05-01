"use client"

import { useEffect, useMemo, useRef, useState } from "react"
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

import { Button } from "@/shared/components/ui/button"
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

export type SortKey = "relevance" | "rating" | "priceAsc" | "priceDesc" | "recent"

export type SearchLayoutStatus = "idle" | "loading" | "error"

export interface SearchPageLayoutProps {
  title: string
  subtitle?: string
  persona: SearchDocumentPersona
  /**
   * documents is the raw legacy SQL response (PublicProfileSummary
   * shape). The layout runs the adapter to convert each entry into
   * the frozen SearchDocument shape before rendering. Mutually
   * exclusive with `preMappedDocuments`.
   */
  documents?: RawSearchDocumentLike[]
  /**
   * preMappedDocuments is used by the Typesense path which already
   * adapts the raw Typesense document into a SearchDocument before
   * passing it down. When both are provided, pre-mapped wins.
   */
  preMappedDocuments?: SearchDocument[]
  status: SearchLayoutStatus
  hasMore: boolean
  isLoadingMore: boolean
  onLoadMore: () => void
  onRetry?: () => void
  query: string
  onQueryChange: (next: string) => void
  /**
   * Controlled filter state. When provided, the layout forwards
   * changes through `onFiltersChange` instead of keeping local
   * state. Required for the Typesense path where the parent has to
   * pipe filters back into the query hook.
   */
  filters?: SearchFilters
  onFiltersChange?: (next: SearchFilters) => void
  /**
   * Controlled sort state. Same rationale as `filters`: the
   * Typesense parent maps the SortKey to a Typesense `sort_by`
   * string before passing it to useSearch.
   */
  sort?: SortKey
  onSortChange?: (next: SortKey) => void
  /**
   * Total match count across all pages (Typesense `found`). Used
   * by the sidebar header when larger than the currently-rendered
   * document slice. Falls back to the local document count.
   */
  totalFound?: number
  /**
   * Optional click handler — invoked when the user clicks on a
   * result card. Phase 3 passes a click-tracking beacon here; SQL
   * path leaves it undefined so the cards behave as pre-3 links.
   */
  onSelect?: (docID: string, position: number) => void
}

export function SearchPageLayout(props: SearchPageLayoutProps) {
  const t = useTranslations("search")
  const [internalFilters, setInternalFilters] = useState<SearchFilters>(
    EMPTY_SEARCH_FILTERS,
  )
  const [internalSort, setInternalSort] = useState<SortKey>("relevance")
  const [drawerOpen, setDrawerOpen] = useState(false)

  const filters = props.filters ?? internalFilters
  const setFilters = (next: SearchFilters) => {
    if (props.onFiltersChange) {
      props.onFiltersChange(next)
    } else {
      setInternalFilters(next)
    }
  }
  const sort = props.sort ?? internalSort
  const setSort = (next: SortKey) => {
    if (props.onSortChange) {
      props.onSortChange(next)
    } else {
      setInternalSort(next)
    }
  }

  const mappedDocuments: SearchDocument[] = useMemo(() => {
    if (props.preMappedDocuments) {
      return props.preMappedDocuments
    }
    return (props.documents ?? []).map((raw) =>
      toSearchDocument(raw, props.persona),
    )
  }, [props.documents, props.preMappedDocuments, props.persona])

  const handleApply = () => {
    // UI-only filter wiring today — the backend does not accept
    // these parameters yet, so clicking Apply just closes the mobile
    // drawer. When Typesense lands this becomes a query invalidation.
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
            resultsCount={props.totalFound ?? mappedDocuments.length}
            persona={props.persona}
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
            onSelect={props.onSelect}
          />
        </div>
      </div>
      <FilterDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        <SearchFilterSidebar
          filters={filters}
          onChange={setFilters}
          onApply={handleApply}
          resultsCount={props.totalFound ?? mappedDocuments.length}
          persona={props.persona}
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
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onOpenDrawer}
          aria-label={t("showFilters")}
          className="inline-flex h-11 items-center gap-2 rounded-xl border border-border bg-card px-3 text-sm font-medium text-foreground shadow-xs hover:border-rose-200 lg:hidden"
        >
          <Filter className="h-4 w-4" aria-hidden />
          <span>{t("showFilters")}</span>
        </Button>
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
  onSelect?: (docID: string, position: number) => void
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
        {props.documents.map((doc, index) => (
          <SearchResultCard
            key={doc.id}
            document={doc}
            onSelect={
              props.onSelect
                ? () => props.onSelect?.(doc.id, index)
                : undefined
            }
          />
        ))}
      </div>
      {props.hasMore ? (
        <InfiniteScrollFooter
          onLoadMore={props.onLoadMore}
          isLoadingMore={props.isLoadingMore}
          loadMoreLabel={props.loadMoreLabel}
          loadingLabel={props.loadingLabel}
        />
      ) : null}
    </>
  )
}

/**
 * InfiniteScrollFooter renders a sentinel `<div>` that auto-triggers
 * `onLoadMore` when scrolled into view, and keeps a visible "Load
 * more" button underneath as an accessible + no-JS fallback.
 *
 * The IntersectionObserver margin is 400px so we start fetching the
 * next page before the user reaches the grid bottom — this gives
 * the network request time to complete and the scroll feels smooth.
 */
function InfiniteScrollFooter({
  onLoadMore,
  isLoadingMore,
  loadMoreLabel,
  loadingLabel,
}: {
  onLoadMore: () => void
  isLoadingMore: boolean
  loadMoreLabel: string
  loadingLabel: string
}) {
  const sentinelRef = useRef<HTMLDivElement | null>(null)
  const loadMoreRef = useRef(onLoadMore)
  const isLoadingRef = useRef(isLoadingMore)

  useEffect(() => {
    loadMoreRef.current = onLoadMore
    isLoadingRef.current = isLoadingMore
  }, [onLoadMore, isLoadingMore])

  useEffect(() => {
    const node = sentinelRef.current
    if (!node || typeof IntersectionObserver === "undefined") return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting) && !isLoadingRef.current) {
          loadMoreRef.current()
        }
      },
      { rootMargin: "400px 0px" },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [])

  return (
    <>
      <div ref={sentinelRef} aria-hidden className="h-px w-full" />
      <div className="flex justify-center pt-2">
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onLoadMore}
          disabled={isLoadingMore}
          className="rounded-lg bg-rose-500 px-6 py-2.5 text-sm font-medium text-white transition-all duration-200 ease-out hover:bg-rose-600 hover:shadow-glow active:scale-[0.98] disabled:opacity-50"
        >
          {isLoadingMore ? loadingLabel : loadMoreLabel}
        </Button>
      </div>
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
      <Button variant="ghost" size="auto"
        type="button"
        onClick={onReset}
        className="mt-2 rounded-lg border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
      >
        {t("cta")}
      </Button>
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
        <Button variant="destructive" size="auto"
          type="button"
          onClick={onRetry}
          className="mt-3 rounded-lg px-4 py-2 text-sm font-medium"
        >
          {retryLabel}
        </Button>
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
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onClose}
          aria-label={t("hideFilters")}
          className="absolute right-4 top-4 inline-flex h-9 w-9 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors hover:text-foreground"
        >
          <X className="h-4 w-4" aria-hidden />
        </Button>
        {children}
      </div>
    </div>
  )
}
