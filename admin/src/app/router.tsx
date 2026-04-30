import { lazy, Suspense } from "react"
import { BrowserRouter, Routes, Route } from "react-router-dom"
import { AdminLayout } from "@/shared/components/layouts/admin-layout"
import { LoginPage } from "@/features/auth/components/login-form"
import { RouteSkeleton } from "@/shared/components/ui/route-skeleton"

// ADMIN-PERF-01 — route-level code splitting.
//
// Each authenticated page is loaded via `React.lazy()` so the bundle
// is split per route. The dashboard's recharts dependency (~500 KB)
// only ships when the user actually navigates to `/`. Pages that
// never get visited stay out of the initial bundle entirely.
//
// LoginPage stays eager because it is the entry surface — splitting
// it would add a network round-trip to every "logged out" landing.

const DashboardPage = lazy(() =>
  import("@/features/dashboard/components/dashboard-page").then((m) => ({
    default: m.DashboardPage,
  })),
)
const ModerationPage = lazy(() =>
  import("@/features/moderation/components/moderation-page").then((m) => ({
    default: m.ModerationPage,
  })),
)
const UsersPage = lazy(() =>
  import("@/features/users/components/users-page").then((m) => ({
    default: m.UsersPage,
  })),
)
const UserDetailPage = lazy(() =>
  import("@/features/users/components/user-detail-page").then((m) => ({
    default: m.UserDetailPage,
  })),
)
const ConversationsPage = lazy(() =>
  import("@/features/conversations/components/conversations-page").then((m) => ({
    default: m.ConversationsPage,
  })),
)
const ConversationDetailPage = lazy(() =>
  import(
    "@/features/conversations/components/conversation-detail-page"
  ).then((m) => ({ default: m.ConversationDetailPage })),
)
const JobsPage = lazy(() =>
  import("@/features/jobs/components/jobs-page").then((m) => ({
    default: m.JobsPage,
  })),
)
const JobDetailPage = lazy(() =>
  import("@/features/jobs/components/job-detail-page").then((m) => ({
    default: m.JobDetailPage,
  })),
)
const ReviewsPage = lazy(() =>
  import("@/features/reviews/components/reviews-page").then((m) => ({
    default: m.ReviewsPage,
  })),
)
const ReviewDetailPage = lazy(() =>
  import("@/features/reviews/components/review-detail-page").then((m) => ({
    default: m.ReviewDetailPage,
  })),
)
const MediaPage = lazy(() =>
  import("@/features/media/components/media-page").then((m) => ({
    default: m.MediaPage,
  })),
)
const MediaDetailPage = lazy(() =>
  import("@/features/media/components/media-detail-page").then((m) => ({
    default: m.MediaDetailPage,
  })),
)
const DisputesPage = lazy(() =>
  import("@/features/disputes/components/disputes-page").then((m) => ({
    default: m.DisputesPage,
  })),
)
const DisputeDetailPage = lazy(() =>
  import("@/features/disputes/components/dispute-detail-page").then((m) => ({
    default: m.DisputeDetailPage,
  })),
)
const InvoicesPage = lazy(() =>
  import("@/features/invoices/components/invoices-page").then((m) => ({
    default: m.InvoicesPage,
  })),
)

export function AppRouter() {
  return (
    <BrowserRouter>
      <Suspense fallback={<RouteSkeleton />}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />

          <Route element={<AdminLayout />}>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/moderation" element={<ModerationPage />} />
            <Route path="/users" element={<UsersPage />} />
            <Route path="/users/:id" element={<UserDetailPage />} />
            <Route path="/conversations" element={<ConversationsPage />} />
            <Route
              path="/conversations/:id"
              element={<ConversationDetailPage />}
            />
            <Route path="/jobs" element={<JobsPage />} />
            <Route path="/jobs/:id" element={<JobDetailPage />} />
            <Route path="/reviews" element={<ReviewsPage />} />
            <Route path="/reviews/:id" element={<ReviewDetailPage />} />
            <Route path="/media" element={<MediaPage />} />
            <Route path="/media/:id" element={<MediaDetailPage />} />
            <Route path="/disputes" element={<DisputesPage />} />
            <Route path="/disputes/:id" element={<DisputeDetailPage />} />
            <Route path="/invoices" element={<InvoicesPage />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}
