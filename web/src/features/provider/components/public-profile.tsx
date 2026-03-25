"use client"

import { useQuery } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { Star, ArrowLeft } from "lucide-react"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { apiClient } from "@/shared/lib/api-client"
import type { Profile } from "../api/profile-api"

type ProfileType = "agency" | "freelancer" | "referrer"

const TYPE_BACK_LINKS: Record<ProfileType, string> = {
  agency: "/agencies",
  freelancer: "/freelancers",
  referrer: "/freelancers",
}

const TYPE_BACK_LABELS: Record<ProfileType, string> = {
  agency: "backToAgencies",
  freelancer: "backToFreelancers",
  referrer: "backToFreelancers",
}

interface PublicProfileProps {
  userId: string
  type: ProfileType
}

export function PublicProfile({ userId, type }: PublicProfileProps) {
  const t = useTranslations("publicProfile")

  const { data: profile, isLoading, error } = useQuery({
    queryKey: ["public-profile", userId],
    queryFn: () => apiClient<Profile>(`/api/v1/profiles/${userId}`),
  })

  if (isLoading) {
    return <PublicProfileSkeleton />
  }

  if (error || !profile) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 dark:border-red-500/20 dark:bg-red-500/10 p-8 text-center">
        <p className="text-sm text-red-600 dark:text-red-400">
          {t("profileNotFound")}
        </p>
        <Link
          href={TYPE_BACK_LINKS[type]}
          className="mt-3 inline-flex items-center gap-1.5 text-sm text-rose-600 hover:text-rose-700 dark:text-rose-400"
        >
          <ArrowLeft className="h-4 w-4" />
          {t(TYPE_BACK_LABELS[type])}
        </Link>
      </div>
    )
  }

  const initials = profile.user_id.substring(0, 2).toUpperCase()

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        href={TYPE_BACK_LINKS[type]}
        className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t(TYPE_BACK_LABELS[type])}
      </Link>

      {/* Profile header */}
      <section className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 shadow-sm">
        <div className="flex flex-col sm:flex-row items-start gap-5">
          {/* Photo */}
          <div className="shrink-0">
            {profile.photo_url ? (
              <img
                src={profile.photo_url}
                alt={t("photoAlt")}
                className={cn(
                  "h-24 w-24 object-cover",
                  type === "agency" ? "rounded-lg" : "rounded-full",
                )}
              />
            ) : (
              <div
                className={cn(
                  "flex h-24 w-24 items-center justify-center bg-gradient-to-br from-rose-500 to-purple-600 text-xl font-semibold text-white",
                  type === "agency" ? "rounded-lg" : "rounded-full",
                )}
              >
                {initials}
              </div>
            )}
          </div>

          {/* Info */}
          <div className="flex-1 min-w-0 space-y-1.5">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {profile.title || t("untitledProfile")}
            </h1>
            <div className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400">
              <Star className="h-4 w-4" />
              <span>{t("noReviews")}</span>
            </div>
          </div>
        </div>
      </section>

      {/* About section */}
      {profile.about && (
        <section className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            {t("about")}
          </h2>
          <p className="mt-3 text-sm leading-relaxed text-gray-600 dark:text-gray-300 whitespace-pre-line">
            {profile.about}
          </p>
        </section>
      )}

      {/* Video section */}
      {profile.presentation_video_url && (
        <section className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            {t("presentationVideo")}
          </h2>
          <div className="mt-3">
            <video
              src={profile.presentation_video_url}
              controls
              className="w-full max-w-2xl rounded-lg"
            >
              {t("videoNotSupported")}
            </video>
          </div>
        </section>
      )}

      {/* Project history placeholder */}
      <section className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          {t("projectHistory")}
        </h2>
        <div className="mt-3 rounded-lg border border-dashed border-gray-200 dark:border-gray-700 p-8 text-center">
          <p className="text-sm text-gray-400 dark:text-gray-500">
            {t("noProjectsYet")}
          </p>
        </div>
      </section>
    </div>
  )
}

function PublicProfileSkeleton() {
  return (
    <div className="space-y-6">
      <div className="h-4 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
      <div className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 shadow-sm">
        <div className="flex items-start gap-5">
          <div className="h-24 w-24 shrink-0 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
          <div className="flex-1 space-y-3">
            <div className="h-6 w-1/2 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-4 w-1/4 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          </div>
        </div>
      </div>
      <div className="rounded-xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 shadow-sm">
        <div className="h-5 w-24 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
        <div className="mt-3 space-y-2">
          <div className="h-3 w-full animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          <div className="h-3 w-4/5 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          <div className="h-3 w-3/5 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
        </div>
      </div>
    </div>
  )
}
