import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ProfileVideo } from "../profile-video"

// Mock the UploadModal to avoid portal/complexity issues in unit tests
vi.mock("@/shared/components/upload-modal", () => ({
  UploadModal: ({
    open,
    title,
    onClose,
  }: {
    open: boolean
    title: string
    onClose: () => void
  }) =>
    open ? (
      <div data-testid="upload-modal" aria-label={title}>
        <button onClick={onClose}>Close modal</button>
      </div>
    ) : null,
}))

function renderProfileVideo(
  props: Partial<Parameters<typeof ProfileVideo>[0]> = {},
) {
  const defaultProps = {
    videoUrl: undefined as string | undefined,
    onUploadVideo: vi.fn().mockResolvedValue(undefined),
    uploadingVideo: false,
    ...props,
  }

  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProfileVideo {...defaultProps} />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ProfileVideo", () => {
  it("renders empty state when no video", () => {
    renderProfileVideo({ videoUrl: undefined })

    expect(screen.getByText(messages.profile.noVideo)).toBeInTheDocument()
    expect(
      screen.getByText(messages.profile.addVideoDesc),
    ).toBeInTheDocument()
  })

  it("renders video player when video URL provided", () => {
    renderProfileVideo({ videoUrl: "https://example.com/video.mp4" })

    const video = screen.getByLabelText(messages.profile.videoTitle)
    expect(video).toBeInTheDocument()
    expect(video.tagName).toBe("VIDEO")
    expect(video).toHaveAttribute("src", "https://example.com/video.mp4")
  })

  it("shows add video button when no video", () => {
    renderProfileVideo({ videoUrl: undefined })

    expect(
      screen.getByRole("button", { name: messages.profile.addVideo }),
    ).toBeInTheDocument()
  })

  it("shows change video button when video exists", () => {
    renderProfileVideo({ videoUrl: "https://example.com/video.mp4" })

    expect(
      screen.getByRole("button", { name: messages.profile.changeVideo }),
    ).toBeInTheDocument()
  })

  it("shows remove button when video exists and onDeleteVideo provided", () => {
    const mockDelete = vi.fn()
    renderProfileVideo({
      videoUrl: "https://example.com/video.mp4",
      onDeleteVideo: mockDelete,
    })

    expect(
      screen.getByRole("button", { name: new RegExp(messages.profile.removeVideo) }),
    ).toBeInTheDocument()
  })

  it("does not show remove button when onDeleteVideo is not provided", () => {
    renderProfileVideo({
      videoUrl: "https://example.com/video.mp4",
      onDeleteVideo: undefined,
    })

    expect(
      screen.queryByRole("button", { name: new RegExp(messages.profile.removeVideo) }),
    ).not.toBeInTheDocument()
  })

  it("opens upload modal when add video button is clicked", async () => {
    const user = userEvent.setup()
    renderProfileVideo({ videoUrl: undefined })

    const addButton = screen.getByRole("button", {
      name: messages.profile.addVideo,
    })
    await user.click(addButton)

    expect(screen.getByTestId("upload-modal")).toBeInTheDocument()
  })

  it("opens upload modal when change video button is clicked", async () => {
    const user = userEvent.setup()
    renderProfileVideo({ videoUrl: "https://example.com/video.mp4" })

    const changeButton = screen.getByRole("button", {
      name: messages.profile.changeVideo,
    })
    await user.click(changeButton)

    expect(screen.getByTestId("upload-modal")).toBeInTheDocument()
  })

  it("calls onDeleteVideo when remove button is clicked", async () => {
    const mockDelete = vi.fn()
    const user = userEvent.setup()
    renderProfileVideo({
      videoUrl: "https://example.com/video.mp4",
      onDeleteVideo: mockDelete,
    })

    const removeButton = screen.getByRole("button", {
      name: new RegExp(messages.profile.removeVideo),
    })
    await user.click(removeButton)

    expect(mockDelete).toHaveBeenCalledOnce()
  })

  it("renders section heading", () => {
    renderProfileVideo({ videoUrl: undefined })

    expect(
      screen.getByRole("heading", { name: messages.profile.videoTitle }),
    ).toBeInTheDocument()
  })

  it("accepts custom title", () => {
    renderProfileVideo({ videoUrl: undefined, title: "My Custom Video" })

    expect(
      screen.getByRole("heading", { name: "My Custom Video" }),
    ).toBeInTheDocument()
  })
})
