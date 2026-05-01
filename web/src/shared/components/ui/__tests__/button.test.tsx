import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { createRef } from "react"
import { Button } from "../button"

describe("Button", () => {
	it("renders children", () => {
		render(<Button type="button">Click me</Button>)
		expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument()
	})

	it("forwards `type` attribute as-is so callers must set it", () => {
		const { rerender } = render(<Button type="submit">Submit</Button>)
		expect(screen.getByRole("button")).toHaveAttribute("type", "submit")

		rerender(<Button type="button">Action</Button>)
		expect(screen.getByRole("button")).toHaveAttribute("type", "button")

		rerender(<Button type="reset">Reset</Button>)
		expect(screen.getByRole("button")).toHaveAttribute("type", "reset")
	})

	it("applies primary variant by default with rose gradient", () => {
		render(<Button type="button">Primary</Button>)
		const btn = screen.getByRole("button")
		expect(btn.className).toContain("gradient-primary")
		expect(btn.className).toContain("text-white")
		// Glow on hover is the design-system signature for primary CTAs.
		expect(btn.className).toContain("hover:shadow-glow")
		expect(btn.className).toContain("active:scale-[0.98]")
	})

	it.each([
		["secondary", "bg-slate-100"],
		["outline", "border-slate-200"],
		["ghost", "hover:bg-slate-100"],
		["destructive", "bg-red-500"],
	] as const)("applies %s variant classes", (variant, marker) => {
		render(
			<Button type="button" variant={variant}>
				{variant}
			</Button>,
		)
		expect(screen.getByRole("button").className).toContain(marker)
	})

	it.each([
		["sm", "h-8"],
		["md", "h-9"],
		["lg", "h-10"],
	] as const)("applies %s size", (size, marker) => {
		render(
			<Button type="button" size={size}>
				{size}
			</Button>,
		)
		expect(screen.getByRole("button").className).toContain(marker)
	})

	it("merges custom className with variant classes", () => {
		render(
			<Button type="button" className="custom-class">
				Merged
			</Button>,
		)
		const btn = screen.getByRole("button")
		expect(btn.className).toContain("custom-class")
		expect(btn.className).toContain("gradient-primary")
	})

	it("respects disabled state and blocks pointer events", () => {
		render(
			<Button type="button" disabled>
				Disabled
			</Button>,
		)
		const btn = screen.getByRole("button")
		expect(btn).toBeDisabled()
		expect(btn.className).toContain("disabled:pointer-events-none")
		expect(btn.className).toContain("disabled:opacity-50")
	})

	it("forwards ref to the underlying button element", () => {
		const ref = createRef<HTMLButtonElement>()
		render(
			<Button type="button" ref={ref}>
				Ref
			</Button>,
		)
		expect(ref.current).toBeInstanceOf(HTMLButtonElement)
		expect(ref.current?.textContent).toBe("Ref")
	})

	it("invokes onClick when clicked", async () => {
		const onClick = vi.fn()
		const user = userEvent.setup()
		render(
			<Button type="button" onClick={onClick}>
				Action
			</Button>,
		)
		await user.click(screen.getByRole("button"))
		expect(onClick).toHaveBeenCalledTimes(1)
	})

	it("does not invoke onClick when disabled", async () => {
		const onClick = vi.fn()
		const user = userEvent.setup()
		render(
			<Button type="button" onClick={onClick} disabled>
				Disabled
			</Button>,
		)
		await user.click(screen.getByRole("button"))
		expect(onClick).not.toHaveBeenCalled()
	})

	it("is reachable via Tab navigation", async () => {
		const user = userEvent.setup()
		render(
			<>
				<a href="#a">Link</a>
				<Button type="button">Btn</Button>
			</>,
		)
		await user.tab()
		await user.tab()
		expect(screen.getByRole("button")).toHaveFocus()
	})

	it("activates on Enter and Space keys", async () => {
		const onClick = vi.fn()
		const user = userEvent.setup()
		render(
			<Button type="button" onClick={onClick}>
				Press
			</Button>,
		)
		const btn = screen.getByRole("button")
		btn.focus()
		await user.keyboard("{Enter}")
		expect(onClick).toHaveBeenCalledTimes(1)

		await user.keyboard(" ")
		expect(onClick).toHaveBeenCalledTimes(2)
	})

	it("propagates aria attributes for assistive tech", () => {
		render(
			<Button type="button" aria-label="Close" aria-pressed>
				X
			</Button>,
		)
		const btn = screen.getByRole("button")
		expect(btn).toHaveAttribute("aria-label", "Close")
		expect(btn).toHaveAttribute("aria-pressed", "true")
	})

	it("has displayName for debugging", () => {
		expect(Button.displayName).toBe("Button")
	})
})
