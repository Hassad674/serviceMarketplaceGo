"use client"

import { useEffect, useMemo } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import {
  Github,
  Globe,
  Instagram,
  Linkedin,
  Loader2,
  Twitter,
  Youtube,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { useTranslations } from "next-intl"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
import { Modal } from "@/shared/components/ui/modal"
import type { SocialLinkEntry } from "./social-links-card"

/**
 * SocialLinksEditorModal — Soleil v2 dialog for editing the six
 * supported social networks. Replaces the previous inline editor on
 * every profile page (provider, freelance, referrer) so the UX matches
 * the rest of the profile page (which already uses modals everywhere).
 *
 * Validation strategy:
 *   - Each platform field is optional.
 *   - When non-empty, the value must:
 *       1. parse as a valid URL (zod's z.string().url())
 *       2. have a hostname that matches the platform's regex
 *          (e.g. linkedin.com for linkedin, youtube.com|youtu.be for
 *          youtube, twitter.com|x.com for twitter).
 *   - The "website" field has no domain restriction — any valid URL.
 *
 * The save button stays disabled while:
 *   - the form is invalid, OR
 *   - the form is unchanged from initial values, OR
 *   - a save is currently in flight.
 *
 * The modal is dumb about persistence: it calls `onUpsert` for every
 * non-empty value and `onDelete` for every previously set value that
 * was cleared. The parent (provider/freelance/referrer section) wires
 * the actual TanStack Query mutations.
 */

interface PlatformMeta {
  key: PlatformKey
  icon: LucideIcon
  color: string
  /** Hostname allowlist regex. `null` for free-form URLs (website). */
  hostnameRegex: RegExp | null
  /** i18n key under "profile" namespace for the validation error. */
  errorKey: string
}

type PlatformKey =
  | "linkedin"
  | "instagram"
  | "youtube"
  | "twitter"
  | "github"
  | "website"

const PLATFORMS: readonly PlatformMeta[] = [
  {
    key: "linkedin",
    icon: Linkedin,
    color: "text-[#0A66C2]",
    hostnameRegex: /(^|\.)linkedin\.com$/i,
    errorKey: "socialLinkErrorLinkedin",
  },
  {
    key: "instagram",
    icon: Instagram,
    color: "text-[#E4405F]",
    hostnameRegex: /(^|\.)instagram\.com$/i,
    errorKey: "socialLinkErrorInstagram",
  },
  {
    key: "youtube",
    icon: Youtube,
    color: "text-[#FF0000]",
    hostnameRegex: /(^|\.)(youtube\.com|youtu\.be)$/i,
    errorKey: "socialLinkErrorYoutube",
  },
  {
    key: "twitter",
    icon: Twitter,
    color: "text-foreground",
    hostnameRegex: /(^|\.)(twitter\.com|x\.com)$/i,
    errorKey: "socialLinkErrorTwitter",
  },
  {
    key: "github",
    icon: Github,
    color: "text-foreground",
    hostnameRegex: /(^|\.)github\.com$/i,
    errorKey: "socialLinkErrorGithub",
  },
  {
    key: "website",
    icon: Globe,
    color: "text-primary",
    hostnameRegex: null,
    errorKey: "",
  },
] as const

type FormValues = Record<PlatformKey, string>

/**
 * Build the zod schema with i18n-aware error messages. Recreated on
 * every render where the locale could change — cheap, and keeps the
 * messages in sync with the active translation.
 *
 * We model every field as a `z.string()` (allowing the empty string)
 * + two refinements, instead of `z.string().url().optional()`. Reason:
 * react-hook-form initialises every field with `""` and zodResolver
 * requires the input + output types to be identical — using
 * `.optional()` here would diverge the input/output types and break
 * the typing on `useForm<FormValues>`.
 */
function buildSchema(
  tInvalid: string,
  tDomainErrors: Record<PlatformKey, string>,
) {
  const fieldFor = (key: PlatformKey) => {
    const meta = PLATFORMS.find((p) => p.key === key)!
    return z
      .string()
      .trim()
      .refine(
        (value) => {
          if (!value) return true
          // Step 1: must be a valid URL.
          try {
            new URL(value)
            return true
          } catch {
            return false
          }
        },
        { message: tInvalid },
      )
      .refine(
        (value) => {
          if (!value) return true
          if (!meta.hostnameRegex) return true
          let hostname: string
          try {
            hostname = new URL(value).hostname
          } catch {
            // Invalid URL — first refine already caught it; skip.
            return true
          }
          return meta.hostnameRegex.test(hostname)
        },
        { message: tDomainErrors[key] },
      )
  }

  return z.object({
    linkedin: fieldFor("linkedin"),
    instagram: fieldFor("instagram"),
    youtube: fieldFor("youtube"),
    twitter: fieldFor("twitter"),
    github: fieldFor("github"),
    website: fieldFor("website"),
  })
}

function entriesToValues(links: SocialLinkEntry[]): FormValues {
  const seed: FormValues = {
    linkedin: "",
    instagram: "",
    youtube: "",
    twitter: "",
    github: "",
    website: "",
  }
  for (const link of links) {
    if (link.platform in seed) {
      seed[link.platform as PlatformKey] = link.url
    }
  }
  return seed
}

export interface SocialLinksEditorModalProps {
  open: boolean
  onClose: () => void
  /**
   * Current persisted set — used to seed defaults and to decide which
   * platforms got cleared (delete) vs updated (upsert) on submit.
   */
  links: SocialLinkEntry[]
  onUpsert: (platform: string, url: string) => Promise<void>
  onDelete: (platform: string) => Promise<void>
}

export function SocialLinksEditorModal({
  open,
  onClose,
  links,
  onUpsert,
  onDelete,
}: SocialLinksEditorModalProps) {
  const t = useTranslations("profile")
  const tCommon = useTranslations("common")

  const initialValues = useMemo(() => entriesToValues(links), [links])

  const schema = useMemo(() => {
    const domainErrors: Record<PlatformKey, string> = {
      linkedin: t("socialLinkErrorLinkedin"),
      instagram: t("socialLinkErrorInstagram"),
      youtube: t("socialLinkErrorYoutube"),
      twitter: t("socialLinkErrorTwitter"),
      github: t("socialLinkErrorGithub"),
      website: "",
    }
    return buildSchema(t("socialLinksUrlInvalid"), domainErrors)
  }, [t])

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isDirty, isValid, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: initialValues,
    mode: "onChange",
  })

  // Reset the form whenever the modal re-opens or the persisted links
  // change. Without this, reopening the modal after a save shows the
  // stale "isDirty" state.
  useEffect(() => {
    if (open) {
      reset(initialValues)
    }
  }, [open, initialValues, reset])

  async function onSubmit(values: FormValues) {
    const previous = entriesToValues(links)
    for (const meta of PLATFORMS) {
      const next = (values[meta.key] ?? "").trim()
      const before = (previous[meta.key] ?? "").trim()
      if (next && next !== before) {
        await onUpsert(meta.key, next)
      } else if (!next && before) {
        await onDelete(meta.key)
      }
    }
    onClose()
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t("socialLinksModalTitle")}
      maxWidthClassName="max-w-lg"
    >
      <form
        noValidate
        onSubmit={handleSubmit(onSubmit)}
        className="space-y-4"
      >
        <p className="text-sm text-muted-foreground">
          {t("socialLinksModalDescription")}
        </p>

        {PLATFORMS.map((meta) => {
          const fieldError = errors[meta.key]
          const inputId = `social-link-${meta.key}`
          const errorId = `${inputId}-error`
          const Icon = meta.icon
          return (
            <div key={meta.key} className="space-y-1.5">
              <label
                htmlFor={inputId}
                className="flex items-center gap-2 text-sm font-medium text-foreground"
              >
                <Icon
                  className={`h-4 w-4 ${meta.color}`}
                  aria-hidden="true"
                />
                {t(meta.key)}
              </label>
              <Input
                id={inputId}
                type="url"
                placeholder={t("enterUrl")}
                autoComplete="off"
                aria-invalid={Boolean(fieldError) || undefined}
                aria-describedby={fieldError ? errorId : undefined}
                disabled={isSubmitting}
                className="w-full"
                {...register(meta.key)}
              />
              {fieldError ? (
                <p
                  id={errorId}
                  role="alert"
                  className="text-xs text-red-600 dark:text-red-400"
                >
                  {fieldError.message}
                </p>
              ) : null}
            </div>
          )
        })}

        <div className="flex justify-end gap-2 pt-2">
          <Button
            type="button"
            variant="ghost"
            size="md"
            onClick={onClose}
            disabled={isSubmitting}
          >
            {tCommon("cancel")}
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="md"
            disabled={!isDirty || !isValid || isSubmitting}
            className="inline-flex items-center gap-2"
          >
            {isSubmitting ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            ) : null}
            {tCommon("save")}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
