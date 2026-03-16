import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
} from "react"
import { adminApi } from "@/lib/api-client.ts"

interface AuthState {
  token: string | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | null>(null)

interface LoginResponse {
  token: string
  user: { id: string; email: string; is_admin: boolean }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(
    () => localStorage.getItem("admin_token"),
  )

  const login = useCallback(async (email: string, password: string) => {
    const data = await adminApi<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: { email, password },
    })

    if (!data.user.is_admin) {
      throw new Error("Accès réservé aux administrateurs")
    }

    localStorage.setItem("admin_token", data.token)
    setToken(data.token)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem("admin_token")
    setToken(null)
    window.location.href = "/login"
  }, [])

  return (
    <AuthContext.Provider
      value={{ token, isAuthenticated: !!token, login, logout }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
