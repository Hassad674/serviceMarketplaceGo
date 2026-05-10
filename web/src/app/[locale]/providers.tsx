"use client"

import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { CookieBanner } from "@/shared/components/analytics/cookie-banner"
import { GoogleAnalyticsProvider } from "@/shared/components/analytics/google-analytics-provider"
import { PostHogProvider } from "@/shared/components/analytics/posthog-provider"
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

/**
 * Reads `meta.suppressGlobalErrorToast` from a query/mutation's options.
 * When `true` the global QueryCache / MutationCache toast handler skips
 * the error — the local consumer is expected to surface its own message.
 *
 * This is the surgical escape hatch for flows where a 403 from a
 * background refetch (or a chained invalidation) is EXPECTED and would
 * otherwise spuriously alarm the user — e.g. the role-permissions editor,
 * which fans out a `["session"]` invalidation immediately after a
 * successful save and used to surface a false "permission denied" toast
 * as the post-success refetch raced the now-stale permission snapshot.
 */
function shouldSuppressGlobalErrorToast(meta: unknown): boolean {
  if (typeof meta !== "object" || meta === null) return false
  const flag = (meta as { suppressGlobalErrorToast?: unknown })
    .suppressGlobalErrorToast
  return flag === true
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
            // PERF-FIX-W-IDLE-CPU: a single tab running ~10 polling
            // hooks at once can hit the global IP rate limit (100
            // req/min). When the backend returns 429 to *every*
            // endpoint, retrying the same query just piles on. Treat
            // every 4xx as terminal — only network errors and 5xx
            // get the (single) retry. Mutations never retry.
            retry: (failureCount, error) => {
              if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
                return false
              }
              return failureCount < 1
            },
            // Cap retry delay so a 5xx blip + recovery does not fire
            // 12 retries inside the 60 s rate-limit window.
            retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 30_000),
            refetchOnWindowFocus: false, // avoid surprise refetches when switching tabs
            refetchOnReconnect: false, // dev: WS reconnect storms drive this — opt in per-hook only
          },
          mutations: {
            // Replays for create-style POSTs are owned by the
            // Idempotency-Key middleware on the backend, not by
            // client-side timer storms.
            retry: false,
          },
        },
        queryCache: new QueryCache({
          onError: (error, query) => {
            if (!(error instanceof ApiError)) return
            if (shouldSuppressGlobalErrorToast(query.meta)) return
            const message = getPermissionErrorMessage(error)
            if (message) {
              toast.error(message)
            }
          },
        }),
        mutationCache: new MutationCache({
          onError: (error, _vars, _onMutateResult, mutation) => {
            if (!(error instanceof ApiError)) return
            if (shouldSuppressGlobalErrorToast(mutation.meta)) return
            const message = getPermissionErrorMessage(error)
            if (message) {
              toast.error(message)
            }
          },
        }),
      }),
  )

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeInitializer />
      {/*
        PostHogProvider must live INSIDE QueryClientProvider so it can
        consume useSession() to identify the logged-in user. It renders
        nothing — pure side-effect on the SDK lifecycle. The banner is
        rendered last so it floats above page content without forcing
        anyone to wrap their layouts.
      */}
      <PostHogProvider />
      {/*
        GoogleAnalyticsProvider mounts the gtag.js script via
        `@next/third-parties/google`. It renders nothing until BOTH
        NEXT_PUBLIC_GA_MEASUREMENT_ID is set AND the user opted in
        through the cookie banner. RGPD-compatible: no script loaded
        before consent.
      */}
      <GoogleAnalyticsProvider />
      {children}
      <CookieBanner />
    </QueryClientProvider>
  )
}
