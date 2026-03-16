"use client"

import { useState, useRef } from "react"
import { Camera, Star, Edit2 } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import type { Profile } from "../api/profile-api"

type RoleContext = "agency" | "provider" | "referrer"

interface ProfileHeaderProps {
  profile: Profile | undefined
  displayName: string
  roleContext: RoleContext
  onUpdateTitle: (title: string) => void
}

const ROLE_LABELS: Record<RoleContext, { imageLabel: string; badge: string | null }> = {
  agency: { imageLabel: "Logo", badge: null },
  provider: { imageLabel: "Photo", badge: null },
  referrer: { imageLabel: "Photo", badge: "Apporteur d'affaire" },
}

export function ProfileHeader({
  profile,
  displayName,
  roleContext,
  onUpdateTitle,
}: ProfileHeaderProps) {
  const [isEditingTitle, setIsEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState(profile?.title ?? "")
  const fileInputRef = useRef<HTMLInputElement>(null)
  const titleInputRef = useRef<HTMLInputElement>(null)

  const { imageLabel, badge } = ROLE_LABELS[roleContext]
  const isRounded = roleContext === "agency"

  function handleTitleClick() {
    setTitleDraft(profile?.title ?? "")
    setIsEditingTitle(true)
    setTimeout(() => titleInputRef.current?.focus(), 0)
  }

  function handleTitleSubmit() {
    setIsEditingTitle(false)
    const trimmed = titleDraft.trim()
    if (trimmed && trimmed !== profile?.title) {
      onUpdateTitle(trimmed)
    }
  }

  function handleTitleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter") handleTitleSubmit()
    if (event.key === "Escape") setIsEditingTitle(false)
  }

  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <div className="flex flex-col sm:flex-row items-start gap-6">
        {/* Photo / Logo */}
        <div className="relative shrink-0">
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            className={cn(
              "w-24 h-24 bg-muted flex items-center justify-center overflow-hidden",
              "border-2 border-dashed border-border hover:border-primary transition-colors",
              "focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2",
              isRounded ? "rounded-lg" : "rounded-full",
            )}
            aria-label={`Modifier votre ${imageLabel.toLowerCase()}`}
          >
            {profile?.photo_url ? (
              <img
                src={profile.photo_url}
                alt={`${imageLabel} de ${displayName}`}
                className="w-full h-full object-cover"
              />
            ) : (
              <Camera className="w-8 h-8 text-muted-foreground" aria-hidden="true" />
            )}
          </button>
          <span className="absolute -bottom-2 left-1/2 -translate-x-1/2 text-xs text-muted-foreground bg-card px-2">
            {imageLabel}
          </span>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            aria-label={`Importer votre ${imageLabel.toLowerCase()}`}
          />
        </div>

        {/* Name, title, stats */}
        <div className="flex-1 min-w-0 space-y-2">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-xl font-semibold text-foreground truncate">
              {displayName}
            </h1>
            {badge && (
              <span className="rounded-full bg-accent text-accent-foreground px-3 py-1 text-xs font-medium">
                {badge}
              </span>
            )}
          </div>

          {/* Editable title */}
          {isEditingTitle ? (
            <input
              ref={titleInputRef}
              type="text"
              value={titleDraft}
              onChange={(event) => setTitleDraft(event.target.value)}
              onBlur={handleTitleSubmit}
              onKeyDown={handleTitleKeyDown}
              placeholder="Votre titre professionnel"
              className="w-full max-w-md bg-muted border border-input rounded-md px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              aria-label="Titre professionnel"
            />
          ) : (
            <button
              type="button"
              onClick={handleTitleClick}
              className="group flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Modifier le titre professionnel"
            >
              <span className={cn(!profile?.title && "italic")}>
                {profile?.title || "Ajouter un titre professionnel"}
              </span>
              <Edit2 className="w-3.5 h-3.5 opacity-0 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
            </button>
          )}

          <p className="text-sm text-muted-foreground">0 projets termines</p>
        </div>

        {/* Rating placeholder */}
        <div className="flex items-center gap-2 text-sm text-muted-foreground shrink-0">
          <Star className="w-4 h-4" aria-hidden="true" />
          <span>Aucun avis</span>
        </div>
      </div>
    </section>
  )
}
