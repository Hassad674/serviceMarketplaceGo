---
name: add-web-feature
description: Scaffold a complete new feature for the Next.js web app. Creates API functions, TanStack Query hooks, components, and a dashboard page with proper feature isolation.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# Add Web Feature

Create the web feature: **$ARGUMENTS**

You are scaffolding a new feature for the Next.js web app. Follow EVERY step below in order. Read existing code for patterns before writing anything.

---

## STEP 0 — Understand the request

Parse `$ARGUMENTS` to determine:
- **Feature name** (kebab-case for files, PascalCase for types/components)
- **Core entity** and its fields (what data does this feature manage?)
- **CRUD operations** or custom interactions needed
- **Which roles** see this feature: `agency`, `enterprise`, `provider`, or multiple

If the request is ambiguous, ask the user to clarify before proceeding.

---

## STEP 1 — Create API file

Create `web/src/features/{feature}/api/{feature}-api.ts`:

**Pattern** (reference `web/src/features/auth/api/auth-api.ts`):
- Import `apiClient` from `@/shared/lib/api-client`
- Define types inline (they will be replaced by generated OpenAPI types later)
- Named export for each API function
- All functions return typed Promises

```ts
import { apiClient } from "@/shared/lib/api-client"

export type {Entity} = {
  id: string
  // ... fields matching API response
  created_at: string
  updated_at: string
}

export type Create{Entity}Data = {
  // ... fields for creation
}

export type Update{Entity}Data = Partial<Create{Entity}Data>

export async function get{Entity}s(token: string): Promise<{Entity}[]> {
  return apiClient<{Entity}[]>("/api/v1/{feature}s", { token })
}

export async function get{Entity}(token: string, id: string): Promise<{Entity}> {
  return apiClient<{Entity}>(`/api/v1/{feature}s/${id}`, { token })
}

export async function create{Entity}(token: string, data: Create{Entity}Data): Promise<{Entity}> {
  return apiClient<{Entity}>("/api/v1/{feature}s", { method: "POST", body: data, token })
}

export async function update{Entity}(token: string, id: string, data: Update{Entity}Data): Promise<{Entity}> {
  return apiClient<{Entity}>(`/api/v1/{feature}s/${id}`, { method: "PATCH", body: data, token })
}

export async function delete{Entity}(token: string, id: string): Promise<void> {
  return apiClient<void>(`/api/v1/{feature}s/${id}`, { method: "DELETE", token })
}
```

Only include the functions that match the requested operations. Do not generate unused endpoints.

---

## STEP 2 — Create TanStack Query hooks

Create `web/src/features/{feature}/hooks/use-{feature}.ts`:

**Rules:**
- `"use client"` directive at the top
- Import types and API functions from `../api/{feature}-api`
- Import `useAuth` from `@/shared/hooks/use-auth` for the access token
- Use `useQuery` for reads, `useMutation` + `useQueryClient` for writes
- Query keys follow `["{feature}s"]` or `["{feature}s", id]` pattern
- Mutations invalidate related queries on success

```ts
"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/shared/hooks/use-auth"
import {
  get{Entity}s,
  get{Entity},
  create{Entity},
  update{Entity},
  delete{Entity},
  type {Entity},
  type Create{Entity}Data,
} from "../api/{feature}-api"

export function use{Entity}s() {
  const { accessToken } = useAuth()
  return useQuery({
    queryKey: ["{feature}s"],
    queryFn: () => get{Entity}s(accessToken!),
    enabled: !!accessToken,
  })
}

export function use{Entity}(id: string) {
  const { accessToken } = useAuth()
  return useQuery({
    queryKey: ["{feature}s", id],
    queryFn: () => get{Entity}(accessToken!, id),
    enabled: !!accessToken && !!id,
  })
}

export function useCreate{Entity}() {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Create{Entity}Data) => create{Entity}(accessToken!, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["{feature}s"] }),
  })
}

export function useDelete{Entity}() {
  const { accessToken } = useAuth()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => delete{Entity}(accessToken!, id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["{feature}s"] }),
  })
}
```

Only include hooks for operations that are actually needed.

---

## STEP 3 — Create components

Create components in `web/src/features/{feature}/components/`:

Typical set (adjust to the feature):
- `{feature}-list.tsx` — Server Component or client list with loading/empty states
- `{feature}-card.tsx` — Card rendering a single entity
- `{feature}-form.tsx` — `"use client"` form using react-hook-form + zod

**Component rules:**
- Named exports only: `export function {Feature}List()`
- Server Components by default — add `"use client"` only for interactivity
- Import types from `../api/{feature}-api` and hooks from `../hooks/use-{feature}`
- Import shared UI from `@/shared/components/ui/` and utilities from `@/shared/lib/utils`
- Styling: Tailwind only, use `cn()` for conditional classes
- All `<img>` must use `next/image`
- Include loading states (skeleton or spinner) for async data
- Include empty states when lists have no items
- Keep each component under 200 lines of JSX

---

## STEP 4 — Create dashboard page

Create the page at `web/src/app/(dashboard)/{role}/{feature}/page.tsx`:

**Pattern** (reference existing pages in `web/src/app/(dashboard)/`):
- Thin page: metadata + feature component composition only
- Server Component by default
- Under 20 lines

```tsx
import type { Metadata } from "next"
import { {Feature}List } from "@/features/{feature}/components/{feature}-list"

export const metadata: Metadata = {
  title: "{Feature title}",
}

export default function {Role}{Feature}Page() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold text-foreground">{Page heading}</h1>
      <{Feature}List />
    </div>
  )
}
```

If the feature is visible to multiple roles, create a page under each role directory. Pages can differ in heading text but import from the same feature.

---

## STEP 5 — Update sidebar navigation

Update `web/src/shared/components/layouts/sidebar.tsx`:
- Add a `NavItem` entry to the appropriate role's navigation array (`agencyNav`, `enterpriseNav`, `providerNav`)
- Choose an appropriate icon from `lucide-react`
- Use the correct `href`: `/dashboard/{role}/{feature}`
- Place the item in a logical position within the nav array

---

## STEP 6 — Verify feature isolation

Run these checks before reporting done:

1. **No cross-feature imports**: grep all files in `web/src/features/{feature}/` — they must ONLY import from:
   - `@/shared/` (components, hooks, lib, types)
   - Their own feature directory (`../api/`, `../hooks/`, `./`)
   - External packages (react, next, tanstack, zod, lucide-react)

2. **No other feature imports this one**: grep the entire `web/src/features/` directory for imports referencing the new feature name — only `app/` pages should import from it.

3. **Named exports only**: no `export default` in feature files (pages in `app/` use default exports, features do not).

4. **No `any` types**: grep for `: any` or `as any` in all new files.

If any check fails, fix it before finishing.

---

## Output

When finished, report:
1. Files created (grouped: api, hooks, components, pages)
2. Files modified (sidebar.tsx)
3. Feature isolation verification result
4. Any decisions made and why
