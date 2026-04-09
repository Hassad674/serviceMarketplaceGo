"use client"

import { useState } from "react"
import { Plus, Briefcase, Sparkles, ImagePlus } from "lucide-react"
import { useTranslations } from "next-intl"
import { useMyPortfolio, usePortfolioByUser, useDeletePortfolioItem } from "../hooks/use-portfolio"
import { PortfolioItemCard } from "./portfolio-item-card"
import { PortfolioDetailModal } from "./portfolio-detail-modal"
import { PortfolioFormModal } from "./portfolio-form-modal"
import type { PortfolioItem } from "../api/portfolio-api"

const MAX_ITEMS = 30

// --- Edit mode (profile dashboard) ---

export function PortfolioSection() {
  const { data, isLoading } = useMyPortfolio()
  const deleteItem = useDeletePortfolioItem()
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
    <section className="rounded-2xl border border-border bg-card p-6 shadow-sm">
      {/* Header */}
      <div className="mb-5 flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-rose-100 to-rose-50">
            <Briefcase className="h-5 w-5 text-rose-600" />
          </div>
          <div>
            <h2 className="text-lg font-semibold tracking-tight text-foreground">
              {t("sectionTitle")}
            </h2>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {items.length > 0
                ? t("publicItemCount", { count: items.length })
                : t("sectionSubtitle")}
            </p>
          </div>
        </div>

        {items.length > 0 && items.length < MAX_ITEMS && (
          <button
            onClick={openCreate}
            className="flex h-9 shrink-0 items-center gap-1.5 rounded-xl bg-gradient-to-r from-rose-500 to-rose-600 px-4 text-sm font-medium text-white shadow-md transition-all hover:shadow-lg hover:shadow-rose-500/30 active:scale-[0.98]"
          >
            <Plus className="h-4 w-4" />
            {t("addProject")}
          </button>
        )}
      </div>

      {/* Content */}
      {isLoading ? (
        <PortfolioGridSkeleton />
      ) : items.length === 0 ? (
        <EmptyState onCreate={openCreate} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((item, index) => (
            <div
              key={item.id}
              className="animate-slide-up"
              style={{ animationDelay: `${Math.min(index * 50, 250)}ms` }}
            >
              <PortfolioItemCard
                item={item}
                onView={() => setViewItem(item)}
                onEdit={() => openEdit(item)}
                onDelete={() => handleDelete(item.id)}
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
    <div className="relative overflow-hidden rounded-2xl border-2 border-dashed border-rose-200 bg-gradient-to-br from-rose-50/60 via-white to-purple-50/40 px-6 py-12 text-center">
      {/* Decorative blurs */}
      <div className="pointer-events-none absolute -left-16 -top-16 h-48 w-48 rounded-full bg-rose-200/30 blur-3xl" />
      <div className="pointer-events-none absolute -right-16 -bottom-16 h-48 w-48 rounded-full bg-purple-200/30 blur-3xl" />

      <div className="relative">
        <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-rose-500 to-rose-600 shadow-lg shadow-rose-500/30">
          <ImagePlus className="h-7 w-7 text-white" />
        </div>
        <h3 className="text-base font-semibold text-foreground">
          {t("emptyTitle")}
        </h3>
        <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
          {t("emptyDescription")}
        </p>
        <button
          onClick={onCreate}
          className="mt-5 inline-flex h-10 items-center gap-1.5 rounded-xl bg-gradient-to-r from-rose-500 to-rose-600 px-5 text-sm font-semibold text-white shadow-md transition-all hover:shadow-lg hover:shadow-rose-500/30 active:scale-[0.98]"
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
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {[0, 1, 2].map((i) => (
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
  userId: string
}

export function PublicPortfolioSection({ userId }: PublicPortfolioSectionProps) {
  const { data, isLoading } = usePortfolioByUser(userId)
  const [viewItem, setViewItem] = useState<PortfolioItem | null>(null)
  const t = useTranslations("portfolio")

  const items = data?.data ?? []

  if (!isLoading && items.length === 0) return null

  return (
    <section className="rounded-2xl border border-border bg-card p-6 shadow-sm">
      <div className="mb-5 flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-rose-100 to-rose-50">
          <Briefcase className="h-5 w-5 text-rose-600" />
        </div>
        <div>
          <h2 className="text-lg font-semibold tracking-tight text-foreground">
            {t("sectionTitle")}
          </h2>
          {items.length > 0 && (
            <p className="mt-0.5 text-xs text-muted-foreground">
              {t("publicItemCount", { count: items.length })}
            </p>
          )}
        </div>
      </div>

      {isLoading ? (
        <PortfolioGridSkeleton />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
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
