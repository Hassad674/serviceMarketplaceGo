import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { createRef } from "react"
import {
	Card,
	CardHeader,
	CardTitle,
	CardDescription,
	CardContent,
	CardFooter,
} from "../card"

describe("Card", () => {
	it("renders children inside a div surface", () => {
		render(
			<Card data-testid="root">
				<span>content</span>
			</Card>,
		)
		const root = screen.getByTestId("root")
		expect(root.tagName).toBe("DIV")
		expect(root.textContent).toBe("content")
	})

	it("applies default variant + padding tokens", () => {
		render(<Card data-testid="root">x</Card>)
		const root = screen.getByTestId("root")
		expect(root.className).toContain("rounded-2xl")
		expect(root.className).toContain("border-slate-100")
		expect(root.className).toContain("shadow-sm")
		expect(root.className).toContain("p-6")
	})

	it("applies interactive variant with hover affordances", () => {
		render(
			<Card data-testid="root" variant="interactive">
				x
			</Card>,
		)
		const root = screen.getByTestId("root")
		expect(root.className).toContain("hover:shadow-md")
		expect(root.className).toContain("hover:border-rose-200")
		expect(root.className).toContain("hover:-translate-y-0.5")
		expect(root.className).toContain("cursor-pointer")
	})

	it.each([
		["none", null],
		["sm", "p-4"],
		["md", "p-6"],
		["lg", "p-8"],
	] as const)("applies padding %s", (padding, marker) => {
		render(
			<Card data-testid="root" padding={padding}>
				x
			</Card>,
		)
		const root = screen.getByTestId("root")
		if (marker) {
			expect(root.className).toContain(marker)
		} else {
			expect(root.className).not.toMatch(/\bp-(4|6|8)\b/)
		}
	})

	it("merges custom className over variant classes", () => {
		render(
			<Card data-testid="root" className="custom-x">
				x
			</Card>,
		)
		expect(screen.getByTestId("root").className).toContain("custom-x")
	})

	it("forwards ref to the underlying div", () => {
		const ref = createRef<HTMLDivElement>()
		render(<Card ref={ref}>x</Card>)
		expect(ref.current).toBeInstanceOf(HTMLDivElement)
	})

	it("propagates HTML attributes (role, aria, data)", () => {
		render(
			<Card data-testid="root" role="region" aria-label="Stats">
				x
			</Card>,
		)
		const root = screen.getByTestId("root")
		expect(root).toHaveAttribute("role", "region")
		expect(root).toHaveAttribute("aria-label", "Stats")
	})
})

describe("Card subcomponents", () => {
	it("CardHeader has header padding", () => {
		render(<CardHeader data-testid="h">x</CardHeader>)
		expect(screen.getByTestId("h").className).toContain("px-6")
		expect(screen.getByTestId("h").className).toContain("pt-6")
	})

	it("CardTitle renders an h3 with title typography", () => {
		render(<CardTitle data-testid="t">Title</CardTitle>)
		const t = screen.getByTestId("t")
		expect(t.tagName).toBe("H3")
		expect(t.className).toContain("text-lg")
		expect(t.className).toContain("font-semibold")
	})

	it("CardDescription renders a p with muted color", () => {
		render(<CardDescription data-testid="d">d</CardDescription>)
		const d = screen.getByTestId("d")
		expect(d.tagName).toBe("P")
		expect(d.className).toContain("text-slate-500")
	})

	it("CardContent has content padding", () => {
		render(<CardContent data-testid="c">x</CardContent>)
		expect(screen.getByTestId("c").className).toContain("px-6")
		expect(screen.getByTestId("c").className).toContain("py-4")
	})

	it("CardFooter has footer border + padding", () => {
		render(<CardFooter data-testid="f">x</CardFooter>)
		const f = screen.getByTestId("f")
		expect(f.className).toContain("border-t")
		expect(f.className).toContain("border-slate-100")
		expect(f.className).toContain("px-6")
		expect(f.className).toContain("py-4")
	})

	it("subcomponents merge custom className", () => {
		render(
			<>
				<CardHeader data-testid="h" className="custom-h" />
				<CardTitle data-testid="t" className="custom-t" />
				<CardDescription data-testid="d" className="custom-d" />
				<CardContent data-testid="c" className="custom-c" />
				<CardFooter data-testid="f" className="custom-f" />
			</>,
		)
		expect(screen.getByTestId("h").className).toContain("custom-h")
		expect(screen.getByTestId("t").className).toContain("custom-t")
		expect(screen.getByTestId("d").className).toContain("custom-d")
		expect(screen.getByTestId("c").className).toContain("custom-c")
		expect(screen.getByTestId("f").className).toContain("custom-f")
	})

	it("subcomponents forward refs", () => {
		const headerRef = createRef<HTMLDivElement>()
		const titleRef = createRef<HTMLHeadingElement>()
		const descRef = createRef<HTMLParagraphElement>()
		const contentRef = createRef<HTMLDivElement>()
		const footerRef = createRef<HTMLDivElement>()
		render(
			<>
				<CardHeader ref={headerRef} />
				<CardTitle ref={titleRef} />
				<CardDescription ref={descRef} />
				<CardContent ref={contentRef} />
				<CardFooter ref={footerRef} />
			</>,
		)
		expect(headerRef.current).toBeInstanceOf(HTMLDivElement)
		expect(titleRef.current).toBeInstanceOf(HTMLHeadingElement)
		expect(descRef.current).toBeInstanceOf(HTMLParagraphElement)
		expect(contentRef.current).toBeInstanceOf(HTMLDivElement)
		expect(footerRef.current).toBeInstanceOf(HTMLDivElement)
	})
})

describe("Card display names", () => {
	it("each component has a debug-friendly displayName", () => {
		expect(Card.displayName).toBe("Card")
		expect(CardHeader.displayName).toBe("CardHeader")
		expect(CardTitle.displayName).toBe("CardTitle")
		expect(CardDescription.displayName).toBe("CardDescription")
		expect(CardContent.displayName).toBe("CardContent")
		expect(CardFooter.displayName).toBe("CardFooter")
	})
})
