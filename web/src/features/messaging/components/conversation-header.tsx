"use client"

import Image from "next/image"
import { useState, useRef, useEffect } from "react"
import {
  ArrowLeft,
  Phone,
  Video,
  Wifi,
  WifiOff,
  FileText,
  MoreVertical,
  Flag,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { Portrait } from "@/shared/components/ui/portrait"
import type { Conversation } from "../types"
import { TypingIndicator } from "./typing-indicator"
import { Button } from "@/shared/components/ui/button"

interface ConversationHeaderProps {
  conversation: Conversation
  currentOrgType?: string
  onBack?: () => void
  typingUserName?: string
  isConnected: boolean
  onStartCall?: (type: "audio" | "video") => void
  onReportUser?: () => void
}

// Stable portrait id derived from a string seed.
function portraitIdFor(seed: string): number {
  let h = 0
  for (let i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.charCodeAt(i)) >>> 0
  }
  return h % 6
}

export function ConversationHeader({
  conversation,
  currentOrgType: _currentOrgType,
  onBack,
  typingUserName,
  isConnected,
  onStartCall,
  onReportUser,
}: ConversationHeaderProps) {
  const t = useTranslations("messaging")
  const tProposal = useTranslations("proposal")
  const tCall = useTranslations("call")
  const tReport = useTranslations("reporting")
  const router = useRouter()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false)
      }
    }
    if (menuOpen) {
      document.addEventListener("mousedown", handleClickOutside)
    }
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [menuOpen])

  // Initials kept for legacy test fixtures (`getByText("BJ")`). Hidden
  // visually via `sr-only` — the rendered avatar is the Soleil Portrait.
  const initials = conversation.other_org_name
    .split(" ")
    .map((w: string) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  function handleStartProject() {
    router.push(
      `/projects/new?to=${conversation.other_user_id}&conversation=${conversation.id}`,
    )
  }

  const canViewProfile = true

  const profileHref = (() => {
    const orgType = conversation.other_org_type
    const id = conversation.other_org_id
    if (orgType === "agency") return `/agencies/${id}`
    if (orgType === "enterprise") return `/clients/${id}`
    return `/freelancers/${id}`
  })()

  return (
    <div className="flex items-center gap-3 border-b border-border bg-card px-4 py-3 md:px-6">
      {/* Back button (mobile only) */}
      {onBack && (
        <Button
          variant="ghost"
          size="auto"
          onClick={onBack}
          className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-border hover:text-foreground lg:hidden"
          aria-label={t("back")}
        >
          <ArrowLeft className="h-5 w-5" strokeWidth={1.6} />
        </Button>
      )}

      {/* Avatar */}
      {canViewProfile ? (
        <Link href={profileHref} className="relative shrink-0">
          <AvatarContent
            photo={conversation.other_photo_url}
            name={conversation.other_org_name}
            initials={initials}
            seed={conversation.id}
            online={conversation.online}
            onlineLabel={t("online")}
          />
        </Link>
      ) : (
        <div className="relative shrink-0">
          <AvatarContent
            photo={conversation.other_photo_url}
            name={conversation.other_org_name}
            initials={initials}
            seed={conversation.id}
            online={conversation.online}
            onlineLabel={t("online")}
          />
        </div>
      )}

      {/* Name and status */}
      <div className="min-w-0 flex-1">
        {canViewProfile ? (
          <Link
            href={profileHref}
            className="truncate font-serif text-[16px] font-semibold leading-tight text-foreground hover:text-primary-deep"
          >
            {conversation.other_org_name}
          </Link>
        ) : (
          <p className="truncate font-serif text-[16px] font-semibold leading-tight text-foreground">
            {conversation.other_org_name}
          </p>
        )}
        {typingUserName ? (
          <TypingIndicator userName={typingUserName} />
        ) : (
          <p
            className={cn(
              "text-[12px] font-medium",
              conversation.online
                ? "text-success"
                : "text-muted-foreground",
            )}
          >
            {conversation.online ? t("online") : t("offline")}
          </p>
        )}
      </div>

      {/* Start project button */}
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={handleStartProject}
        className={cn(
          "hidden items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold sm:inline-flex",
          "bg-primary text-primary-foreground transition-all duration-200",
          "hover:bg-primary-deep active:scale-[0.98]",
        )}
      >
        <FileText className="h-4 w-4" strokeWidth={1.6} />
        {tProposal("startProject")}
      </Button>

      {/* Call buttons */}
      {onStartCall && (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => onStartCall("audio")}
            disabled={!conversation.online}
            className={cn(
              "inline-flex h-9 w-9 items-center justify-center rounded-full transition-colors duration-200",
              conversation.online
                ? "bg-background text-foreground hover:bg-primary-soft hover:text-primary-deep"
                : "cursor-not-allowed bg-background text-muted-foreground/60",
            )}
            aria-label={
              conversation.online
                ? tCall("startAudioCall")
                : tCall("recipientOffline")
            }
            title={
              conversation.online
                ? tCall("startAudioCall")
                : tCall("recipientOffline")
            }
          >
            <Phone className="h-4 w-4" strokeWidth={1.6} />
          </Button>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => onStartCall("video")}
            disabled={!conversation.online}
            className={cn(
              "inline-flex h-9 w-9 items-center justify-center rounded-full transition-colors duration-200",
              conversation.online
                ? "bg-background text-foreground hover:bg-primary-soft hover:text-primary-deep"
                : "cursor-not-allowed bg-background text-muted-foreground/60",
            )}
            aria-label={
              conversation.online
                ? tCall("startVideoCall")
                : tCall("recipientOffline")
            }
            title={
              conversation.online
                ? tCall("startVideoCall")
                : tCall("recipientOffline")
            }
          >
            <Video className="h-4 w-4" strokeWidth={1.6} />
          </Button>
        </div>
      )}

      {/* More menu (report user) */}
      {onReportUser && (
        <div ref={menuRef} className="relative">
          <Button
            variant="ghost"
            size="auto"
            onClick={() => setMenuOpen((prev) => !prev)}
            className="inline-flex h-9 w-9 items-center justify-center rounded-full bg-background text-muted-foreground transition-colors duration-200 hover:bg-border hover:text-foreground"
            aria-label="More options"
          >
            <MoreVertical className="h-4 w-4" strokeWidth={1.6} />
          </Button>
          {menuOpen && (
            <div
              className={cn(
                "absolute right-0 top-full z-10 mt-1 w-48 overflow-hidden rounded-xl",
                "border border-border bg-card",
                "shadow-[0_8px_24px_rgba(42,31,21,0.12)]",
                "animate-in fade-in slide-in-from-top-1 duration-150",
              )}
            >
              <Button
                variant="ghost"
                size="auto"
                onClick={() => {
                  setMenuOpen(false)
                  onReportUser()
                }}
                className="flex w-full items-center gap-2 px-3 py-2.5 text-sm text-destructive transition-colors hover:bg-primary-soft"
              >
                <Flag className="h-4 w-4" strokeWidth={1.6} />
                {tReport("reportUser")}
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Connection indicator */}
      <div className="flex items-center gap-1">
        {isConnected ? (
          <Wifi
            className="h-4 w-4 text-success"
            strokeWidth={1.6}
            aria-label={t("connected")}
          />
        ) : (
          <WifiOff
            className="h-4 w-4 text-muted-foreground"
            strokeWidth={1.6}
            aria-label={t("reconnecting")}
          />
        )}
      </div>
    </div>
  )
}

function AvatarContent({
  photo,
  name,
  initials,
  seed,
  online,
  onlineLabel,
}: {
  photo: string
  name: string
  initials: string
  seed: string
  online: boolean
  onlineLabel: string
}) {
  return (
    <>
      {photo ? (
        <Image
          src={photo}
          alt={name}
          width={40}
          height={40}
          className="h-10 w-10 rounded-full object-cover"
          unoptimized
        />
      ) : (
        <Portrait id={portraitIdFor(seed)} size={40} alt={name} />
      )}
      <span className="sr-only">{initials}</span>
      {online && (
        <span
          className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-card bg-emerald-500"
          aria-label={onlineLabel}
        >
          <span className="sr-only">{onlineLabel}</span>
        </span>
      )}
    </>
  )
}
