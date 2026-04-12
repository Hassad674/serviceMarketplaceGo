"use client"

import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { useTheme } from "@/shared/hooks/use-theme"
import { ApiError } from "@/shared/lib/api-client"

/** Map 403 error codes to user-friendly French messages for global toast. */
function getPermissionErrorMessage(error: ApiError): string | null {
  if (error.status !== 403) return null
  if (error.code === "no_organization") {
    return "Vous devez appartenir à une organisation pour effectuer cette action"
  }
  if (error.code === "permission_denied" || error.code === "forbidden") {
    return "Permission refusée — vous n'avez pas accès à cette fonctionnalité"
  }
  return null
}

function ThemeInitializer() {
  const { theme } = useTheme()

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark")
  }, [theme])

  return null
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 2 * 60 * 1000, // 2 minutes — prevents refetching on every component mount
            gcTime: 10 * 60 * 1000, // 10 minutes — keep unused cache entries longer
            retry: (failureCount, error) => {
              // Never retry 403 — permission errors are not transient
              if (error instanceof ApiError && error.status === 403) return false
              return failureCount < 1
            },
            refetchOnWindowFocus: false, // avoid surprise refetches when switching tabs
          },
        },
        queryCache: new QueryCache({
          onError: (error) => {
            if (error instanceof ApiError) {
              const message = getPermissionErrorMessage(error)
              if (message) {
                toast.error(message)
              }
            }
          },
        }),
        mutationCache: new MutationCache({
          onError: (error) => {
            if (error instanceof ApiError) {
              const message = getPermissionErrorMessage(error)
              if (message) {
                toast.error(message)
              }
            }
          },
        }),
      }),
  )

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeInitializer />
      {children}
    </QueryClientProvider>
  )
}
