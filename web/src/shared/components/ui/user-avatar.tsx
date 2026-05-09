"use client"

import Image from "next/image"
import { useState } from "react"

import { Portrait } from "@/shared/components/ui/portrait"
import { useProfile } from "@/features/provider/hooks/use-profile"
import { cn } from "@/shared/lib/utils"

/**
 * UserAvatar — renders the signed-in user's uploaded photo when
 * available, otherwise falls back to the deterministic `Portrait`
 * SVG primitive keyed by role.
 *
 * The photo lives on the role-specific profile (provider /
 * freelance / agency / enterprise). For now this hook reads via
 * `useProfile()` which is the provider-side fetcher; for agency
 * and enterprise users the request 404s and we transparently fall
 * back to `Portrait`. Extending to those roles is a separate
 * follow-up — flagged in the PR.
 *
 * Renders a perfectly round image the same size as the Portrait
 * fallback so swapping is visually seamless. Object-fit: cover so
 * non-square uploads aren't distorted.
 */
type UserAvatarProps = {
	portraitId: number
	size: number
	alt?: string
	className?: string
}

export function UserAvatar({
	portraitId,
	size,
	alt = "",
	className,
}: UserAvatarProps) {
	const { data: profile } = useProfile()
	const [errored, setErrored] = useState(false)
	const photoUrl = profile?.photo_url

	if (!photoUrl || errored) {
		return (
			<Portrait
				id={portraitId}
				size={size}
				alt={alt}
				className={className}
			/>
		)
	}

	// Sized wrapper + `h-full w-full object-cover` image — caps the
	// rendered photo to a perfect square regardless of the natural
	// aspect ratio. Without the wrapper, Tailwind's preflight rule
	// `img { max-width: 100%; height: auto }` overrides the width/
	// height attributes that <Image> emits, causing portrait uploads
	// to paint at their natural aspect ratio (vertical overflow). The
	// inline `style={{ width, height }}` is required because `size`
	// is a runtime numeric prop — Tailwind cannot generate dynamic
	// class names from a number. Same pattern as
	// shared/components/ui/profile-identity-header.tsx PhotoBlock.
	return (
		<div
			className={cn(
				"relative shrink-0 overflow-hidden rounded-full",
				className,
			)}
			style={{ width: size, height: size }}
		>
			<Image
				src={photoUrl}
				alt={alt}
				width={size}
				height={size}
				className="h-full w-full object-cover"
				onError={() => setErrored(true)}
				unoptimized
			/>
		</div>
	)
}
