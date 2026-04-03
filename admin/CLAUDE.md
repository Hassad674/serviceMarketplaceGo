# Admin Panel -- Vite 7 + React 19 + Tailwind 4

## Tech Stack

- **Build tool**: Vite 7 with `@vitejs/plugin-react`
- **UI framework**: React 19
- **Language**: TypeScript 5.9 (strict mode, `erasableSyntaxOnly: true`)
- **Routing**: react-router-dom v7 (BrowserRouter, declarative routes)
- **Server state**: TanStack Query v5 (`@tanstack/react-query`)
- **Data tables**: TanStack React Table v8 (`@tanstack/react-table`)
- **Styling**: Tailwind CSS 4 (CSS-first `@theme` tokens, NO config file)
- **Charts**: Recharts
- **Icons**: Lucide React (18px inline, 20px buttons, 24px standalone)
- **Validation**: Zod
- **API types**: Generated from backend OpenAPI schema via `openapi-typescript`
- **Utilities**: clsx + tailwind-merge (`cn()` helper)

---

## Architecture

Feature-based architecture with **zero cross-feature imports**, mirroring the web app.

```
admin/src/
├── app/                    -> App shell: providers, router
│   ├── providers.tsx       -> QueryClientProvider + AuthProvider
│   └── router.tsx          -> All routes (composition point)
│
├── features/               -> Self-contained business modules
│   ├── auth/               -> Login flow
│   │   └── components/     -> LoginPage
│   ├── dashboard/          -> Dashboard stats + charts
│   │   └── components/     -> DashboardPage, StatCard
│   └── users/              -> User management (list, detail, moderation)
│       ├── api/            -> Pure async API functions
│       ├── components/     -> UsersPage, UserDetailPage, UsersFilters, usersColumns
│       ├── hooks/          -> TanStack Query hooks (useUsers, useUser, mutations)
│       └── types.ts        -> AdminUser, AdminUserListResponse, UserFilters
│
├── shared/                 -> Cross-feature reusable code
│   ├── components/
│   │   ├── data-table/     -> DataTable, DataTableToolbar, DataTablePagination
│   │   ├── layouts/        -> AdminLayout, Sidebar, Header, PageHeader
│   │   └── ui/             -> Atomic UI primitives (Button, Input, Badge, etc.)
│   ├── hooks/              -> Shared hooks (useAuth)
│   └── lib/                -> Utilities (api-client, query-client, cn/format helpers)
│
├── styles/
│   └── globals.css         -> Tailwind 4 @import + @theme tokens + animations
│
├── App.tsx                 -> Root: Providers wrapping AppRouter
└── main.tsx                -> Entry point: StrictMode + createRoot
```

---

## Feature Isolation Rules

These rules are the same as the web app and the root CLAUDE.md. They are non-negotiable.

1. **Features never import from other features.** `features/users/` must never import from `features/dashboard/`. If two features need the same logic, extract it to `shared/`.
2. **Features can import from `shared/` but never from `app/` or other features.**
3. **Composition happens in `app/router.tsx` only.** The router imports page components from different features and wires them to routes. Features remain unaware of each other.
4. **Each feature is self-contained.** Deleting `features/users/` and its routes in `router.tsx` must not cause compilation errors anywhere else.

### Feature directory structure

Every feature follows this pattern:

```
features/{name}/
├── api/            -> Pure async functions using adminApi<T>()
├── components/     -> Feature-specific React components
├── hooks/          -> TanStack Query hooks wrapping API functions
└── types.ts        -> Feature-specific TypeScript types
```

Not every feature needs all four directories. `auth/` only has `components/` because it uses `useAuth` directly. `dashboard/` only has `components/` because it uses static data. But any feature with API calls must follow the full pattern.

---

## Code Quality Standards

These limits are inherited from the root CLAUDE.md and are non-negotiable.

| Metric | Limit | Action when exceeded |
|--------|-------|----------------------|
| Lines per file | 600 max | Split into focused files by sub-domain |
| Lines per function/component | 50 max | Extract helper functions or sub-components |
| Parameters per function | 4 max | Group into a props type or options object |
| JSX nesting depth | 3 levels max | Extract sub-components at level 4 |
| Cyclomatic complexity | < 10 per function | Simplify conditionals, use lookup tables |

### TypeScript rules

- **Strict mode** enabled (`strict: true` in `tsconfig.app.json`).
- **`erasableSyntaxOnly: true`** -- no parameter properties in constructors, no enums, no namespaces. Use plain class fields and `as const` objects instead.
- **No `any`** -- use `unknown` and narrow with type guards.
- **No `as` assertions** without a comment explaining why it is safe.
- **Named exports only**: `export function ComponentName()` -- never `export default` (except `App.tsx` which is the Vite entry point).
- **Use `type` imports** for type-only imports: `import type { Foo } from "./bar"`.
- **Path alias**: `@/*` maps to `src/*` (configured in both `tsconfig.app.json` and `vite.config.ts`).

### Naming conventions

| Category | Convention | Example |
|----------|-----------|---------|
| Files | kebab-case | `users-page.tsx`, `api-client.ts` |
| Components | PascalCase export | `export function UsersPage()` |
| Hooks | camelCase, `use` prefix | `use-users.ts` / `useUsers()` |
| Types/Interfaces | PascalCase | `type AdminUser = { ... }` |
| Constants | UPPER_SNAKE_CASE | `const API_URL = "..."` |
| CSS variables | kebab-case | `--color-primary` |

---

## Authentication

The admin panel is restricted to users with `is_admin === true`. Non-admin users are rejected at login.

### Login flow

1. `LoginPage` collects email + password.
2. Calls `adminApi<LoginResponse>("/api/v1/auth/login", { method: "POST", body: { email, password }, headers: { "X-Auth-Mode": "token" } })`.
3. The `X-Auth-Mode: token` header tells the backend to return a token response instead of setting cookies.
4. Backend responds with `{ access_token, user: { id, email, is_admin } }`.
5. Frontend checks `data.user.is_admin === true`. If `false`, throws `"Acces reserve aux administrateurs"`.
6. On success, stores `access_token` in `localStorage` under key `"admin_token"`.

### Auth state

- `AuthProvider` (React Context) wraps the entire app in `app/providers.tsx`.
- `useAuth()` hook returns `{ token, isAuthenticated, login, logout }`.
- `isAuthenticated` is derived from `!!token`.
- `logout()` removes the token from `localStorage` and redirects to `/login`.

### Route protection

- `AdminLayout` component acts as the auth guard.
- It checks `isAuthenticated` from `useAuth()`. If `false`, renders `<Navigate to="/login" replace />`.
- All protected routes are children of the `<AdminLayout />` route element.
- Public routes (`/login`) sit outside the `AdminLayout` wrapper.

### Auto-redirect on 401

- The `adminApi` function checks every response for `status === 401`.
- On 401: removes `"admin_token"` from `localStorage`, redirects to `/login`, throws `ApiError`.
- This handles expired tokens transparently.

---

## API Client

### `adminApi<T>(path, options)` -- `shared/lib/api-client.ts`

A generic typed fetch wrapper. This is the only way to make API calls.

```typescript
// Signature
async function adminApi<T>(path: string, options?: RequestOptions): Promise<T>

// RequestOptions
type RequestOptions = {
  method?: string           // default: "GET"
  body?: unknown            // auto-serialized to JSON
  headers?: Record<string, string>
}
```

**Behavior:**
- Reads token from `localStorage.getItem("admin_token")`.
- Always sets `Content-Type: application/json`.
- Attaches `Authorization: Bearer <token>` when token exists.
- On 401: clears token, redirects to `/login`, throws `ApiError`.
- On any non-ok response: parses JSON error body, throws `ApiError(status, code, message)`.
- On 204 No Content: returns `undefined`.

### `ApiError` class

```typescript
class ApiError extends Error {
  status: number   // HTTP status code
  code: string     // Backend error code (e.g., "unauthorized", "validation_error")
  // message inherited from Error -- human-readable error text
}
```

Note: `ApiError` uses plain class fields, not parameter properties (required by `erasableSyntaxOnly`).

### Backend URL

- Default: `http://localhost:8083` (the project uses port 8083 locally, not 8080).
- Configurable via `VITE_API_URL` environment variable.

### API function pattern

API functions are **pure async functions** that return typed promises. They live in `features/{name}/api/` and use `adminApi<T>()` directly. They never use hooks.

```typescript
// features/users/api/users-api.ts
import { adminApi } from "@/shared/lib/api-client"
import type { AdminUserListResponse, UserFilters } from "../types"

export function listUsers(filters: UserFilters): Promise<AdminUserListResponse> {
  const params = new URLSearchParams()
  if (filters.role) params.set("role", filters.role)
  if (filters.search) params.set("search", filters.search)
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<AdminUserListResponse>(`/api/v1/admin/users${qs ? `?${qs}` : ""}`)
}
```

---

## Data Fetching Patterns

### TanStack Query for all server state

Every API interaction goes through TanStack Query. Never fetch in `useEffect`. Never store server data in `useState`.

### Query client defaults (`shared/lib/query-client.ts`)

```typescript
{
  queries: {
    staleTime: 2 * 60 * 1000,      // 2 minutes
    gcTime: 10 * 60 * 1000,         // 10 minutes garbage collection
    retry: 1,                       // single retry on failure
    refetchOnWindowFocus: false,    // no auto-refetch on tab switch
  }
}
```

### Query key pattern

All admin query keys are prefixed with `"admin"` and follow a consistent structure:

```typescript
// List queries: ["admin", "resource", filters]
["admin", "users", { role: "agency", status: "active", search: "", cursor: "" }]

// Detail queries: ["admin", "resource", id]
["admin", "users", "550e8400-e29b-41d4-a716-446655440000"]
```

Use factory functions for query keys:

```typescript
export function usersQueryKey(filters: UserFilters) {
  return ["admin", "users", filters] as const
}

export function userQueryKey(id: string) {
  return ["admin", "users", id] as const
}
```

### Hooks wrap API functions

Hooks live in `features/{name}/hooks/` and are thin wrappers:

```typescript
// Query hook
export function useUsers(filters: UserFilters) {
  return useQuery({
    queryKey: usersQueryKey(filters),
    queryFn: () => listUsers(filters),
    staleTime: 30 * 1000,  // override for lists: 30 seconds
  })
}

// Mutation hook
export function useSuspendUser(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: SuspendUserPayload) => suspendUser(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "users", id] })
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] })
    },
  })
}
```

### Loading, error, and empty states

Every data-driven component must handle all three states:

```typescript
// Loading: skeleton placeholders (never spinners)
if (isLoading) return <TableSkeleton rows={8} cols={5} />

// Error: styled error container with French message
if (error) return (
  <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
    Erreur lors du chargement des utilisateurs
  </div>
)

// Empty: EmptyState component with icon, title, description
if (users.length === 0) return (
  <EmptyState icon={Users} title="Aucun utilisateur" description="..." />
)
```

---

## Shared UI Components

All shared components live in `shared/components/ui/`. They are atomic, composable, and use the design system tokens.

### Button

**File**: `shared/components/ui/button.tsx`
**Variants**: `primary` | `secondary` | `outline` | `ghost` | `destructive`
**Sizes**: `sm` (h-8) | `md` (h-9) | `lg` (h-10)

```typescript
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "outline" | "ghost" | "destructive"
  size?: "sm" | "md" | "lg"
}
```

- Primary variant uses `bg-primary` with `hover:shadow-glow` and `active:scale-[0.98]`.
- Destructive variant also has `active:scale-[0.98]`.
- All variants have `focus-visible:ring-2 focus-visible:ring-primary/50`.
- Uses `forwardRef` for ref forwarding.

### Badge, RoleBadge, StatusBadge

**File**: `shared/components/ui/badge.tsx`

**Badge** -- generic pill with variants:
- `default`: `bg-primary/10 text-primary`
- `success`: `bg-success/10 text-success`
- `warning`: `bg-warning/10 text-warning`
- `destructive`: `bg-destructive/10 text-destructive`
- `outline`: `border border-border text-muted-foreground`

**RoleBadge** -- maps backend roles to French labels with semantic colors:
- `agency`: `bg-blue-50 text-blue-700` -- "Prestataire"
- `enterprise`: `bg-violet-50 text-violet-700` -- "Entreprise"
- `provider`: `bg-rose-50 text-rose-700` -- "Freelance"
- `admin`: `bg-slate-100 text-slate-700`

**StatusBadge** -- maps user statuses to badge variants:
- `active`: success variant -- "Actif"
- `suspended`: warning variant -- "Suspendu"
- `banned`: destructive variant -- "Banni"

### Card

**File**: `shared/components/ui/card.tsx`
**Parts**: `Card` | `CardHeader` | `CardTitle` | `CardContent` | `CardFooter`

```
Card:        rounded-xl border border-gray-100 bg-white shadow-sm
CardHeader:  px-6 pt-6
CardTitle:   text-lg font-semibold text-card-foreground
CardContent: px-6 py-4
CardFooter:  flex items-center border-t border-border px-6 py-4
```

### Input

**File**: `shared/components/ui/input.tsx`
**Props**: extends `InputHTMLAttributes` + `label?: string` + `error?: string`

- Uses `forwardRef`.
- Auto-generates `id` from `label` text when `id` prop not provided.
- Focus state: `focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20`.
- Error state: `border-destructive focus:ring-destructive/20` + red error message below.

### Select

**File**: `shared/components/ui/select.tsx`
**Props**: extends `SelectHTMLAttributes` + `label` + `options: { value, label }[]` + `placeholder`

- Uses `forwardRef`.
- Renders native `<select>` with styled wrapper.
- Same focus styling as Input.

### Textarea

**File**: `shared/components/ui/textarea.tsx`
**Props**: extends `TextareaHTMLAttributes` + `label` + `error`

- Uses `forwardRef`.
- Same pattern as Input (auto-id, error display, focus ring).

### Dialog

**File**: `shared/components/ui/dialog.tsx`
**Parts**: `Dialog` | `DialogTitle` | `DialogDescription` | `DialogFooter`

```typescript
type DialogProps = {
  open: boolean
  onClose: () => void
  children: ReactNode
}
```

- Renders portal-free modal overlay (`fixed inset-0 z-50`).
- Closes on Escape key and overlay click.
- Entry animation: `animate-fade-in` on overlay, `animate-scale-in` on content.
- Content: `max-w-md rounded-xl border bg-card p-6 shadow-lg`.
- `role="dialog"` and `aria-modal="true"` for accessibility.

### Skeleton and TableSkeleton

**File**: `shared/components/ui/skeleton.tsx`

- `Skeleton`: generic shimmer block (`animate-shimmer rounded-lg bg-muted`).
- `TableSkeleton({ rows, cols })`: renders a grid of skeleton blocks matching table layout.

### EmptyState

**File**: `shared/components/ui/empty-state.tsx`

```typescript
type EmptyStateProps = {
  icon: LucideIcon
  title: string
  description?: string
  action?: { label: string; onClick: () => void }
}
```

- Centered layout with icon in muted circle, title, description, and optional action button.

### Avatar

**File**: `shared/components/ui/avatar.tsx`
**Sizes**: `sm` (h-8) | `md` (h-10) | `lg` (h-12)

- Shows image when `src` is provided and loads successfully.
- Falls back to initials on missing `src` or image load error.
- Initials fallback: `bg-primary/10 text-primary` circle with up to 2 letters.

### DropdownMenu and DropdownItem

**File**: `shared/components/ui/dropdown-menu.tsx`

```typescript
type DropdownMenuProps = {
  trigger: ReactNode
  children: ReactNode
  align?: "left" | "right"  // default: "right"
}
```

- Opens on trigger click, closes on outside click.
- Content: `min-w-[180px] rounded-lg border bg-card shadow-lg`.
- `DropdownItem` accepts `destructive?: boolean` for red-colored dangerous actions.

### Tooltip

**File**: `shared/components/ui/tooltip.tsx`

```typescript
type TooltipProps = {
  content: string
  children: ReactNode
}
```

- Shows on mouse enter, hides on mouse leave.
- Positioned above the trigger (`bottom-full`), centered.
- Dark background: `bg-foreground text-background`.

---

## DataTable Pattern

All tabular data uses TanStack React Table v8, wrapped in shared components.

### DataTable\<T\>

**File**: `shared/components/data-table/data-table.tsx`

```typescript
type DataTableProps<T> = {
  columns: ColumnDef<T, unknown>[]
  data: T[]
  onRowClick?: (row: T) => void
}
```

- Renders a styled `<table>` with rounded container, border, and shadow.
- Header row: `border-b border-border bg-muted/50`, uppercase text.
- Body rows: `divide-y divide-border`, hover highlight, cursor pointer when `onRowClick` provided.
- Empty state: "Aucun resultat" centered in table body.

### DataTableToolbar

**File**: `shared/components/data-table/data-table-toolbar.tsx`

```typescript
type DataTableToolbarProps = {
  searchValue: string
  onSearchChange: (value: string) => void
  searchPlaceholder?: string  // default: "Rechercher..."
  children?: ReactNode        // filter dropdowns
}
```

- Search input with `Search` icon prefix.
- `children` slot for filter `Select` components placed beside the search.

### DataTablePagination

**File**: `shared/components/data-table/data-table-pagination.tsx`

```typescript
type DataTablePaginationProps = {
  hasMore: boolean
  onNextPage: () => void
  onPreviousPage?: () => void
  hasPreviousPage?: boolean
  totalCount?: number
}
```

- Cursor-based pagination (not offset-based). Displays "Precedent" / "Suivant" buttons.
- Shows total count on the left: `"{n} resultat(s)"`.

### Column definition pattern

Define columns in a separate file (`*-columns.tsx`) as a typed array:

```typescript
import type { ColumnDef } from "@tanstack/react-table"

export const usersColumns: ColumnDef<AdminUser, unknown>[] = [
  {
    id: "user",
    header: "Utilisateur",
    cell: ({ row }) => {
      const u = row.original
      return (
        <div className="flex items-center gap-3">
          <Avatar name={name} size="sm" />
          <div>
            <p className="font-medium text-foreground">{name}</p>
            <p className="text-xs text-muted-foreground">{u.email}</p>
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "role",
    header: "Role",
    cell: ({ row }) => <RoleBadge role={row.original.role} />,
  },
  // ...
]
```

---

## Layout Components

### AdminLayout

**File**: `shared/components/layouts/admin-layout.tsx`

- **Auth guard**: checks `isAuthenticated`, redirects to `/login` if false.
- **Structure**: full-height flex with Sidebar (left) + content area (right).
- Content area: Header (sticky top) + `<main>` with `<Outlet />`.
- Main area: `flex-1 overflow-y-auto bg-gray-50/50 p-6`.

### Sidebar

**File**: `shared/components/layouts/sidebar.tsx`

- Width: `w-[280px]`.
- Glass effect: `bg-white/80 backdrop-blur-xl`.
- Logo area: `h-16`, gradient text "Marketplace Admin" (`from-rose-500 to-rose-600`).
- Navigation: `NavLink` components with active state indicator.
- Active link: `bg-rose-50 text-rose-600` with a 3px rose vertical bar on the left (`absolute left-0 h-5 w-[3px] bg-rose-500 rounded-r-full`).
- Inactive link: `text-muted-foreground hover:bg-gray-50 hover:text-foreground`.
- Footer: logout button at the bottom with border-top separator.

### Header

**File**: `shared/components/layouts/header.tsx`

- Sticky: `sticky top-0 z-30`.
- Height: `h-14`.
- Glass effect: `bg-white/80 backdrop-blur-xl`.
- Contains notification bell icon button on the right.

### PageHeader

**File**: `shared/components/layouts/page-header.tsx`

```typescript
type PageHeaderProps = {
  title: string
  description?: string
  actions?: ReactNode
}
```

- Title: `text-2xl font-bold text-foreground`.
- Description: `text-sm text-muted-foreground`.
- Actions slot: right-aligned flex container for buttons.

---

## Design System

Aligned with the web app design system. Rose primary color.

### Color Tokens (oklch, defined in `globals.css` `@theme`)

| Token | Value | Usage |
|-------|-------|-------|
| `--color-primary` | `oklch(0.645 0.246 16.439)` (rose) | Primary buttons, links, active states |
| `--color-primary-foreground` | `#ffffff` | Text on primary background |
| `--color-background` | `oklch(0.985 ...)` | Page background |
| `--color-foreground` | `oklch(0.145 ...)` | Primary text |
| `--color-card` | `#ffffff` | Card backgrounds |
| `--color-card-foreground` | `oklch(0.145 ...)` | Card text |
| `--color-muted` | `oklch(0.967 ...)` | Muted backgrounds |
| `--color-muted-foreground` | `oklch(0.556 ...)` | Secondary text |
| `--color-border` | `oklch(0.913 ...)` | Borders |
| `--color-destructive` | `oklch(0.577 0.245 27.325)` | Error, danger |
| `--color-success` | `oklch(0.627 0.194 149.214)` | Success states |
| `--color-warning` | `oklch(0.769 0.188 85.29)` | Warning states |

### Radius tokens

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 6px | Small elements |
| `--radius-md` | 8px | Buttons, inputs (`rounded-lg`) |
| `--radius-lg` | 12px | Cards, containers (`rounded-xl`) |
| `--radius-xl` | 16px | Large containers (`rounded-2xl`) |

### Glass effect

```css
.glass {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
}
```

Used on: Sidebar (`bg-white/80 backdrop-blur-xl`), Header (`bg-white/80 backdrop-blur-xl`).

### Shadow glow

```css
.shadow-glow {
  box-shadow: 0 4px 24px -4px rgba(244, 63, 94, 0.4);
}
```

Used on: primary Button hover, CTA elements.

### Animations

| Class | Duration | Easing | Usage |
|-------|----------|--------|-------|
| `animate-shimmer` | 1.5s infinite | ease-in-out | Skeleton loading placeholders |
| `animate-scale-in` | 200ms | ease-out | Dialog/modal entrance |
| `animate-slide-up` | 300ms | ease-out | Content entrance |
| `animate-fade-in` | 200ms | ease-out | Overlay fade-in |

### Transitions

All interactive elements: `transition-all duration-200 ease-out`.

### Focus states

- Buttons: `focus-visible:ring-2 focus-visible:ring-primary/50`.
- Inputs: `focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20`.

### Active states

- Primary/destructive buttons: `active:scale-[0.98]`.

### Shadows

| Usage | Classes |
|-------|---------|
| Cards at rest | `shadow-sm` |
| Table container | `shadow-sm` |
| Dialogs | `shadow-lg` |
| Dropdowns | `shadow-lg` |
| Primary button hover | `shadow-glow` (custom class) |

---

## Routing

### Router structure (`app/router.tsx`)

```typescript
<BrowserRouter>
  <Routes>
    {/* Public */}
    <Route path="/login" element={<LoginPage />} />

    {/* Protected -- AdminLayout handles auth redirect */}
    <Route element={<AdminLayout />}>
      <Route path="/" element={<DashboardPage />} />
      <Route path="/users" element={<UsersPage />} />
      <Route path="/users/:id" element={<UserDetailPage />} />
    </Route>
  </Routes>
</BrowserRouter>
```

### Rules

- All routes defined in `app/router.tsx`. No route definitions elsewhere.
- Protected routes are children of `<AdminLayout />` which handles the auth check.
- `LoginPage` redirects to `/` if already authenticated (`<Navigate to="/" replace />`).
- Feature components are imported directly into `router.tsx` -- this is the composition point.

---

## Styling Conventions

- **Tailwind 4 CSS-first**: theme tokens defined in `@theme` block in `globals.css`. No `tailwind.config.ts` or `tailwind.config.js`.
- **`cn()` utility** (`shared/lib/utils.ts`): combines `clsx` + `tailwind-merge` for conditional class merging.
- **No inline `style={{}}`** -- always Tailwind utility classes.
- **No hardcoded colors** -- use semantic tokens (`bg-primary`, `text-muted-foreground`, `border-border`).
- **Variant maps** for components with multiple visual states (see Button, Badge).
- **French labels everywhere**: "Utilisateurs", "Rechercher", "Suspendre", "Tableau de bord", "Prestataire", "Entreprise", "Freelance".

### Formatting utilities (`shared/lib/utils.ts`)

```typescript
cn(...inputs: ClassValue[])             // Merge Tailwind classes
formatDate(date: string | Date)          // "15 novembre 2025" (fr-FR long)
formatCurrency(amount: number)           // "87 450,00 EUR" (fr-FR EUR)
formatRelativeDate(date: string | Date)  // "Il y a 3h", "Il y a 2j", etc.
```

---

## State Management

| What | Where | Tool |
|------|-------|------|
| Server data (API responses) | `features/*/hooks/` | TanStack Query v5 |
| Auth state | `shared/hooks/use-auth.tsx` | React Context (AuthProvider) |
| Form state | Component-local | `useState` |
| Ephemeral UI state (dialogs, filters) | Component-local | `useState` |
| Pagination cursors | Component-local | `useState` (array stack) |

- **No Redux, no Zustand** -- the admin is simpler than the web app. React Context for auth, TanStack Query for everything else.
- **No prop drilling past 2 levels** -- use composition (`children`) or extract to hooks.

---

## Accessibility Standards

Minimum compliance: **WCAG 2.1 Level AA**.

- **Keyboard navigation**: all interactive elements reachable via Tab, Enter, Escape.
- **Focus indicators**: `focus-visible:ring-2` on all focusable elements. Never remove without replacement.
- **ARIA**: Dialogs have `role="dialog"` and `aria-modal="true"`. Icon-only buttons have `aria-label`.
- **Form labels**: every `<input>` has an associated `<label>`. Auto-generated `id` from label text.
- **Color contrast**: 4.5:1 minimum for normal text.
- **Error messages**: displayed inline below the relevant input.

---

## Commands

```bash
npm run dev          # Dev server on port 5174
npm run build        # TypeScript type-check (tsc -b) + Vite production build
npm run preview      # Preview production build locally
npm run lint         # ESLint
npm run generate-api # Generate types from backend OpenAPI schema at localhost:8080
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `http://localhost:8083` | Backend API base URL |

- Never commit `.env` files -- they are gitignored.
- Access in code via `import.meta.env.VITE_API_URL`.

---

## Adding a New Feature

Follow this checklist exactly:

### 1. Create the feature directory

```bash
mkdir -p src/features/{name}/api
mkdir -p src/features/{name}/components
mkdir -p src/features/{name}/hooks
touch src/features/{name}/types.ts
```

### 2. Define types (`types.ts`)

```typescript
export type MyEntity = {
  id: string
  // ...fields matching backend response
  created_at: string
  updated_at: string
}

export type MyEntityListResponse = {
  data: MyEntity[]
  meta: {
    request_id: string
    pagination: {
      next_cursor: string
      has_more: boolean
      count: number
      total: number
    }
  }
}

export type MyEntityFilters = {
  search: string
  cursor: string
  // ...feature-specific filters
}
```

### 3. Write API functions (`api/{name}-api.ts`)

Pure async functions, no hooks. Use `adminApi<T>()`.

```typescript
import { adminApi } from "@/shared/lib/api-client"
import type { MyEntityListResponse, MyEntityFilters } from "../types"

export function listMyEntities(filters: MyEntityFilters): Promise<MyEntityListResponse> {
  const params = new URLSearchParams()
  if (filters.search) params.set("search", filters.search)
  if (filters.cursor) params.set("cursor", filters.cursor)
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<MyEntityListResponse>(`/api/v1/admin/my-entities${qs ? `?${qs}` : ""}`)
}
```

### 4. Write hooks (`hooks/use-{name}.ts`)

Thin wrappers around API functions with TanStack Query.

```typescript
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { listMyEntities } from "../api/{name}-api"
import type { MyEntityFilters } from "../types"

export function useMyEntities(filters: MyEntityFilters) {
  return useQuery({
    queryKey: ["admin", "my-entities", filters],
    queryFn: () => listMyEntities(filters),
    staleTime: 30 * 1000,
  })
}
```

### 5. Build components (`components/`)

- List page: use `DataTable`, `DataTableToolbar`, `DataTablePagination`, `EmptyState`, `TableSkeleton`.
- Detail page: use `Card`, `Badge`, `Avatar`, `Dialog` for actions.
- Filters: use `DataTableToolbar` with `Select` children.
- Columns: define in separate `*-columns.tsx` file.

### 6. Add route in `app/router.tsx`

```typescript
import { MyEntitiesPage } from "@/features/my-entity/components/my-entities-page"

// Inside <Route element={<AdminLayout />}>
<Route path="/my-entities" element={<MyEntitiesPage />} />
```

### 7. Add sidebar link in `shared/components/layouts/sidebar.tsx`

```typescript
import { MyIcon } from "lucide-react"

const navigation = [
  // ...existing items
  { to: "/my-entities", label: "Mon Entite", icon: MyIcon },
]
```

### Verification

After adding a feature, verify independence: mentally delete the feature folder and its routes/sidebar entry. The rest of the app must still compile and run.

---

## SOLID Principles (Frontend Adaptation)

- **S -- Single Responsibility**: one component = one job. A table does not fetch data. A form does not manage layout. A page composes components but contains minimal logic.
- **O -- Open/Closed**: components are extensible via props and `children`. Never modify an existing component to support a new use case -- wrap it or extend its props.
- **L -- Liskov Substitution**: any component accepting the same props interface can replace another.
- **I -- Interface Segregation**: small, focused hooks. `useUsers()` not `useEverything()`. A hook that fetches + mutates + formats is doing too much.
- **D -- Dependency Inversion**: features depend on `shared/lib/api-client.ts`, never on raw `fetch`. Server state goes through TanStack Query hooks, never raw API calls in components.

---

## Anti-Patterns (What to NEVER Do)

- **No `any` types** -- use `unknown` and narrow.
- **No `useEffect` for data fetching** -- use TanStack Query.
- **No global mutable state** -- auth is in Context, server data in TanStack Query, UI state in `useState`.
- **No cross-feature imports** -- if `features/users/` needs something from `features/dashboard/`, extract to `shared/`.
- **No inline styles** -- Tailwind classes only.
- **No full-page spinners** -- always skeleton loading matching the final layout.
- **No magic strings** -- define filter options, route paths, and query keys as constants or factory functions.
- **No prop drilling past 2 levels** -- use composition or hooks.
- **No `export default`** -- named exports only (except `App.tsx`).
- **No parameter properties** -- `erasableSyntaxOnly` forbids them. Use plain class fields.
