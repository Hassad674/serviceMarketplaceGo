"use client"

import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { useTheme } from "@/shared/hooks/use-theme"

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
            retry: 1,
            refetchOnWindowFocus: false, // avoid surprise refetches when switching tabs
          },
        },
      }),
  )

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeInitializer />
      {children}
    </QueryClientProvider>
  )
}
