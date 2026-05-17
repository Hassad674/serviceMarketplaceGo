import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { BrandLogo } from "../brand-logo"

describe("BrandLogo", () => {
	it("renders the full lockup by default with an accessible name", () => {
		render(<BrandLogo />)
		const logo = screen.getByRole("img", { name: "DesignedTrust Services" })
		expect(logo).toBeInTheDocument()
		expect(logo.tagName.toLowerCase()).toBe("svg")
		// Full lockup carries the wordmark text.
		expect(logo).toHaveTextContent("Designed")
		expect(logo).toHaveTextContent("Trust")
		expect(logo).toHaveTextContent("SERVICES")
	})

	it("renders the mark-only variant without the wordmark text", () => {
		render(<BrandLogo variant="mark" />)
		const logo = screen.getByRole("img", { name: "DesignedTrust Services" })
		expect(logo).toBeInTheDocument()
		expect(logo.tagName.toLowerCase()).toBe("svg")
		expect(logo).not.toHaveTextContent("Designed")
		expect(logo).not.toHaveTextContent("SERVICES")
	})

	it("forwards className for sizing", () => {
		render(<BrandLogo className="h-7 w-auto" />)
		const logo = screen.getByRole("img", { name: "DesignedTrust Services" })
		expect(logo).toHaveClass("h-7", "w-auto")
	})

	it("uses the brand orange in the pictogram for both variants", () => {
		const { container: full } = render(<BrandLogo />)
		expect(full.querySelector('path[fill="#FF7A1F"]')).not.toBeNull()

		const { container: mark } = render(<BrandLogo variant="mark" />)
		expect(mark.querySelector('path[fill="#FF7A1F"]')).not.toBeNull()
	})
})
