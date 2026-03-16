"use client"

import { useAuth } from "@/shared/hooks/use-auth"
import { useProfile, useUpdateProfile } from "@/features/provider/hooks/use-profile"
import { ProfileHeader } from "@/features/provider/components/profile-header"
import { ProfileVideo } from "@/features/provider/components/profile-video"
import { ProfileHistory } from "@/features/provider/components/profile-history"
import { ProfileSkeleton } from "@/features/provider/components/profile-skeleton"

export default function ProviderProfilePage() {
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
        roleContext="provider"
        onUpdateTitle={(title) => updateProfile.mutate({ title })}
      />
      <ProfileVideo videoUrl={profile?.presentation_video_url} />
      <ProfileHistory />
    </div>
  )
}
