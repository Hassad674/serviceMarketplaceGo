import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useState } from "react"
import { Modal } from "../modal"

describe("Modal", () => {
	it("renders nothing when closed", () => {
		const { container } = render(
			<Modal open={false} onClose={() => {}} title="Hidden">
				<p>body</p>
			</Modal>,
		)
		expect(container.querySelector("[role='dialog']")).toBeNull()
		expect(screen.queryByText("body")).toBeNull()
	})

	it("renders into a portal on document.body when open", () => {
		const { baseElement } = render(
			<Modal open onClose={() => {}} title="Visible">
				<p>body</p>
			</Modal>,
		)
		// Portal mounts onto document.body, so the dialog is *not* a child
		// of the test container — but it IS within baseElement (which is
		// document.body by default in @testing-library/react).
		const dialog = baseElement.querySelector("[role='dialog']")
		expect(dialog).not.toBeNull()
		expect(screen.getByText("body")).toBeInTheDocument()
	})

	it("sets aria-modal=true and aria-labelledby on the dialog", () => {
		render(
			<Modal open onClose={() => {}} title="Confirmation">
				<p>body</p>
			</Modal>,
		)
		const dialog = screen.getByRole("dialog")
		expect(dialog).toHaveAttribute("aria-modal", "true")
		const labelledBy = dialog.getAttribute("aria-labelledby")
		expect(labelledBy).toBeTruthy()
		const title = document.getElementById(labelledBy ?? "")
		expect(title?.textContent).toBe("Confirmation")
	})

	it("renders title + close button when showHeader is true (default)", () => {
		render(
			<Modal open onClose={() => {}} title="Title">
				<p>body</p>
			</Modal>,
		)
		expect(screen.getByText("Title")).toBeInTheDocument()
		expect(screen.getByRole("button", { name: "Fermer" })).toBeInTheDocument()
	})

	it("hides the header when showHeader is false", () => {
		render(
			<Modal open onClose={() => {}} title="Hidden" showHeader={false}>
				<p>body</p>
			</Modal>,
		)
		expect(screen.queryByText("Hidden")).toBeNull()
		expect(screen.queryByRole("button", { name: "Fermer" })).toBeNull()
	})

	it("calls onClose when the close button is clicked", async () => {
		const onClose = vi.fn()
		const user = userEvent.setup()
		render(
			<Modal open onClose={onClose} title="Title">
				<p>body</p>
			</Modal>,
		)
		await user.click(screen.getByRole("button", { name: "Fermer" }))
		expect(onClose).toHaveBeenCalledTimes(1)
	})

	it("calls onClose when Escape is pressed", () => {
		const onClose = vi.fn()
		render(
			<Modal open onClose={onClose} title="Title">
				<p>body</p>
			</Modal>,
		)
		fireEvent.keyDown(document, { key: "Escape" })
		expect(onClose).toHaveBeenCalledTimes(1)
	})

	it("does not bind Escape when closed", () => {
		const onClose = vi.fn()
		render(
			<Modal open={false} onClose={onClose} title="Title">
				<p>body</p>
			</Modal>,
		)
		fireEvent.keyDown(document, { key: "Escape" })
		expect(onClose).not.toHaveBeenCalled()
	})

	it("calls onClose when the backdrop is clicked", async () => {
		const onClose = vi.fn()
		const user = userEvent.setup()
		const { baseElement } = render(
			<Modal open onClose={onClose} title="Title">
				<p>body</p>
			</Modal>,
		)
		const backdrop = baseElement.querySelector(
			"[role='presentation']",
		) as HTMLElement
		await user.click(backdrop)
		expect(onClose).toHaveBeenCalled()
	})

	it("does not call onClose when clicking inside the dialog content", async () => {
		const onClose = vi.fn()
		const user = userEvent.setup()
		render(
			<Modal open onClose={onClose} title="Title">
				<button type="button">inside</button>
			</Modal>,
		)
		await user.click(screen.getByRole("button", { name: "inside" }))
		expect(onClose).not.toHaveBeenCalled()
	})

	it("focuses the first focusable element on open", () => {
		render(
			<Modal open onClose={() => {}} title="Title">
				<button type="button">First</button>
				<button type="button">Second</button>
			</Modal>,
		)
		// On mount, focus moves to the first focusable inside the dialog.
		// The header close button comes before the children, so it gets
		// focus first.
		const closeBtn = screen.getByRole("button", { name: "Fermer" })
		expect(document.activeElement).toBe(closeBtn)
	})

	it("traps focus inside the dialog with Tab + Shift+Tab", () => {
		render(
			<Modal open onClose={() => {}} title="Title" showHeader={false}>
				<button type="button">First</button>
				<button type="button">Last</button>
			</Modal>,
		)
		const first = screen.getByRole("button", { name: "First" })
		const last = screen.getByRole("button", { name: "Last" })

		last.focus()
		expect(document.activeElement).toBe(last)
		// Tab from last should wrap to first.
		fireEvent.keyDown(document, { key: "Tab" })
		expect(document.activeElement).toBe(first)

		first.focus()
		// Shift+Tab from first should wrap to last.
		fireEvent.keyDown(document, { key: "Tab", shiftKey: true })
		expect(document.activeElement).toBe(last)
	})

	it("applies custom maxWidthClassName to the dialog panel", () => {
		render(
			<Modal
				open
				onClose={() => {}}
				title="Title"
				maxWidthClassName="max-w-2xl"
			>
				<p>body</p>
			</Modal>,
		)
		expect(screen.getByRole("dialog").className).toContain("max-w-2xl")
	})

	it("uses default maxWidthClassName when not provided", () => {
		render(
			<Modal open onClose={() => {}} title="Title">
				<p>body</p>
			</Modal>,
		)
		expect(screen.getByRole("dialog").className).toContain("max-w-md")
	})

	it("can be closed and reopened cleanly (no leaked listeners)", () => {
		function Harness() {
			const [open, setOpen] = useState(true)
			return (
				<>
					<button type="button" onClick={() => setOpen((v) => !v)}>
						toggle
					</button>
					<Modal open={open} onClose={() => setOpen(false)} title="Title">
						<p>body</p>
					</Modal>
				</>
			)
		}
		const { rerender } = render(<Harness />)
		expect(screen.getByText("body")).toBeInTheDocument()
		fireEvent.click(screen.getByText("toggle"))
		// Forced rerender to flush effects.
		rerender(<Harness />)
		// Toggle a second time — should reopen without throwing.
		fireEvent.click(screen.getByText("toggle"))
		expect(screen.queryByText("body")).toBeInTheDocument()
	})
})
