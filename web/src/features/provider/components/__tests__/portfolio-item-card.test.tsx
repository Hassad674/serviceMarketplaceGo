/**
 * Pinning test for PortfolioItemCard's portfolio cover migration to
 * next/image fill mode. The cover renders inside an aspect-[4/5]
 * absolute container, so the image gets `fill` + a responsive
 * `sizes` hint.
 */
import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { PortfolioItemCard } from "../portfolio-item-card"
import type { PortfolioItem, PortfolioMedia } from "../../api/portfolio-api"

const messages = {
  portfolio: {
    edit: "Edit",
    delete: "Delete",
  },
}

vi.mock("next/image", () => ({
  default: ({
    src,
    alt,
    fill,
    sizes,
    className,
  }: {
    src: string
    alt: string
    fill?: boolean
    sizes?: string
    className?: string
  }) => (
    // eslint-disable-next-line @next/next/no-img-element -- test mock substituting next/image
    <img
      src={src}
      alt={alt}
      data-fill={fill ? "true" : undefined}
      data-sizes={sizes}
      className={className}
    />
  ),
}))

function makeMedia(overrides: Partial<PortfolioMedia> = {}): PortfolioMedia {
  return {
    id: "m-1",
    media_url: "https://cdn.example.com/cover.jpg",
    media_type: "image",
    thumbnail_url: "",
    position: 0,
    created_at: "",
    ...overrides,
  }
}

function makeItem(media: PortfolioMedia[]): PortfolioItem {
  return {
    id: "item-1",
    organization_id: "org-1",
    title: "Demo project",
    description: "",
    link_url: "",
    cover_url: "",
    position: 0,
    media,
    created_at: "",
    updated_at: "",
  }
}

function renderCard(item: PortfolioItem) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <PortfolioItemCard item={item} readOnly />
    </NextIntlClientProvider>,
  )
}

describe("PortfolioItemCard cover", () => {
  it("renders an Image with fill and a responsive sizes hint when cover is an image", () => {
    renderCard(makeItem([makeMedia({ media_type: "image" })]))
    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://cdn.example.com/cover.jpg")
    expect(img).toHaveAttribute("alt", "Demo project")
    expect(img).toHaveAttribute("data-fill", "true")
    expect(img.getAttribute("data-sizes")).toContain("vw")
    expect(img.className).toMatch(/object-cover/)
  })

  it("renders an Image of the thumbnail when the cover is a video with a custom thumbnail", () => {
    renderCard(
      makeItem([
        makeMedia({
          media_type: "video",
          media_url: "https://cdn.example.com/clip.mp4",
          thumbnail_url: "https://cdn.example.com/clip-thumb.jpg",
        }),
      ]),
    )
    const img = screen.getByRole("img")
    expect(img).toHaveAttribute("src", "https://cdn.example.com/clip-thumb.jpg")
    expect(img).toHaveAttribute("data-fill", "true")
  })

  it("does not render an Image when there is no media", () => {
    renderCard(makeItem([]))
    expect(screen.queryByRole("img")).toBeNull()
  })
})
