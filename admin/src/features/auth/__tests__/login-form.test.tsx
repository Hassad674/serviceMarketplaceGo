import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { LoginPage } from "../components/login-form"

const mockLogin = vi.fn()
const mockNavigate = vi.fn()
let mockIsAuthenticated = false

vi.mock("@/shared/hooks/use-auth", () => ({
  useAuth: () => ({
    isAuthenticated: mockIsAuthenticated,
    login: mockLogin,
    logout: vi.fn(),
    user: null,
  }),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>(
    "react-router-dom",
  )
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

beforeEach(() => {
  vi.clearAllMocks()
  mockIsAuthenticated = false
})

describe("admin LoginPage", () => {
  it("renders the form with email + password + submit", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    expect(screen.getByText("Administration")).toBeInTheDocument()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/mot de passe/i)).toBeInTheDocument()
  })

  it("redirects authenticated users to /", () => {
    mockIsAuthenticated = true
    const { container } = render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    // The Navigate component renders nothing visible
    expect(container.querySelector("h1")).toBeNull()
  })

  it("calls login with the typed credentials on submit", async () => {
    mockLogin.mockResolvedValue({})
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "admin@x.com" },
    })
    fireEvent.change(screen.getByLabelText(/mot de passe/i), {
      target: { value: "secret" },
    })
    fireEvent.click(screen.getByText("Se connecter"))

    await waitFor(() =>
      expect(mockLogin).toHaveBeenCalledWith("admin@x.com", "secret"),
    )
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith("/"))
  })

  it("shows the error message when login throws", async () => {
    mockLogin.mockRejectedValue(new Error("Invalid credentials"))
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "x@x.com" },
    })
    fireEvent.change(screen.getByLabelText(/mot de passe/i), {
      target: { value: "bad" },
    })
    fireEvent.click(screen.getByText("Se connecter"))
    await waitFor(() =>
      expect(screen.getByText("Invalid credentials")).toBeInTheDocument(),
    )
  })

  it("uses generic error if a non-Error is thrown", async () => {
    mockLogin.mockRejectedValue("string error")
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "x@x.com" },
    })
    fireEvent.change(screen.getByLabelText(/mot de passe/i), {
      target: { value: "bad" },
    })
    fireEvent.click(screen.getByText("Se connecter"))
    await waitFor(() =>
      expect(screen.getByText("Erreur de connexion")).toBeInTheDocument(),
    )
  })

  it("disables button while loading", async () => {
    let resolveLogin: (() => void) | undefined
    mockLogin.mockReturnValue(new Promise<void>((r) => (resolveLogin = r)))
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "x@x.com" },
    })
    fireEvent.change(screen.getByLabelText(/mot de passe/i), {
      target: { value: "p" },
    })
    fireEvent.click(screen.getByText("Se connecter"))
    await waitFor(() =>
      expect(screen.getByText("Connexion...")).toBeInTheDocument(),
    )
    resolveLogin?.()
  })
})
