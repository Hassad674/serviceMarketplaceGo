"use client"

import { useState } from "react"
import {
  Edit2,
  Linkedin,
  Instagram,
  Youtube,
  Twitter,
  Github,
  Globe,
  ExternalLink,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { useTranslations } from "next-intl"

import { Button } from "@/shared/components/ui/button"
import { SocialLinksEditorModal } from "./social-links-editor-modal"
// Shape of a single link used by the shared card. Kept minimal so
// every persona feature can re-use the card without having to expose
// its full API response shape.
export interface SocialLinkEntry {
  platform: string
  url: string
}

// Editor wiring passed by the parent feature. The card never knows
// about mutation hooks — it just fires callbacks. Fits well with
// TanStack Query mutations, Zustand stores, or any ad-hoc promises.
export interface SocialLinksEditor {
  canEdit: boolean
  onUpsert: (platform: string, url: string) => Promise<void>
  onDelete: (platform: string) => Promise<void>
}

export interface SocialLinksCardProps {
  links: SocialLinkEntry[]
  isLoading?: boolean
  editor?: SocialLinksEditor
}

type PlatformMeta = {
  key: string
  icon: LucideIcon
  color: string
}

// Canonical platform list — matches the backend allowlist. Kept
// private to the card so each feature stays decoupled from the
// platform enum.
const PLATFORMS: PlatformMeta[] = [
  { key: "linkedin", icon: Linkedin, color: "text-[#0A66C2]" },
  { key: "instagram", icon: Instagram, color: "text-[#E4405F]" },
  { key: "youtube", icon: Youtube, color: "text-[#FF0000]" },
  { key: "twitter", icon: Twitter, color: "text-foreground" },
  { key: "github", icon: Github, color: "text-foreground" },
  { key: "website", icon: Globe, color: "text-primary" },
]

// SocialLinksCard renders a persona's social link set. When `editor`
// is supplied and `editor.canEdit` is true it shows an edit button
// that opens the shared SocialLinksEditorModal — otherwise it stays
// read-only. Collapses to nothing when read-only and empty, so public
// viewers see no orphan card.
export function SocialLinksCard({
  links,
  isLoading = false,
  editor,
}: SocialLinksCardProps) {
  const t = useTranslations("profile")
  const canEdit = Boolean(editor?.canEdit)
  const [modalOpen, setModalOpen] = useState(false)

  if (isLoading) {
    return <SocialLinksSkeleton />
  }

  // Read-only + empty: collapse to nothing so public viewers never
  // see an empty placeholder card.
  if (!canEdit && links.length === 0) {
    return null
  }

  return (
    <>
      <section className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]">
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-serif text-xl font-medium tracking-[-0.005em] text-foreground">
            {t("socialLinks")}
          </h2>
          {canEdit ? (
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setModalOpen(true)}
              aria-label={t("editSocialLinks")}
              className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              <Edit2 className="w-[18px] h-[18px]" aria-hidden="true" />
            </Button>
          ) : null}
        </div>

        {links.length > 0 ? (
          <SocialLinksDisplay links={links} />
        ) : (
          <p className="text-sm text-muted-foreground italic">
            {t("noSocialLinks")}
          </p>
        )}
      </section>

      {editor && canEdit ? (
        <SocialLinksEditorModal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          links={links}
          onUpsert={editor.onUpsert}
          onDelete={editor.onDelete}
        />
      ) : null}
    </>
  )
}

// ---- Display mode ----

function SocialLinksDisplay({ links }: { links: SocialLinkEntry[] }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
      {links.map((link) => {
        const meta = PLATFORMS.find((p) => p.key === link.platform)
        if (!meta) return null
        const Icon = meta.icon
        return (
          <a
            key={link.platform}
            href={link.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 p-3 rounded-lg border border-border hover:border-primary/30 hover:bg-muted/50 transition-all duration-150 group"
          >
            <div
              className={`p-2 rounded-lg bg-muted ${meta.color} group-hover:scale-110 transition-transform flex-shrink-0`}
            >
              <Icon className="h-[18px] w-[18px]" aria-hidden="true" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground truncate">
                {meta.key}
              </p>
              <p className="text-xs text-muted-foreground truncate">
                {link.url.replace(/(^\w+:|^)\/\//, "")}
              </p>
            </div>
            <ExternalLink
              className="h-3.5 w-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
              aria-hidden="true"
            />
          </a>
        )
      })}
    </div>
  )
}

// ---- Skeleton ----

function SocialLinksSkeleton() {
  return (
    <section className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]">
      <div className="h-6 w-40 bg-muted rounded animate-shimmer mb-4" />
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {[1, 2].map((i) => (
          <div key={i} className="h-16 bg-muted rounded-lg animate-shimmer" />
        ))}
      </div>
    </section>
  )
}
