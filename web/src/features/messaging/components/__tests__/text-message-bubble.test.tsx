import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, act } from "@testing-library/react"
import { TextMessageBubble } from "../text-message-bubble"
import type { Message } from "../../types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("../message-status-icon", () => ({
  MessageStatusIcon: ({ status }: { status: string }) => (
    <span data-testid={`status-${status}`} />
  ),
}))

vi.mock("../file-message", () => ({
  FileMessage: () => <span data-testid="file-message" />,
}))

vi.mock("../voice-message", () => ({
  VoiceMessage: () => <span data-testid="voice-message" />,
}))

vi.mock("../message-context-menu", () => ({
  MessageContextMenu: ({
    onEdit,
    onDelete,
    onReply,
    onReport,
  }: {
    onEdit?: () => void
    onDelete?: () => void
    onReply?: () => void
    onReport?: () => void
  }) => {
    if (!onEdit && !onDelete && !onReply && !onReport) return null
    return (
      <span data-testid="context-menu">
        {onReply && <button onClick={onReply} data-testid="reply-btn" />}
        {onEdit && <button onClick={onEdit} data-testid="edit-btn" />}
        {onDelete && <button onClick={onDelete} data-testid="delete-btn" />}
        {onReport && <button onClick={onReport} data-testid="report-btn" />}
      </span>
    )
  },
}))

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "msg-1",
    conversation_id: "conv-1",
    sender_id: "u-1",
    content: "hello",
    type: "text",
    metadata: null,
    seq: 1,
    status: "sent",
    edited_at: null,
    deleted_at: null,
    created_at: "2026-04-01T10:00:00Z",
    ...overrides,
  }
}

function defaultActions() {
  return {
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onReply: vi.fn(),
    onReport: vi.fn(),
  }
}

describe("TextMessageBubble — base rendering", () => {
  it("renders the text content", () => {
    render(
      <TextMessageBubble
        message={makeMessage({ content: "Hello world" })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText("Hello world")).toBeInTheDocument()
  })

  it("renders the time string", () => {
    render(
      <TextMessageBubble
        message={makeMessage({ created_at: "2026-04-01T15:30:00Z" })}
        isOwn
        actions={defaultActions()}
      />,
    )
    // Time format is HH:MM in local time. Just assert that SOME 4-char
    // colon-separated value is on the screen.
    const textNodes = Array.from(document.querySelectorAll("p"))
    expect(textNodes.some((p) => /\d{2}:\d{2}/.test(p.textContent ?? ""))).toBe(true)
  })

  it("shows status icon when isOwn=true", () => {
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("status-sent")).toBeInTheDocument()
  })

  it("hides status icon when isOwn=false", () => {
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn={false}
        actions={defaultActions()}
      />,
    )
    expect(screen.queryByTestId("status-sent")).not.toBeInTheDocument()
  })

  it("uses flex-row-reverse layout for own messages", () => {
    const { container } = render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(container.querySelector(".flex-row-reverse")).not.toBeNull()
  })

  it("uses flex-row layout for other-user messages", () => {
    const { container } = render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn={false}
        actions={defaultActions()}
      />,
    )
    const groupDiv = container.querySelector(".group")
    expect(groupDiv?.className).toContain("flex-row")
    expect(groupDiv?.className).not.toContain("flex-row-reverse")
  })

  it("renders the edited label when edited_at is present", () => {
    render(
      <TextMessageBubble
        message={makeMessage({ edited_at: "2026-04-01T11:00:00Z" })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText(/messageEdited/)).toBeInTheDocument()
  })
})

describe("TextMessageBubble — file/voice variants", () => {
  it("renders FileMessage for file type with filename metadata", () => {
    render(
      <TextMessageBubble
        message={makeMessage({
          type: "file",
          metadata: { filename: "x.pdf" } as never,
        })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("file-message")).toBeInTheDocument()
  })

  it("renders VoiceMessage for voice type with duration metadata", () => {
    render(
      <TextMessageBubble
        message={makeMessage({
          type: "voice",
          metadata: { duration: 12 } as never,
        })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("voice-message")).toBeInTheDocument()
  })
})

describe("TextMessageBubble — reply preview", () => {
  it("renders a reply preview block when reply_to is set", () => {
    render(
      <TextMessageBubble
        message={makeMessage({
          reply_to: {
            id: "r1",
            sender_id: "x",
            content: "Original message text",
            type: "text",
          },
        })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText(/Original message text/)).toBeInTheDocument()
  })

  it("truncates long reply content with ellipsis", () => {
    const long = "x".repeat(120)
    render(
      <TextMessageBubble
        message={makeMessage({
          reply_to: {
            id: "r1",
            sender_id: "x",
            content: long,
            type: "text",
          },
        })}
        isOwn
        actions={defaultActions()}
      />,
    )
    // The truncate call slices to 80 + "..."
    expect(screen.getByText(/^x{80}\.{3}$/)).toBeInTheDocument()
  })

  it("renders a placeholder when the reply content is empty", () => {
    render(
      <TextMessageBubble
        message={makeMessage({
          reply_to: {
            id: "r1",
            sender_id: "x",
            content: "",
            type: "text",
          },
        })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText("...")).toBeInTheDocument()
  })
})

describe("TextMessageBubble — context menu", () => {
  it("hides the context menu for temp- ids", () => {
    render(
      <TextMessageBubble
        message={makeMessage({ id: "temp-123" })}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.queryByTestId("context-menu")).not.toBeInTheDocument()
  })

  it("shows reply/edit/delete on own messages", () => {
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("context-menu")).toBeInTheDocument()
    expect(screen.getByTestId("reply-btn")).toBeInTheDocument()
    expect(screen.getByTestId("edit-btn")).toBeInTheDocument()
    expect(screen.getByTestId("delete-btn")).toBeInTheDocument()
  })

  it("hides edit/delete on other-user messages", () => {
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn={false}
        actions={{ ...defaultActions(), onReport: vi.fn() }}
      />,
    )
    expect(screen.queryByTestId("edit-btn")).not.toBeInTheDocument()
    expect(screen.queryByTestId("delete-btn")).not.toBeInTheDocument()
    expect(screen.getByTestId("report-btn")).toBeInTheDocument()
  })

  it("hides report on own messages even when onReport is supplied", () => {
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn
        actions={{ ...defaultActions(), onReport: vi.fn() }}
      />,
    )
    expect(screen.queryByTestId("report-btn")).not.toBeInTheDocument()
  })
})

describe("TextMessageBubble — edit flow", () => {
  it("enters edit mode and shows the input + save/cancel buttons", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    const input = screen.getByDisplayValue("old") as HTMLInputElement
    expect(input).toBeInTheDocument()
    expect(screen.getByText("save")).toBeInTheDocument()
    expect(screen.getByText("cancel")).toBeInTheDocument()
  })

  it("saves the edit on Enter when content changed", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-x", content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    const input = screen.getByDisplayValue("old")
    fireEvent.change(input, { target: { value: "new content" } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(actions.onEdit).toHaveBeenCalledWith("m-x", "new content")
  })

  it("does not call onEdit when the trimmed value is empty", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    const input = screen.getByDisplayValue("old")
    fireEvent.change(input, { target: { value: "   " } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(actions.onEdit).not.toHaveBeenCalled()
  })

  it("does not call onEdit when the trimmed value equals the original", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ content: "same" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    const input = screen.getByDisplayValue("same")
    fireEvent.change(input, { target: { value: "  same  " } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(actions.onEdit).not.toHaveBeenCalled()
  })

  it("dismisses edit mode on Escape", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    const input = screen.getByDisplayValue("old")
    fireEvent.keyDown(input, { key: "Escape" })
    expect(screen.queryByDisplayValue("old")).not.toBeInTheDocument()
  })

  it("dismisses edit mode via the cancel button without calling onEdit", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    fireEvent.click(screen.getByText("cancel"))
    expect(actions.onEdit).not.toHaveBeenCalled()
  })

  it("saves via the save button", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-y", content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("edit-btn"))
    const input = screen.getByDisplayValue("old")
    fireEvent.change(input, { target: { value: "renamed" } })
    fireEvent.click(screen.getByText("save"))
    expect(actions.onEdit).toHaveBeenCalledWith("m-y", "renamed")
  })
})

describe("TextMessageBubble — reply / delete / report buttons", () => {
  it("calls onReply with the message", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("reply-btn"))
    expect(actions.onReply).toHaveBeenCalledWith(
      expect.objectContaining({ id: "msg-1" }),
    )
  })

  it("calls onDelete with the message id", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-del" })}
        isOwn
        actions={actions}
      />,
    )
    fireEvent.click(screen.getByTestId("delete-btn"))
    expect(actions.onDelete).toHaveBeenCalledWith("m-del")
  })

  it("calls onReport with the message id when present and isOwn=false", () => {
    const onReport = vi.fn()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-rep" })}
        isOwn={false}
        actions={{ ...defaultActions(), onReport }}
      />,
    )
    fireEvent.click(screen.getByTestId("report-btn"))
    expect(onReport).toHaveBeenCalledWith("m-rep")
  })
})

describe("TextMessageBubble — long-press mobile menu", () => {
  it("opens via the contextmenu event and dismisses via overlay click", () => {
    render(
      <TextMessageBubble
        message={makeMessage()}
        isOwn
        actions={defaultActions()}
      />,
    )
    const bubble = document.querySelector(".rounded-2xl") as HTMLElement
    fireEvent.contextMenu(bubble)
    expect(screen.getByText("reply")).toBeInTheDocument()
    expect(screen.getByText("editMessage")).toBeInTheDocument()
    expect(screen.getByText("deleteMessage")).toBeInTheDocument()
  })

  it("opens via long-press touch and dismisses on touchEnd before timeout", async () => {
    vi.useFakeTimers()
    try {
      render(
        <TextMessageBubble
          message={makeMessage()}
          isOwn
          actions={defaultActions()}
        />,
      )
      const bubble = document.querySelector(".rounded-2xl") as HTMLElement
      fireEvent.touchStart(bubble)
      // 500ms long-press threshold
      await act(async () => {
        vi.advanceTimersByTime(500)
      })
      expect(screen.getByText("reply")).toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it("touchEnd cancels a pending long-press", async () => {
    vi.useFakeTimers()
    try {
      render(
        <TextMessageBubble
          message={makeMessage()}
          isOwn
          actions={defaultActions()}
        />,
      )
      const bubble = document.querySelector(".rounded-2xl") as HTMLElement
      fireEvent.touchStart(bubble)
      fireEvent.touchEnd(bubble)
      await act(async () => {
        vi.advanceTimersByTime(600)
      })
      // Mobile menu should NOT be open
      expect(screen.queryByText("editMessage")).not.toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it("touchMove cancels a pending long-press", async () => {
    vi.useFakeTimers()
    try {
      render(
        <TextMessageBubble
          message={makeMessage()}
          isOwn
          actions={defaultActions()}
        />,
      )
      const bubble = document.querySelector(".rounded-2xl") as HTMLElement
      fireEvent.touchStart(bubble)
      fireEvent.touchMove(bubble)
      await act(async () => {
        vi.advanceTimersByTime(600)
      })
      expect(screen.queryByText("editMessage")).not.toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })

  it("dispatches reply from the mobile menu", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-mob" })}
        isOwn
        actions={actions}
      />,
    )
    const bubble = document.querySelector(".rounded-2xl") as HTMLElement
    fireEvent.contextMenu(bubble)
    fireEvent.click(screen.getByText("reply"))
    expect(actions.onReply).toHaveBeenCalled()
  })

  it("dispatches delete from the mobile menu (own message)", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-mob" })}
        isOwn
        actions={actions}
      />,
    )
    const bubble = document.querySelector(".rounded-2xl") as HTMLElement
    fireEvent.contextMenu(bubble)
    fireEvent.click(screen.getByText("deleteMessage"))
    expect(actions.onDelete).toHaveBeenCalledWith("m-mob")
  })

  it("dispatches edit from the mobile menu (own message)", () => {
    const actions = defaultActions()
    render(
      <TextMessageBubble
        message={makeMessage({ content: "old" })}
        isOwn
        actions={actions}
      />,
    )
    const bubble = document.querySelector(".rounded-2xl") as HTMLElement
    fireEvent.contextMenu(bubble)
    fireEvent.click(screen.getByText("editMessage"))
    // Menu closes; edit input shows
    expect(screen.getByDisplayValue("old")).toBeInTheDocument()
  })

  it("dispatches report from the mobile menu when onReport is set", () => {
    const onReport = vi.fn()
    render(
      <TextMessageBubble
        message={makeMessage({ id: "m-rep2" })}
        isOwn={false}
        actions={{ ...defaultActions(), onReport }}
      />,
    )
    const bubble = document.querySelector(".rounded-2xl") as HTMLElement
    fireEvent.contextMenu(bubble)
    fireEvent.click(screen.getByText("report"))
    expect(onReport).toHaveBeenCalledWith("m-rep2")
  })
})
