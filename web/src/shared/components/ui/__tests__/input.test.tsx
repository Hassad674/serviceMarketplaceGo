import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { createRef } from "react"
import { Input } from "../input"

describe("Input", () => {
	it("renders an input element", () => {
		render(<Input aria-label="Email" type="email" />)
		expect(screen.getByLabelText("Email")).toBeInTheDocument()
	})

	it("associates label with input via htmlFor + id", () => {
		render(<Input label="Email address" type="email" />)
		const input = screen.getByLabelText("Email address")
		expect(input).toBeInTheDocument()
		const label = screen.getByText("Email address")
		expect(label).toHaveAttribute("for", input.id)
		expect(input.id).toBeTruthy()
	})

	it("respects an explicitly passed id for the input/label pair", () => {
		render(<Input id="custom-id" label="Custom" />)
		expect(screen.getByLabelText("Custom").id).toBe("custom-id")
		expect(screen.getByText("Custom")).toHaveAttribute("for", "custom-id")
	})

	it("renders an error message tied via aria-describedby", () => {
		render(<Input label="Email" error="Adresse invalide" />)
		const input = screen.getByLabelText("Email")
		const error = screen.getByRole("alert")
		expect(error).toHaveTextContent("Adresse invalide")
		expect(input).toHaveAttribute("aria-invalid", "true")
		expect(input.getAttribute("aria-describedby")).toContain(error.id)
	})

	it("switches to the error visual state when error prop is set", () => {
		render(<Input aria-label="Email" error="bad" />)
		const input = screen.getByLabelText("Email")
		expect(input.className).toContain("border-red-500")
	})

	it("renders hint text when no error is present", () => {
		render(<Input label="Mot de passe" hint="8 caracteres minimum" />)
		const input = screen.getByLabelText("Mot de passe")
		expect(screen.getByText("8 caracteres minimum")).toBeInTheDocument()
		expect(input.getAttribute("aria-describedby")).toContain(`${input.id}-hint`)
	})

	it("hides the hint when an error is present", () => {
		render(
			<Input label="Pwd" hint="should be ignored" error="too short" />,
		)
		expect(screen.queryByText("should be ignored")).not.toBeInTheDocument()
		expect(screen.getByText("too short")).toBeInTheDocument()
	})

	it.each([
		["sm", "h-8"],
		["md", "h-10"],
		["lg", "h-12"],
	] as const)("applies %s size", (size, marker) => {
		render(<Input aria-label="x" size={size} />)
		expect(screen.getByLabelText("x").className).toContain(marker)
	})

	it("has rose focus ring + border on default state", () => {
		render(<Input aria-label="Email" />)
		const input = screen.getByLabelText("Email")
		expect(input.className).toContain("focus:border-rose-500")
		expect(input.className).toContain("focus:ring-rose-500/10")
		expect(input.className).toContain("border-slate-200")
	})

	it("disabled state blocks interaction", async () => {
		const onChange = vi.fn()
		const user = userEvent.setup()
		render(<Input aria-label="Email" disabled onChange={onChange} />)
		const input = screen.getByLabelText("Email")
		expect(input).toBeDisabled()
		await user.type(input, "hello")
		expect(onChange).not.toHaveBeenCalled()
	})

	it("forwards ref to the native input", () => {
		const ref = createRef<HTMLInputElement>()
		render(<Input ref={ref} aria-label="Ref" />)
		expect(ref.current).toBeInstanceOf(HTMLInputElement)
	})

	it("accepts and forwards arbitrary input attributes", () => {
		render(
			<Input
				aria-label="Email"
				type="email"
				name="email"
				placeholder="you@example.com"
				autoComplete="email"
				required
			/>,
		)
		const input = screen.getByLabelText("Email")
		expect(input).toHaveAttribute("type", "email")
		expect(input).toHaveAttribute("name", "email")
		expect(input).toHaveAttribute("placeholder", "you@example.com")
		expect(input).toHaveAttribute("autocomplete", "email")
		expect(input).toBeRequired()
	})

	it("merges aria-describedby from caller with the auto-generated ones", () => {
		render(
			<Input
				label="Email"
				error="bad"
				aria-describedby="external-hint"
			/>,
		)
		const input = screen.getByLabelText("Email")
		const describedBy = input.getAttribute("aria-describedby") ?? ""
		expect(describedBy).toContain("external-hint")
		expect(describedBy).toContain(`${input.id}-error`)
	})

	it("respects explicit aria-invalid override", () => {
		render(<Input aria-label="Email" aria-invalid={false} error="ignored" />)
		expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "false")
	})

	it("merges custom className with variant classes", () => {
		render(<Input aria-label="Email" className="custom-class" />)
		const input = screen.getByLabelText("Email")
		expect(input.className).toContain("custom-class")
		expect(input.className).toContain("rounded-lg")
	})

	it("merges wrapperClassName onto the outer container", () => {
		const { container } = render(
			<Input aria-label="Email" wrapperClassName="wrapper-x" />,
		)
		const wrapper = container.firstElementChild
		expect(wrapper?.className).toContain("wrapper-x")
		expect(wrapper?.className).toContain("flex-col")
	})

	it("renders inline (no wrapper) when no label/error/hint/wrapperClassName", () => {
		// Critical for migrations from raw <input> inside custom layouts:
		// a wrapping <div> would break flex/grid positioning.
		const { container } = render(<Input aria-label="Email" />)
		const root = container.firstElementChild
		expect(root?.tagName).toBe("INPUT")
	})

	it("renders the wrapper as soon as any wrapper-only prop is set", () => {
		const cases = [
			<Input key="label" label="L" />,
			<Input key="error" aria-label="E" error="bad" />,
			<Input key="hint" aria-label="H" hint="h" />,
			<Input key="wrapper" aria-label="W" wrapperClassName="x" />,
		]
		for (const ui of cases) {
			const { container } = render(ui)
			expect(container.firstElementChild?.tagName).toBe("DIV")
		}
	})

	it("invokes onChange and reflects user input", async () => {
		const onChange = vi.fn()
		const user = userEvent.setup()
		render(<Input aria-label="Email" onChange={onChange} />)
		const input = screen.getByLabelText("Email") as HTMLInputElement
		await user.type(input, "abc")
		expect(onChange).toHaveBeenCalled()
		expect(input.value).toBe("abc")
	})

	it("has displayName for debugging", () => {
		expect(Input.displayName).toBe("Input")
	})
})
