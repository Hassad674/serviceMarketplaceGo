"use client"

import { useState, useCallback } from "react"
import {
  Edit2,
  Loader2,
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
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useMySocialLinks, useUpsertSocialLink, useDeleteSocialLink } from "../hooks/use-social-links"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
type PlatformMeta = {
  key: string
  icon: LucideIcon
  color: string
}

const PLATFORMS: PlatformMeta[] = [
  { key: "linkedin", icon: Linkedin, color: "text-[#0A66C2]" },
  { key: "instagram", icon: Instagram, color: "text-[#E4405F]" },
  { key: "youtube", icon: Youtube, color: "text-[#FF0000]" },
  { key: "twitter", icon: Twitter, color: "text-foreground" },
  { key: "github", icon: Github, color: "text-foreground" },
  { key: "website", icon: Globe, color: "text-primary" },
]

export function SocialLinksSection() {
  const t = useTranslations("profile")
  const canEdit = useHasPermission("org_profile.edit")
  const { data: links = [], isLoading } = useMySocialLinks()
  const upsertMutation = useUpsertSocialLink()
  const deleteMutation = useDeleteSocialLink()
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)

  const startEditing = useCallback(() => {
    const initial: Record<string, string> = {}
    for (const link of links) {
      initial[link.platform] = link.url
    }
    setDraft(initial)
    setEditing(true)
  }, [links])

  const cancelEditing = useCallback(() => {
    setDraft({})
    setEditing(false)
  }, [])

  const handleSave = useCallback(async () => {
    setSaving(true)
    try {
      const existingPlatforms = new Set(links.map((l) => l.platform))

      for (const platform of PLATFORMS.map((p) => p.key)) {
        const url = draft[platform]?.trim()
        const hadBefore = existingPlatforms.has(platform)

        if (url) {
          await upsertMutation.mutateAsync({ platform, url })
        } else if (hadBefore) {
          await deleteMutation.mutateAsync(platform)
        }
      }
    } finally {
      setSaving(false)
      setEditing(false)
    }
  }, [draft, links, upsertMutation, deleteMutation])

  if (isLoading) {
    return <SocialLinksSkeleton />
  }

  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-foreground">{t("socialLinks")}</h2>
        {!editing && canEdit && (
          <Button variant="ghost" size="auto"
            type="button"
            onClick={startEditing}
            aria-label={t("editSocialLinks")}
            className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
          >
            <Edit2 className="w-[18px] h-[18px]" aria-hidden="true" />
          </Button>
        )}
      </div>

      {editing ? (
        <SocialLinksEditor
          draft={draft}
          setDraft={setDraft}
          saving={saving}
          onSave={handleSave}
          onCancel={cancelEditing}
        />
      ) : links.length > 0 ? (
        <SocialLinksDisplay links={links} />
      ) : (
        <p className="text-sm text-muted-foreground italic">{t("noSocialLinks")}</p>
      )}
    </section>
  )
}

// ---- Display mode ----

function SocialLinksDisplay({ links }: { links: { platform: string; url: string }[] }) {
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
            <div className={`p-2 rounded-lg bg-muted ${meta.color} group-hover:scale-110 transition-transform flex-shrink-0`}>
              <Icon className="h-[18px] w-[18px]" aria-hidden="true" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground truncate">{meta.key}</p>
              <p className="text-xs text-muted-foreground truncate">
                {link.url.replace(/(^\w+:|^)\/\//, "")}
              </p>
            </div>
            <ExternalLink className="h-3.5 w-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0" aria-hidden="true" />
          </a>
        )
      })}
    </div>
  )
}

// ---- Editor mode ----

interface SocialLinksEditorProps {
  draft: Record<string, string>
  setDraft: (draft: Record<string, string>) => void
  saving: boolean
  onSave: () => void
  onCancel: () => void
}

function SocialLinksEditor({ draft, setDraft, saving, onSave, onCancel }: SocialLinksEditorProps) {
  const t = useTranslations("profile")
  const tCommon = useTranslations("common")

  return (
    <div className="space-y-4">
      {PLATFORMS.map((meta) => {
        const Icon = meta.icon
        return (
          <div key={meta.key} className="space-y-1">
            <label htmlFor={`social-${meta.key}`} className="text-sm font-medium text-foreground flex items-center gap-2">
              <Icon className={`h-4 w-4 ${meta.color}`} aria-hidden="true" />
              {t(meta.key as "linkedin" | "instagram" | "youtube" | "twitter" | "github" | "website")}
            </label>
            <Input
              id={`social-${meta.key}`}
              type="url"
              value={draft[meta.key] || ""}
              onChange={(e) => setDraft({ ...draft, [meta.key]: e.target.value })}
              placeholder={t("enterUrl")}
              className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all duration-150"
            />
          </div>
        )
      })}

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onCancel}
          disabled={saving}
          className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted transition-colors duration-150 disabled:opacity-50"
        >
          {tCommon("cancel")}
        </Button>
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onSave}
          disabled={saving}
          className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 disabled:opacity-50 inline-flex items-center gap-2"
        >
          {saving && <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />}
          {tCommon("save")}
        </Button>
      </div>
    </div>
  )
}

// ---- Skeleton ----

function SocialLinksSkeleton() {
  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <div className="h-6 w-40 bg-muted rounded animate-shimmer mb-4" />
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {[1, 2].map((i) => (
          <div key={i} className="h-16 bg-muted rounded-lg animate-shimmer" />
        ))}
      </div>
    </section>
  )
}
