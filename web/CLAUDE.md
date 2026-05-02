# Web — Next.js 16 + React 19 + Tailwind 4

## Tech Stack

- **Framework**: Next.js 16 (App Router, Server Components by default)
- **UI**: React 19, Tailwind CSS 4 (CSS-first config via `@theme`)
- **Language**: TypeScript 5 (strict mode)
- **API types**: Generated from backend OpenAPI schema via `openapi-typescript`
- **API client**: `openapi-fetch` for type-safe HTTP calls; thin `apiClient` wrapper in `src/shared/lib/api-client.ts`
- **Server state**: TanStack Query v5 for caching, refetching, optimistic updates
- **Client state**: Zustand v5, used only for auth (no other global client state)
- **Forms**: react-hook-form v7 + zod + @hookform/resolvers
- **Icons**: lucide-react
- **Styling utilities**: clsx + tailwind-merge (`cn()` in `src/shared/lib/utils.ts`), class-variance-authority for component variants
- **Testing**: Vitest for unit tests, Playwright for E2E

## Architecture

Feature-based architecture with **zero cross-feature imports**.

```
web/src/
├── app/                -> Routing only (thin pages, layouts, metadata)
│   ├── (auth)/         -> /login, /register
│   ├── (dashboard)/    -> Role-specific dashboard routes
│   │   ├── agency/
│   │   ├── enterprise/
│   │   └── provider/
│   └── (public)/       -> /agencies, /freelances, /projects
├── config/             -> App-level configuration (site.ts)
├── features/           -> Self-contained business modules
│   ├── auth/           -> Login, register, role selection
│   ├── enterprise/     -> Enterprise-specific logic
│   ├── invoice/        -> Invoicing and billing
│   ├── messaging/      -> Real-time messaging
│   ├── mission/        -> Browse missions, apply, manage
│   ├── notification/   -> Notification management
│   ├── project/        -> Project lifecycle
│   ├── provider/       -> Provider-specific logic
│   ├── referral/       -> Referral / apporteur d'affaires
│   ├── review/         -> Reviews and ratings
│   └── team/           -> Team management
├── shared/             -> Cross-feature reusable code
│   ├── components/
│   │   ├── layouts/    -> Sidebar, Header, DashboardShell
│   │   └── ui/         -> Atomic UI components (shadcn/ui style)
│   ├── hooks/          -> Shared hooks (useAuth, etc.)
│   ├── lib/            -> Utilities (cn, apiClient, formatters)
│   └── types/          -> Generated API types (api.d.ts)
├── styles/             -> globals.css with Tailwind 4 @theme tokens
└── middleware.ts       -> Auth route protection
```

---

## Code Quality Standards

These limits are non-negotiable. They prevent complexity from accumulating.

| Metric | Limit | Rationale |
|--------|-------|-----------|
| File length | 600 lines max | Split into modules if exceeded |
| Function/component body | 50 lines max | Extract helpers or sub-components |
| Component props | 4 max | Use composition or a single config object for more |
| JSX nesting depth | 3 levels max | Extract sub-components at level 4 |
| `any` type | Forbidden | Use `unknown` + type narrowing |
| `@ts-ignore` / `@ts-expect-error` | Forbidden without comment | Must include a one-line explanation of why it is necessary |

---

## SOLID Principles (Frontend Adaptation)

- **S — Single Responsibility**: One component = one job. A form does not manage layout. A table does not fetch data.
- **O — Open/Closed**: Components are extensible via props and children. Never modify an existing component's internals to support a new use case — wrap it or extend its props.
- **L — Liskov Substitution**: Any component accepting the same props interface can replace another. `<PrimaryButton>` and `<DangerButton>` are interchangeable if they both satisfy `ButtonProps`.
- **I — Interface Segregation**: Small, focused hooks. `useMessages()` not `useEverything()`. A hook that fetches + mutates + formats + subscribes is doing too much.
- **D — Dependency Inversion**: Features depend on `shared/lib/api-client.ts`, never on raw `fetch`. Server state goes through TanStack Query hooks, never raw API calls in components.

---

## STUPID Anti-Patterns (What to Avoid)

- **No prop drilling past 2 levels** — Use composition (children), context, or Zustand stores.
- **No god components** — If a component's JSX exceeds 200 lines, it needs to be decomposed.
- **No business logic in components** — Extract to custom hooks or utility functions. Components render; hooks compute.
- **No inline styles** — Tailwind utility classes only. Use `cn()` for conditional classes.
- **No magic strings** — Define constants (`ROUTE_PATHS`, `QUERY_KEYS`, `ERROR_CODES`). String literals should appear once.
- **No premature abstraction** — Wait until you have 3 similar cases before abstracting. Duplication is cheaper than the wrong abstraction.

---

## Component Rules

### Server vs Client Components
- **Server Components by default** — only add `"use client"` when the component needs: `useState`, `useEffect`, event handlers (`onClick`, `onChange`), or browser APIs (`window`, `localStorage`).
- When in doubt, start as a Server Component. Move to client only when the compiler tells you to.

### Structure
- **Named exports only**: `export function ComponentName()` — never `export default`.
- **One component per file**, except small internal helpers used only within that file.
- **Props interface defined above the component**, named `{ComponentName}Props`.
- **Composition over configuration**: prefer `children` over render props, prefer render props over complex config objects.

```tsx
// CORRECT
interface MissionCardProps {
  mission: Mission;
  onApply: (id: string) => void;
}

export function MissionCard({ mission, onApply }: MissionCardProps) {
  // ...
}

// WRONG: default export, inline props, too many props
export default function MissionCard({
  title, description, budget, deadline, status, author, tags, onApply, onSave, onShare
}: { /* ... */ }) { /* ... */ }
```

### File Naming

| Category | Convention | Example |
|----------|-----------|---------|
| Files | kebab-case | `dashboard-shell.tsx` |
| Components | PascalCase export | `export function DashboardShell` |
| Hooks | camelCase | `use-auth.ts` / `useAuth` |
| Types | PascalCase | `type AuthState = { ... }` |
| Constants | UPPER_SNAKE_CASE | `const API_URL = "..."` |
| CSS variables | kebab-case | `--color-primary` |

---

## Data Fetching Patterns

### Server Components (preferred for initial page data)
Fetch directly in the component using `async` Server Components. No hooks needed.

### Client Components
Use TanStack Query exclusively. Every feature has query hooks in `features/*/hooks/`.

```tsx
// CORRECT — TanStack Query in a client component
const { data, isLoading, error } = useQuery({
  queryKey: ["missions", filters],
  queryFn: () => apiClient.getMissions(filters),
});

// WRONG — fetching in useEffect
useEffect(() => {
  fetch("/api/missions").then(/* ... */);
}, []);
```

### Rules
- **Never fetch in `useEffect`** — use TanStack Query (`useQuery` / `useMutation`) or RSC.
- **Optimistic updates** for mutations that affect visible UI (mark as read, toggle favorite).
- **Stale-while-revalidate** for list views — show cached data immediately, refresh in background.
- **Error and loading states** are mandatory for every query (see Error Handling below).

---

## Error Handling

### Rendering Errors
- Error boundaries at the layout level (`error.tsx` in each route group).
- Each route group (`(auth)`, `(dashboard)`, `(public)`) has its own error boundary with context-appropriate UI.

### API Errors
- All API errors caught via `ApiError` class from `src/shared/lib/api-client.ts`.
- **Never show raw error messages to users.** Map error codes to user-friendly messages.
- Display errors inline near the action that triggered them, not as global alerts.

### Loading States
- **Skeleton screens over spinners** — always prefer content placeholders that match the layout.
- Every page has a `loading.tsx` in its route directory.
- Every async client operation shows a loading indicator.

### Empty States
- Every list view has an empty state with a clear call-to-action.
- Empty states should guide the user toward the next action, not just say "No data."

---

## Performance Budgets

### Core Web Vitals
| Metric | Target | Measurement |
|--------|--------|-------------|
| LCP (Largest Contentful Paint) | < 2.5s | Lighthouse CI |
| FID (First Input Delay) | < 100ms | Lighthouse CI |
| CLS (Cumulative Layout Shift) | < 0.1 | Lighthouse CI |

### Bundle Size
- Initial JS bundle: **< 200KB gzipped**.
- Route-level code splitting is automatic with Next.js App Router.
- Audit bundle size with `@next/bundle-analyzer` before merging large features.

### Images
- Always use `next/image` — never raw `<img>` tags.
- Format preference: AVIF > WebP > JPEG.
- Always set explicit `width` and `height` to prevent layout shift.
- Lazy load all below-the-fold images (default behavior with `next/image`).

### Fonts
- Preload critical fonts (loaded via `next/font`).
- Subset to used character ranges when possible.
- Use `font-display: swap` to prevent invisible text during load.

### General
- Lazy load below-the-fold content with `React.lazy()` or dynamic imports.
- No layout shift: always reserve space for async content (skeleton placeholders).
- Defer non-critical third-party scripts.

---

## Accessibility (WCAG 2.1 AA)

Every component must meet these standards. Accessibility is not optional.

### Keyboard Navigation
- All interactive elements must be keyboard accessible (focusable, activatable).
- Logical tab order that matches visual order.
- Focus indicators visible on all focusable elements — never remove `outline` without replacing it.
- Keyboard shortcuts for power users where appropriate (Escape to close modals, Enter to submit).

### Semantic HTML
- Use semantic elements: `<nav>`, `<main>`, `<article>`, `<aside>`, `<header>`, `<footer>`, `<section>`.
- One `<main>` per page. One `<h1>` per page.
- Heading hierarchy must not skip levels (`h1` -> `h2` -> `h3`, never `h1` -> `h3`).

### Images and Icons
- All informative images must have meaningful `alt` text.
- Decorative images: `alt=""` (empty string, not omitted).
- Icon-only buttons must have `aria-label`.

### Color and Contrast
- Text color contrast ratio >= 4.5:1 against background.
- Large text (18px+ bold or 24px+ regular): >= 3:1.
- Never convey information through color alone — always pair with text, icons, or patterns.

### Forms
- Every `<input>` must have an associated `<label>` element (or `aria-label`).
- Error messages must be announced to screen readers using `aria-live="polite"` or `role="alert"`.
- Required fields must be indicated both visually and programmatically (`aria-required`).

---

## SEO Standards

Every public-facing page must implement these. Dashboard pages are excluded (noindex).

### Metadata
- Every page must have a unique `<title>` and `<meta name="description">`.
- Define metadata using Next.js `generateMetadata` or static `metadata` export in `page.tsx`.
- Title format: `{Page Title} | Marketplace` (consistent suffix).

### Structured Data
- JSON-LD structured data on all public pages (Organization, Service, BreadcrumbList).
- Provider profiles: `Person` or `Organization` schema.
- Mission listings: `JobPosting` schema.

### Technical SEO
- Canonical URLs on every page (`<link rel="canonical">`).
- `sitemap.xml` auto-generated via Next.js `sitemap.ts`.
- `robots.txt` configured via Next.js `robots.ts`.
- Open Graph meta tags (`og:title`, `og:description`, `og:image`) for social sharing.
- Twitter Card meta tags (`twitter:card`, `twitter:title`, `twitter:description`).

---

## Routing and Pages

- `app/` directory is for **routing only** — thin pages, layouts, metadata, and `loading.tsx` / `error.tsx` boundaries.
- Each `page.tsx` should be 5-20 lines: import feature components, compose them, return JSX.
- Business logic lives in `features/` and is composed in `app/` pages.
- Each route group uses parenthesized folders: `(auth)`, `(dashboard)`, `(public)`.

```tsx
// CORRECT: thin page
import { MissionList } from "@/features/mission/components/mission-list";
import { MissionFilters } from "@/features/mission/components/mission-filters";

export function MissionsPage() {
  return (
    <div>
      <MissionFilters />
      <MissionList />
    </div>
  );
}

// WRONG: logic in page
export function MissionsPage() {
  const [missions, setMissions] = useState([]); // NO — pages don't manage state
}
```

---

## Features

- Each feature is a self-contained module with its own `api/`, `components/`, and `hooks/` directories.
- **Features NEVER import from other features** — composition happens in `app/` pages.
- Features can import from `shared/` but never from `app/` or other features.
- If two features need the same logic, extract it to `shared/`.
- **Enforced by ESLint** (`import/no-restricted-paths` in `eslint.config.mjs`, P9). A feature → feature import fails CI: extract the shared surface (type / helper / hook / component) to `src/shared/<category>/<name>` and update both sides to import from there. The audit count is 0 — keep it that way.

```
features/mission/
├── api/            -> API call functions
├── components/     -> Mission-specific UI
├── hooks/          -> TanStack Query hooks (useMissions, useMission)
└── types.ts        -> Mission-specific types (if not covered by generated API types)
```

---

## State Management

| What | Where | Tool |
|------|-------|------|
| Server data (API responses) | `features/*/hooks/` | TanStack Query v5 |
| Auth state | `shared/hooks/use-auth.ts` | Zustand v5 (persisted) |
| Form state | Component-local | react-hook-form + zod |
| Ephemeral UI state | Component-local | `useState` |

- **No prop drilling** — use composition, context, or stores.
- **No Redux, no Context for server data** — TanStack Query handles caching, refetching, and optimistic updates.

---

## API Client

- Types are **generated, never hand-written** — run `npm run generate-api` to regenerate from backend OpenAPI schema.
- The generated types file is `src/shared/types/api.d.ts` (gitignored, generated on demand).
- Use `apiClient<T>()` from `src/shared/lib/api-client.ts` for all HTTP calls.
- The `ApiError` class provides structured error handling with `status`, `code`, and `message`.
- API base URL: `NEXT_PUBLIC_API_URL` env var (defaults to `http://localhost:8080`).

---

## Styling

- Tailwind CSS 4 with **CSS-first configuration** (no `tailwind.config.ts`).
- Design tokens defined as CSS variables in `src/styles/globals.css` using `@theme inline`.
- Color system: deep blue primary (`--primary`), slate neutrals, professional B2B palette.
- Dark mode via `.dark` class on `<html>` element (`@custom-variant dark`).
- Use `cn()` utility for conditional class merging (clsx + tailwind-merge).
- Use `class-variance-authority` (cva) for component variant definitions.
- shadcn/ui components go in `src/shared/components/ui/`.
- **No inline styles** — Tailwind utility classes only.
- **No hardcoded colors** — always use semantic tokens (`bg-primary`, `text-muted-foreground`).

---

## Auth and Middleware

- `src/middleware.ts` protects `/dashboard/*` routes — redirects to `/login` if no `access_token` cookie.
- Public paths: `/`, `/login`, `/register`, `/agencies`, `/freelances`, `/projects`.
- Auth state persisted in localStorage via Zustand persist middleware (key: `marketplace-auth`).
- User roles: `agency`, `enterprise`, `provider`.
- Sidebar navigation adapts based on user role.

---

## TypeScript

- Strict mode enabled in `tsconfig.json`.
- Path alias: `@/*` maps to `./src/*`.
- Use `type` imports for type-only imports: `import type { Foo } from "./bar"`.
- Prefer `interface` for object shapes that may be extended, `type` for unions/intersections.
- **No `any`** — use `unknown` and narrow with type guards.
- **No `as` type assertions** unless there is a comment explaining why it is safe.

---

## Environment Variables

- `NEXT_PUBLIC_API_URL` — Backend API base URL (default: `http://localhost:8080`)
- `NEXT_PUBLIC_APP_URL` — Frontend app URL (default: `http://localhost:3000`)
- Never commit `.env` files — use `.env.local` for local development.

---

## Commands

```bash
npm run dev             # Start dev server on port 3000
npm run build           # Production build
npm run start           # Start production server
npm run lint            # Run ESLint
npm run generate-api    # Generate TypeScript types from backend OpenAPI schema
```

---

## Adding a New Feature

1. Create the feature directory under `src/features/<name>/`.
2. Add `api/`, `components/`, `hooks/`, and optionally `types.ts`.
3. Implement API hooks using TanStack Query in `hooks/`.
4. Build components in `components/` (use `"use client"` only when needed).
5. Wire into routes in `src/app/` by importing and composing feature components.
6. Never import from other features — if shared logic is needed, extract to `src/shared/`.

## Adding a UI Component (shadcn/ui style)

1. Place the component in `src/shared/components/ui/<name>.tsx`.
2. Use `cva()` for variants, `cn()` for class merging.
3. Use CSS variable tokens from `globals.css` (e.g., `bg-primary`, `text-muted-foreground`).
4. Export named: `export function Button() { ... }`.
5. Props should extend native HTML element props where applicable.

---

## Performance & Caching

### Performance Optimization Patterns

1. **Server Components vs Client Components decision**
   ```
   Can this component render without JavaScript? → Server Component
   Does it need useState/useEffect/onClick?     → Client Component
   Does it only read data?                      → Server Component
   Does it need browser APIs?                   → Client Component
   ```

2. **Code splitting with dynamic imports**
   ```typescript
   // Heavy components loaded only when needed
   const MissionEditor = dynamic(() => import("@/features/mission/components/mission-editor"), {
     loading: () => <MissionEditorSkeleton />,
   })
   ```

3. **Image optimization rules**
   - ALWAYS use `next/image` — never `<img>`
   - Set explicit `width` and `height` (prevents CLS)
   - Use `priority` prop for above-the-fold images (LCP)
   - Format: AVIF > WebP > JPEG (Next.js auto-negotiates)
   - Lazy load below-fold images (default behavior)
   ```typescript
   <Image
     src={agency.logoUrl}
     alt={`Logo de ${agency.name}`}
     width={120}
     height={120}
     className="rounded-lg"
     priority={isAboveFold}
   />
   ```

4. **Data fetching optimization**
   - Server Components: fetch in the component, Next.js deduplicates automatically
   - Parallel fetches with `Promise.all` for independent data:
   ```typescript
   // GOOD: parallel
   const [profile, missions, reviews] = await Promise.all([
     getProfile(id),
     getMissions(id),
     getReviews(id),
   ])

   // BAD: sequential (waterfall)
   const profile = await getProfile(id)
   const missions = await getMissions(id)
   const reviews = await getReviews(id)
   ```

5. **Skeleton loading patterns**
   - Every page with async data MUST have a `loading.tsx` file
   - Use skeleton components that match the layout (not generic spinners)
   - Streaming: use `<Suspense>` boundaries for progressive loading

---

## SEO Strategy (Comprehensive)

This is a B2B marketplace — SEO is the primary acquisition channel. Every public page must be optimized.

### 1. Metadata per page type

```typescript
// app/(public)/agencies/[id]/page.tsx
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const agency = await getAgency(params.id)
  return {
    title: `${agency.name} — Agence ${agency.expertises[0]} | Marketplace Service`,
    description: `${agency.name} est une agence ${agency.expertises.join(", ")} basée à ${agency.city}. ${agency.about?.slice(0, 120)}...`,
    openGraph: {
      title: agency.name,
      description: agency.about?.slice(0, 200),
      images: [{ url: agency.logoUrl, width: 400, height: 400, alt: agency.name }],
      type: "profile",
    },
    twitter: {
      card: "summary",
      title: agency.name,
      description: agency.about?.slice(0, 200),
    },
    alternates: {
      canonical: `https://marketplace-service.com/agencies/${agency.id}`,
    },
  }
}
```

| Page type | Title pattern | Description pattern |
|-----------|--------------|---------------------|
| Home | "Marketplace Service — Trouvez le prestataire idéal" | "Plateforme B2B connectant agences, freelances et entreprises. Trouvez le prestataire parfait pour votre projet." |
| Agency profile | "{name} — Agence {expertise} \| Marketplace Service" | "{name} est une agence {expertises} basée à {city}. {about first 120 chars}" |
| Freelance profile | "{name} — {title} Freelance \| Marketplace Service" | "{name}, {title} freelance à {city}. {about first 120 chars}" |
| Project listing | "Projets disponibles — {count} projets \| Marketplace Service" | "Parcourez {count} projets B2B. Budget de {minBudget}€ à {maxBudget}€." |
| Project detail | "{title} — Projet {domain} \| Marketplace Service" | "{description first 150 chars}" |
| Agencies directory | "Annuaire des agences — {count} agences \| Marketplace Service" | "Trouvez la meilleure agence pour votre projet parmi {count} agences vérifiées." |
| Freelances directory | "Annuaire des freelances — {count} freelances \| Marketplace Service" | "Trouvez le freelance idéal parmi {count} professionnels vérifiés." |

### 2. Structured Data (JSON-LD) per entity

```typescript
// Agency profile → Organization schema
const agencyJsonLd = {
  "@context": "https://schema.org",
  "@type": "Organization",
  name: agency.name,
  description: agency.about,
  url: `https://marketplace-service.com/agencies/${agency.id}`,
  logo: agency.logoUrl,
  address: {
    "@type": "PostalAddress",
    addressLocality: agency.city,
    addressCountry: "FR",
  },
  aggregateRating: agency.averageRating ? {
    "@type": "AggregateRating",
    ratingValue: agency.averageRating,
    reviewCount: agency.reviewCount,
    bestRating: 5,
    worstRating: 1,
  } : undefined,
}

// Freelance profile → Person schema
const freelanceJsonLd = {
  "@context": "https://schema.org",
  "@type": "Person",
  name: `${freelance.firstName} ${freelance.lastName}`,
  jobTitle: freelance.title,
  description: freelance.about,
  address: {
    "@type": "PostalAddress",
    addressLocality: freelance.city,
    addressCountry: "FR",
  },
  knowsAbout: freelance.skills,
}

// Project → JobPosting schema
const projectJsonLd = {
  "@context": "https://schema.org",
  "@type": "JobPosting",
  title: project.title,
  description: project.description,
  datePosted: project.createdAt,
  employmentType: project.budgetType === "long_term" ? "CONTRACT" : "TEMPORARY",
  baseSalary: {
    "@type": "MonetaryAmount",
    currency: "EUR",
    value: {
      "@type": "QuantitativeValue",
      minValue: project.minBudget,
      maxValue: project.maxBudget,
    },
  },
  hiringOrganization: {
    "@type": "Organization",
    name: project.enterprise.name,
  },
  jobLocation: {
    "@type": "Place",
    address: { "@type": "PostalAddress", addressCountry: "FR" },
  },
}
```

### 3. Breadcrumbs schema

```typescript
// Every public page has breadcrumbs for Google
const breadcrumbJsonLd = {
  "@context": "https://schema.org",
  "@type": "BreadcrumbList",
  itemListElement: [
    { "@type": "ListItem", position: 1, name: "Accueil", item: "https://marketplace-service.com" },
    { "@type": "ListItem", position: 2, name: "Agences", item: "https://marketplace-service.com/agencies" },
    { "@type": "ListItem", position: 3, name: agency.name },
  ],
}
```

Render in layout:
```typescript
<script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
```

### 4. Dynamic Sitemap

```typescript
// app/sitemap.ts
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [agencies, freelances, projects] = await Promise.all([
    getPublicAgencies(),
    getPublicFreelances(),
    getPublicProjects(),
  ])

  const staticPages = [
    { url: "https://marketplace-service.com", lastModified: new Date(), changeFrequency: "daily" as const, priority: 1 },
    { url: "https://marketplace-service.com/agencies", lastModified: new Date(), changeFrequency: "daily" as const, priority: 0.9 },
    { url: "https://marketplace-service.com/freelances", lastModified: new Date(), changeFrequency: "daily" as const, priority: 0.9 },
    { url: "https://marketplace-service.com/projects", lastModified: new Date(), changeFrequency: "daily" as const, priority: 0.9 },
  ]

  const agencyPages = agencies.map((a) => ({
    url: `https://marketplace-service.com/agencies/${a.id}`,
    lastModified: a.updatedAt,
    changeFrequency: "weekly" as const,
    priority: 0.8,
  }))

  const freelancePages = freelances.map((f) => ({
    url: `https://marketplace-service.com/freelances/${f.id}`,
    lastModified: f.updatedAt,
    changeFrequency: "weekly" as const,
    priority: 0.8,
  }))

  const projectPages = projects.map((p) => ({
    url: `https://marketplace-service.com/projects/${p.id}`,
    lastModified: p.updatedAt,
    changeFrequency: "daily" as const,
    priority: 0.7,
  }))

  return [...staticPages, ...agencyPages, ...freelancePages, ...projectPages]
}
```

### 5. robots.txt

```typescript
// app/robots.ts
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: ["/dashboard/", "/api/", "/login", "/register"],
      },
    ],
    sitemap: "https://marketplace-service.com/sitemap.xml",
  }
}
```

### 6. Canonical URL strategy

- Every page has a canonical URL to prevent duplicate content
- Pagination: `?cursor=abc` pages have canonical pointing to the first page (no cursor)
- Profile pages: canonical is always the clean URL without query params
- Use `alternates.canonical` in metadata

### 7. Internal linking strategy

- Agency profiles link to their projects, reviews, and team members
- Project pages link back to the enterprise and to similar projects
- Directory pages have filters that generate crawlable URLs: `/agencies?city=paris&expertise=web`
- Footer contains links to all major directory pages

### 8. Performance for SEO

Google uses Core Web Vitals as a ranking factor:
- **LCP < 2.5s**: Use Server Components, preload critical images with `priority`
- **FID < 100ms**: Minimize client-side JS on public pages
- **CLS < 0.1**: Always set width/height on images, use font-display: swap

### 9. Social sharing meta

Every public page must have:
- og:title, og:description, og:image, og:url, og:type
- twitter:card, twitter:title, twitter:description
- Images: 1200x630px for og:image (Facebook/LinkedIn), 400x400 for twitter summary
