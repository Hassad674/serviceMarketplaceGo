import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { createRef } from "react"
import { Select } from "../select"

const colors = [
	{ value: "red", label: "Rouge" },
	{ value: "blue", label: "Bleu" },
	{ value: "green", label: "Vert", disabled: true },
] as const

describe("Select", () => {
	it("renders a native select element", () => {
		render(<Select aria-label="Color" options={colors} />)
		const sel = screen.getByLabelText("Color")
		expect(sel.tagName).toBe("SELECT")
	})

	it("associates label with select via htmlFor + id", () => {
		render(<Select label="Couleur" options={colors} />)
		const sel = screen.getByLabelText("Couleur")
		const label = screen.getByText("Couleur")
		expect(label).toHaveAttribute("for", sel.id)
	})

	it("respects an explicitly passed id", () => {
		render(<Select id="custom" label="Couleur" options={colors} />)
		expect(screen.getByLabelText("Couleur").id).toBe("custom")
	})

	it("renders all options from the options array", () => {
		render(<Select aria-label="Color" options={colors} />)
		expect(screen.getByRole("option", { name: "Rouge" })).toBeInTheDocument()
		expect(screen.getByRole("option", { name: "Bleu" })).toBeInTheDocument()
		const vert = screen.getByRole("option", { name: "Vert" })
		expect(vert).toBeDisabled()
	})

	it("renders a placeholder option when provided", () => {
		render(
			<Select aria-label="Color" options={colors} placeholder="Choisir..." />,
		)
		const placeholderOption = screen.getByRole("option", {
			name: "Choisir...",
		}) as HTMLOptionElement
		expect(placeholderOption.value).toBe("")
		expect(placeholderOption).toBeDisabled()
	})

	it("does not disable the placeholder when a value is set (controlled)", () => {
		render(
			<Select
				aria-label="Color"
				options={colors}
				placeholder="Choisir..."
				value="red"
				onChange={() => {}}
			/>,
		)
		// When a value is selected, the placeholder is no longer disabled
		// (the user is allowed to clear back to it via reset).
		const placeholderOption = screen.getByRole("option", {
			name: "Choisir...",
		}) as HTMLOptionElement
		expect(placeholderOption).not.toBeDisabled()
	})

	it("accepts children in lieu of options array", () => {
		render(
			<Select aria-label="Color">
				<option value="a">Alpha</option>
				<option value="b">Beta</option>
			</Select>,
		)
		expect(screen.getByRole("option", { name: "Alpha" })).toBeInTheDocument()
		expect(screen.getByRole("option", { name: "Beta" })).toBeInTheDocument()
	})

	it("renders an error message tied via aria-describedby", () => {
		render(<Select label="Color" options={colors} error="Choix obligatoire" />)
		const sel = screen.getByLabelText("Color")
		const error = screen.getByRole("alert")
		expect(error).toHaveTextContent("Choix obligatoire")
		expect(sel).toHaveAttribute("aria-invalid", "true")
		expect(sel.getAttribute("aria-describedby")).toContain(error.id)
	})

	it("switches to error visual state when error is set", () => {
		render(<Select aria-label="Color" options={colors} error="bad" />)
		expect(screen.getByLabelText("Color").className).toContain("border-red-500")
	})

	it("renders hint text when no error is set", () => {
		render(<Select label="Color" options={colors} hint="Choisir une couleur" />)
		expect(screen.getByText("Choisir une couleur")).toBeInTheDocument()
	})

	it.each([
		["sm", "h-8"],
		["md", "h-10"],
		["lg", "h-12"],
	] as const)("applies %s size", (size, marker) => {
		render(<Select aria-label="x" options={colors} size={size} />)
		expect(screen.getByLabelText("x").className).toContain(marker)
	})

	it("disabled state blocks user interaction", async () => {
		const onChange = vi.fn()
		const user = userEvent.setup()
		render(
			<Select
				aria-label="Color"
				options={colors}
				onChange={onChange}
				disabled
			/>,
		)
		const sel = screen.getByLabelText("Color") as HTMLSelectElement
		expect(sel).toBeDisabled()
		await user.selectOptions(sel, "red").catch(() => {})
		expect(onChange).not.toHaveBeenCalled()
	})

	it("invokes onChange when a different option is selected", async () => {
		const onChange = vi.fn()
		const user = userEvent.setup()
		render(
			<Select aria-label="Color" options={colors} onChange={onChange} />,
		)
		await user.selectOptions(screen.getByLabelText("Color"), "blue")
		expect(onChange).toHaveBeenCalledTimes(1)
	})

	it("forwards ref to the underlying select", () => {
		const ref = createRef<HTMLSelectElement>()
		render(<Select ref={ref} aria-label="Color" options={colors} />)
		expect(ref.current).toBeInstanceOf(HTMLSelectElement)
	})

	it("merges custom className with variant classes", () => {
		render(
			<Select
				aria-label="Color"
				options={colors}
				className="custom-class"
			/>,
		)
		expect(screen.getByLabelText("Color").className).toContain("custom-class")
	})

	it("renders a decorative chevron with pointer-events-none", () => {
		const { container } = render(
			<Select aria-label="Color" options={colors} />,
		)
		const chevron = container.querySelector("svg")
		expect(chevron).toBeInTheDocument()
		expect(chevron?.getAttribute("aria-hidden")).toBe("true")
		expect(chevron?.className.baseVal).toContain("pointer-events-none")
	})

	it("has displayName for debugging", () => {
		expect(Select.displayName).toBe("Select")
	})

	it("renders without the outer flex-col wrapper when no label/error/hint", () => {
		// Critical for migrations from raw <select> inside custom layouts:
		// the chevron's `relative` parent is still rendered (it has to be,
		// for absolute positioning), but the OUTER `flex flex-col gap-1`
		// wrapper is dropped so flex/grid layouts at call-sites don't
		// gain an unexpected child.
		const { container } = render(<Select aria-label="Color" options={colors} />)
		const root = container.firstElementChild as HTMLElement
		expect(root.tagName).toBe("DIV")
		expect(root.className).toContain("relative")
		expect(root.className).not.toContain("flex-col")
	})

	it("does render the outer wrapper when a wrapper-only prop is set", () => {
		const cases = [
			<Select key="label" label="L" options={colors} />,
			<Select key="error" aria-label="E" options={colors} error="bad" />,
			<Select key="hint" aria-label="H" options={colors} hint="h" />,
			<Select
				key="wrapper"
				aria-label="W"
				options={colors}
				wrapperClassName="x"
			/>,
		]
		for (const ui of cases) {
			const { container } = render(ui)
			const root = container.firstElementChild as HTMLElement
			expect(root.className).toContain("flex-col")
		}
	})
})
