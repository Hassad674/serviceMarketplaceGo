# Admin Panel — Marketplace B2B

## Tech Stack

- **Build tool**: Vite 7
- **UI framework**: React 19
- **Routing**: react-router-dom v7
- **Language**: TypeScript 5.9 (strict mode)
- **Styling**: Tailwind CSS 4 (CSS-first config via `@theme`)
- **Data tables**: @tanstack/react-table v8
- **Server state**: @tanstack/react-query v5
- **Charts**: Recharts
- **Icons**: lucide-react
- **Validation**: Zod
- **API types**: Generated from backend OpenAPI schema via `openapi-typescript`
- **API client**: `openapi-fetch` for type-safe HTTP calls + custom `adminApi` wrapper

## Architecture

Page-based architecture (not feature-based). The admin is a single concern with
flat routing. This is simpler and appropriate for an internal tool.

```
src/
  components/
    layouts/        # AdminLayout (sidebar + header + outlet)
    charts/         # Recharts wrapper components
    data-table/     # TanStack Table components
    ui/             # Shared UI primitives (buttons, inputs, badges)
  hooks/
    use-auth.tsx    # AuthProvider + useAuth context hook
    use-api.ts      # useApi hook wrapping fetch with auth headers
  lib/
    api-client.ts   # adminApi() — standalone fetch wrapper
    utils.ts        # cn(), formatDate(), formatCurrency()
  pages/
    Login.tsx       # Admin login form
    Dashboard.tsx   # Stats cards + chart placeholders
    Users.tsx       # User management table
    Providers.tsx   # Provider management
    Enterprises.tsx # Enterprise management
    Missions.tsx    # Mission tracking
    Projects.tsx    # Project management
    Invoices.tsx    # Invoice/payment management
    Reports.tsx     # Reports & moderation
    Settings.tsx    # Platform settings
  styles/
    globals.css     # Tailwind 4 @import + @theme tokens
  types/
    api.d.ts        # Auto-generated from OpenAPI (do NOT edit manually)
```

## Authentication

- Admin-only access. The login flow calls `/api/v1/auth/login` and checks
  the `is_admin` flag on the returned user object.
- Token stored in `localStorage` under key `admin_token`.
- All API calls include `Authorization: Bearer <token>` header.
- 401 responses automatically clear the token and redirect to `/login`.
- The `AdminLayout` component acts as a route guard: unauthenticated users
  are redirected to `/login` via `<Navigate>`.

## API Client

Two complementary approaches:

1. **`adminApi<T>(path, options)`** in `src/lib/api-client.ts` — standalone
   function for use outside React components (auth hook, utilities).
2. **`useApi()`** hook in `src/hooks/use-api.ts` — same logic wrapped in
   `useCallback` for use inside components.

Both read the token from `localStorage` and handle 401 redirects.

For type-safe API calls, generate types from the backend OpenAPI schema:

```bash
npm run generate-api
```

This runs `openapi-typescript` against the backend Swagger endpoint and writes
`src/types/api.d.ts`. These types can be used with `openapi-fetch` for
fully typed request/response handling.

## Commands

```bash
npm run dev             # Start dev server on port 5174
npm run build           # Type-check + production build
npm run preview         # Preview production build
npm run lint            # ESLint
npm run generate-api    # Generate TypeScript types from backend OpenAPI
```

## Conventions

- French labels in the UI (Tableau de bord, Utilisateurs, Prestataires, etc.)
- Currency formatted as EUR with French locale
- Dates formatted in French long format (e.g., "15 novembre 2025")
- Path alias: `@/` maps to `src/`
- No shared TypeScript packages with `web/` or `mobile/`
- Tailwind 4 CSS-first: theme tokens defined in `globals.css` `@theme` block,
  not in a `tailwind.config` file

## Key Rules

- Types in `src/types/api.d.ts` are auto-generated — never edit manually
- All API calls require admin role — the backend enforces this
- No feature isolation needed — admin is a single concern
- Keep pages self-contained: each page fetches its own data
- Use TanStack Query for server state, not local state
- Use TanStack Table for all data tables (sorting, pagination, filtering)
