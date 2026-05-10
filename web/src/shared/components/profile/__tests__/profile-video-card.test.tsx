import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ProfileVideoCard } from "../profile-video-card"

interface RenderArgs {
  videoUrl: string
  readOnly?: boolean
  showWhenEmpty?: boolean
  withUploadAction?: boolean
}

function renderCard(args: RenderArgs) {
  const { videoUrl, readOnly = false, showWhenEmpty, withUploadAction } = args
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProfileVideoCard
        videoUrl={videoUrl}
        labels={{
          title: "Presentation Video",
          emptyLabel: "No presentation video",
          emptyDescription: "Add a video to present your activity",
        }}
        readOnly={readOnly}
        showWhenEmpty={showWhenEmpty}
        actions={
          withUploadAction
            ? { onUpload: async () => {}, uploading: false }
            : undefined
        }
      />
    </NextIntlClientProvider>,
  )
}

describe("ProfileVideoCard — public-profile empty-state regression", () => {
  it("returns null when read-only and the video URL is empty (default)", () => {
    const { container } = renderCard({ videoUrl: "", readOnly: true })
    expect(container.firstChild).toBeNull()
  })

  it("renders the empty-state when the owner is editing (read-only=false) so the upload CTA stays visible", () => {
    renderCard({ videoUrl: "", readOnly: false, withUploadAction: true })
    // The empty-state surfaces the empty label and an "Add a video"
    // button so the owner can still upload from /profile.
    expect(screen.getByText("No presentation video")).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: /Add a video/i }),
    ).toBeInTheDocument()
  })

  it("still renders the empty-state when read-only with showWhenEmpty=true (legacy opt-in)", () => {
    renderCard({ videoUrl: "", readOnly: true, showWhenEmpty: true })
    expect(screen.getByText("No presentation video")).toBeInTheDocument()
  })

  it("renders the embedded <video> tag when a URL is set, regardless of mode", () => {
    const { container } = renderCard({
      videoUrl: "https://media.example.test/video.mp4",
      readOnly: true,
    })
    const video = container.querySelector("video")
    expect(video).not.toBeNull()
    expect(video?.getAttribute("src")).toBe("https://media.example.test/video.mp4")
  })

  it("renders the embedded <video> tag in editable mode when a URL is set", () => {
    const { container } = renderCard({
      videoUrl: "https://media.example.test/video.mp4",
      readOnly: false,
      withUploadAction: true,
    })
    expect(container.querySelector("video")).not.toBeNull()
  })
})
