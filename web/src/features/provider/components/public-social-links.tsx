"use client"

import {
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
import { usePublicSocialLinks } from "../hooks/use-social-links"

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

interface PublicSocialLinksProps {
  userId: string
}

export function PublicSocialLinks({ userId }: PublicSocialLinksProps) {
  const t = useTranslations("profile")
  const { data: links = [], isLoading } = usePublicSocialLinks(userId)

  if (isLoading) return null
  if (links.length === 0) return null

  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-foreground mb-4">{t("socialLinks")}</h2>
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
    </section>
  )
}
