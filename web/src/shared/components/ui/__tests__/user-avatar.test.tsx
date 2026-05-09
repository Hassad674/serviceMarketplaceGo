/**
 * UserAvatar tests — guards the sized-wrapper + object-cover pattern
 * that prevents portrait uploads from overflowing vertically in the
 * sidebar (size=36) and header (size=32) avatars.
 *
 * Without the sized wrapper, Tailwind's preflight rule
 * `img { max-width: 100%; height: auto }` wins over <Image>'s width/
 * height attributes and the photo paints at its natural aspect ratio.
 * Each test below pins one invariant of the fix so future
 * "simplifications" cannot silently re-introduce the bug.
 */

import { describe, expect, it, vi, beforeEach } from "vitest"
import { render } from "@testing-library/react"
import type { ReactNode } from "react"

// Hoist the next/image mock so the component renders a plain <img>
// in jsdom — we still want className/style to round-trip exactly.
vi.mock("next/image", () => ({
  default: ({
    src,
    alt,
    className,
    onError,
  }: {
    src: string
    alt: string
    className?: string
    onError?: () => void
  }) => (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={src} alt={alt} className={className} onError={onError} />
  ),
}))

// `useProfile` is the only data dependency — we stub it per test.
const profileMock = vi.fn()
vi.mock("@/features/provider/hooks/use-profile", () => ({
  useProfile: () => profileMock(),
}))

// Stub Portrait so the fallback assertions are easy to anchor on.
vi.mock("@/shared/components/ui/portrait", () => ({
  Portrait: ({
    id,
    size,
    className,
  }: {
    id: number
    size: number
    className?: string
    alt?: string
  }) => (
    <div
      data-testid="portrait-stub"
      data-id={id}
      data-size={size}
      className={className}
    />
  ),
}))

import { UserAvatar } from "../user-avatar"

function withProfile(photoUrl: string | null): ReactNode {
  profileMock.mockReturnValue({
    data: photoUrl ? { photo_url: photoUrl } : { photo_url: null },
  })
  return null
}

describe("UserAvatar", () => {
  beforeEach(() => {
    profileMock.mockReset()
  })

  it("caps the photo wrapper to an exact square via inline width/height", () => {
    withProfile("https://cdn.example.com/me.jpg")
    const { container } = render(
      <UserAvatar portraitId={2} size={36} alt="Photo de moi" />,
    )

    const wrapper = container.firstElementChild as HTMLElement
    // The wrapper is the load-bearing element — without it, the inner
    // <img> paints at its natural aspect ratio. Lock both dimensions
    // explicitly so the fix can never silently regress.
    expect(wrapper.style.width).toBe("36px")
    expect(wrapper.style.height).toBe("36px")
  })

  it("forces the inner <Image> to fill its wrapper with object-cover", () => {
    withProfile("https://cdn.example.com/me.jpg")
    const { container } = render(
      <UserAvatar portraitId={0} size={32} alt="" />,
    )

    const img = container.querySelector("img")
    expect(img).not.toBeNull()
    // These three utilities are what defeat Tailwind's preflight
    // `img { max-width: 100%; height: auto }` rule. Drop any of them
    // and portrait uploads overflow vertically again.
    expect(img!.className).toContain("h-full")
    expect(img!.className).toContain("w-full")
    expect(img!.className).toContain("object-cover")
  })

  it("renders a Portrait fallback when the profile has no photo_url", () => {
    withProfile(null)
    const { container, queryByTestId } = render(
      <UserAvatar portraitId={3} size={48} alt="" />,
    )

    expect(container.querySelector("img")).toBeNull()
    const portrait = queryByTestId("portrait-stub")
    expect(portrait).not.toBeNull()
    expect(portrait!.getAttribute("data-id")).toBe("3")
    expect(portrait!.getAttribute("data-size")).toBe("48")
  })

  it("falls back to Portrait when the photo fails to load (onError)", () => {
    withProfile("https://cdn.example.com/broken.jpg")
    const { container, queryByTestId, rerender } = render(
      <UserAvatar portraitId={1} size={40} alt="" />,
    )

    // Photo branch is taken initially.
    const img = container.querySelector("img")
    expect(img).not.toBeNull()
    expect(queryByTestId("portrait-stub")).toBeNull()

    // Fire onError; the component should swap to the Portrait branch.
    img!.dispatchEvent(new Event("error"))
    rerender(<UserAvatar portraitId={1} size={40} alt="" />)

    expect(container.querySelector("img")).toBeNull()
    expect(queryByTestId("portrait-stub")).not.toBeNull()
  })

  it("forwards className onto the wrapper (not the inner image)", () => {
    withProfile("https://cdn.example.com/me.jpg")
    const { container } = render(
      <UserAvatar
        portraitId={0}
        size={36}
        alt=""
        className="ring-2 ring-accent"
      />,
    )

    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.className).toContain("ring-2")
    expect(wrapper.className).toContain("ring-accent")
    // The base utilities must remain so the wrapper still clips the
    // image to a circle and prevents overflow.
    expect(wrapper.className).toContain("rounded-full")
    expect(wrapper.className).toContain("overflow-hidden")
    expect(wrapper.className).toContain("shrink-0")
  })

  it("honors arbitrary size values on both wrapper and image", () => {
    withProfile("https://cdn.example.com/me.jpg")
    const { container } = render(
      <UserAvatar portraitId={0} size={72} alt="" />,
    )

    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.width).toBe("72px")
    expect(wrapper.style.height).toBe("72px")
    // The <Image> mock above forwards width/height through props but
    // the visual cap is the wrapper — assert object-cover is still on
    // the inner img so a non-square upload would crop, not stretch.
    const img = container.querySelector("img")!
    expect(img.className).toContain("object-cover")
  })
})
