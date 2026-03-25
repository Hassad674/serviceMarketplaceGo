"use client"

import { useAuth } from "@/shared/hooks/use-auth"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"

export default function ProviderReferralPage() {
  const { user } = useAuth()
  const { data: profile, isLoading } = useProfile()
  const updateProfile = useUpdateProfile()

  if (isLoading) return <ProfileSkeleton />

  const displayName = user
    ? `${user.first_name} ${user.last_name}`
    : ""

  return (
    <div className="space-y-6">
      <ProfileHeader
        profile={profile}
        displayName={displayName}
        roleContext="referrer"
        onUpdateTitle={(title) => updateProfile.mutate({ title })}
      />
      <ProfileVideo
        videoUrl={profile?.referrer_video_url}
        title="Video de presentation — Apporteur d'affaire"
        emptyLabel="Aucune video d'apporteur"
        emptyDescription="Ajoutez une video pour presenter votre activite d'apporteur d'affaire"
      />
    </div>
  )
}
