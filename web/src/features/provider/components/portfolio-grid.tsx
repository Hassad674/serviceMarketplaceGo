"use client"

import { useState } from "react"
import { Plus, Briefcase, Sparkles, ImagePlus } from "lucide-react"
import { useTranslations } from "next-intl"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useMyPortfolio, usePortfolioByOrganization, useDeletePortfolioItem } from "../hooks/use-portfolio"
import { PortfolioItemCard } from "./portfolio-item-card"
import { PortfolioDetailModal } from "./portfolio-detail-modal"
import { PortfolioFormModal } from "./portfolio-form-modal"
import type { PortfolioItem } from "../api/portfolio-api"

const MAX_ITEMS = 30

// --- Edit mode (profile dashboard) ---

export function PortfolioSection() {
  const { data, isLoading } = useMyPortfolio()
  const deleteItem = useDeletePortfolioItem()
  const canEdit = useHasPermission("org_profile.edit")
  const t = useTranslations("portfolio")

  const [viewItem, setViewItem] = useState<PortfolioItem | null>(null)
  const [editItem, setEditItem] = useState<PortfolioItem | undefined>(undefined)
  const [showForm, setShowForm] = useState(false)

  const items = data?.data ?? []

  const handleDelete = (id: string) => {
    if (window.confirm(t("confirmDelete"))) {
      deleteItem.mutate(id)
    }
  }

  const openCreate = () => {
    setEditItem(undefined)
    setShowForm(true)
  }

  const openEdit = (item: PortfolioItem) => {
    setEditItem(item)
    setShowForm(true)
  }

  return (
    <section className="rounded-2xl border border-border bg-card p-4 shadow-sm sm:p-6">
      {/* Header */}
      <div className="mb-4 flex items-center justify-between gap-3 sm:mb-5">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-rose-100 to-rose-50 sm:h-10 sm:w-10">
            <Briefcase className="h-5 w-5 text-rose-600" />
          </div>
          <div className="min-w-0">
            <h2 className="truncate text-base font-semibold tracking-tight text-foreground sm:text-lg">
              {t("sectionTitle")}
            </h2>
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {items.length > 0
                ? t("publicItemCount", { count: items.length })
                : t("sectionSubtitle")}
            </p>
          </div>
        </div>

        {canEdit && items.length > 0 && items.length < MAX_ITEMS && (
          <button
            onClick={openCreate}
            aria-label={t("addProject")}
            className="flex h-9 shrink-0 items-center gap-1.5 rounded-xl bg-gradient-to-r from-rose-500 to-rose-600 px-3 text-sm font-medium text-white shadow-md transition-all hover:shadow-lg hover:shadow-rose-500/30 active:scale-[0.98] sm:px-4"
          >
            <Plus className="h-4 w-4" />
            <span className="hidden sm:inline">{t("addProject")}</span>
          </button>
        )}
      </div>

      {/* Content */}
      {isLoading ? (
        <PortfolioGridSkeleton />
      ) : items.length === 0 ? (
        canEdit ? <EmptyState onCreate={openCreate} /> : null
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3">
          {items.map((item, index) => (
            <div
              key={item.id}
              className="animate-slide-up"
              style={{ animationDelay: `${Math.min(index * 50, 250)}ms` }}
            >
              <PortfolioItemCard
                item={item}
                readOnly={!canEdit}
                onView={() => setViewItem(item)}
                onEdit={canEdit ? () => openEdit(item) : undefined}
                onDelete={canEdit ? () => handleDelete(item.id) : undefined}
              />
            </div>
          ))}
        </div>
      )}

      <PortfolioDetailModal
        item={viewItem}
        open={!!viewItem}
        onClose={() => setViewItem(null)}
      />

      <PortfolioFormModal
        item={editItem}
        open={showForm}
        onClose={() => setShowForm(false)}
        nextPosition={items.length}
      />
    </section>
  )
}

// --- Empty state ---

function EmptyState({ onCreate }: { onCreate: () => void }) {
  const t = useTranslations("portfolio")
  return (
    <div className="relative overflow-hidden rounded-2xl border-2 border-dashed border-rose-200 bg-gradient-to-br from-rose-50/60 via-white to-purple-50/40 px-4 py-8 text-center sm:px-6 sm:py-12">
      {/* Decorative blurs */}
      <div className="pointer-events-none absolute -left-16 -top-16 h-48 w-48 rounded-full bg-rose-200/30 blur-3xl" />
      <div className="pointer-events-none absolute -right-16 -bottom-16 h-48 w-48 rounded-full bg-purple-200/30 blur-3xl" />

      <div className="relative">
        <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-rose-500 to-rose-600 shadow-lg shadow-rose-500/30 sm:mb-4 sm:h-16 sm:w-16">
          <ImagePlus className="h-6 w-6 text-white sm:h-7 sm:w-7" />
        </div>
        <h3 className="text-base font-semibold text-foreground">
          {t("emptyTitle")}
        </h3>
        <p className="mx-auto mt-1 max-w-sm text-xs text-muted-foreground sm:text-sm">
          {t("emptyDescription")}
        </p>
        <button
          onClick={onCreate}
          className="mt-4 inline-flex h-10 items-center gap-1.5 rounded-xl bg-gradient-to-r from-rose-500 to-rose-600 px-4 text-sm font-semibold text-white shadow-md transition-all hover:shadow-lg hover:shadow-rose-500/30 active:scale-[0.98] sm:mt-5 sm:px-5"
        >
          <Sparkles className="h-4 w-4" />
          {t("addFirstProject")}
        </button>
      </div>
    </div>
  )
}

// --- Skeleton loading ---

function PortfolioGridSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3">
      {[0, 1, 2, 3].map((i) => (
        <div
          key={i}
          className="aspect-[4/5] animate-shimmer rounded-2xl bg-gradient-to-br from-muted via-muted/60 to-muted"
        />
      ))}
    </div>
  )
}

// --- Read-only mode (public profile) ---

interface PublicPortfolioSectionProps {
  orgId: string
}

export function PublicPortfolioSection({ orgId }: PublicPortfolioSectionProps) {
  const { data, isLoading } = usePortfolioByOrganization(orgId)
  const [viewItem, setViewItem] = useState<PortfolioItem | null>(null)
  const t = useTranslations("portfolio")

  const items = data?.data ?? []

  if (!isLoading && items.length === 0) return null

  return (
    <section className="rounded-2xl border border-border bg-card p-4 shadow-sm sm:p-6">
      <div className="mb-4 flex items-center gap-3 sm:mb-5">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-rose-100 to-rose-50 sm:h-10 sm:w-10">
          <Briefcase className="h-5 w-5 text-rose-600" />
        </div>
        <div className="min-w-0">
          <h2 className="truncate text-base font-semibold tracking-tight text-foreground sm:text-lg">
            {t("sectionTitle")}
          </h2>
          {items.length > 0 && (
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {t("publicItemCount", { count: items.length })}
            </p>
          )}
        </div>
      </div>

      {isLoading ? (
        <PortfolioGridSkeleton />
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3">
          {items.map((item, index) => (
            <div
              key={item.id}
              className="animate-slide-up"
              style={{ animationDelay: `${Math.min(index * 50, 250)}ms` }}
            >
              <PortfolioItemCard
                item={item}
                readOnly
                onView={() => setViewItem(item)}
              />
            </div>
          ))}
        </div>
      )}

      <PortfolioDetailModal
        item={viewItem}
        open={!!viewItem}
        onClose={() => setViewItem(null)}
      />
    </section>
  )
}
