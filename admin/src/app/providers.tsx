import { useState, type ReactNode } from "react"
import { QueryClientProvider } from "@tanstack/react-query"
import { createQueryClient } from "@/shared/lib/query-client"
import { AuthProvider } from "@/shared/hooks/use-auth"

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => createQueryClient())

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        {children}
      </AuthProvider>
    </QueryClientProvider>
  )
}
