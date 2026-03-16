---
name: add-web-page
description: Add a new page to the Next.js web app in the correct route group, wired to the right feature and sidebar navigation.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# Add Web Page

Create the page: **$ARGUMENTS**

You are adding a new page to the Next.js web app. Follow EVERY step below.

---

## STEP 0 — Understand the request

Parse `$ARGUMENTS` to determine:
- **Page name and purpose**
- **Route group**: `(auth)`, `(dashboard)`, or `(public)`
- **If dashboard**: which role(s) — `agency`, `enterprise`, `provider`, or multiple
- **Which feature** provides the components for this page

If the request is ambiguous, ask the user to clarify before proceeding.

---

## STEP 1 — Determine the file path

Route structure:
```
web/src/app/(auth)/{page}/page.tsx          -> /login, /register
web/src/app/(dashboard)/{role}/{page}/page.tsx -> /dashboard/provider/missions
web/src/app/(public)/{page}/page.tsx        -> /agencies, /freelances
```

- Auth pages: `web/src/app/(auth)/{name}/page.tsx`
- Dashboard pages: `web/src/app/(dashboard)/{role}/{name}/page.tsx`
- Public pages: `web/src/app/(public)/{name}/page.tsx`

If the page needs to be visible to multiple roles, create one `page.tsx` per role directory.

---

## STEP 2 — Verify the feature exists

Before creating the page, check that the feature it depends on exists:
- Confirm `web/src/features/{feature}/components/` has the component(s) the page will import
- If the feature does not exist, tell the user to run `/add-web-feature` first

---

## STEP 3 — Create the page file

**Pattern** (reference existing pages in `web/src/app/`):

```tsx
import type { Metadata } from "next"
import { {Component} } from "@/features/{feature}/components/{component-file}"

export const metadata: Metadata = {
  title: "{Page Title}",
}

export default function {PageName}Page() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold text-foreground">{Heading}</h1>
      <p className="text-sm text-muted-foreground">{Description}</p>
      <{Component} />
    </div>
  )
}
```

**Page rules:**
- Server Component by default — never add `"use client"` to a page
- No `useState`, `useEffect`, or event handlers in page files
- Under 20 lines — compose feature components, do not build UI here
- Named imports from features, default export for the page function
- Include `Metadata` export for SEO
- French-language headings and descriptions (the marketplace targets French B2B)

---

## STEP 4 — Add loading and error boundaries (if needed)

For pages that fetch data, optionally create siblings:

- `loading.tsx` — loading skeleton shown during Suspense
- `error.tsx` — error boundary with retry button (`"use client"` required)

```tsx
// loading.tsx
export default function Loading() {
  return <div className="animate-pulse space-y-4">...</div>
}
```

Only create these if the page uses async Server Components or heavy data fetching.

---

## STEP 5 — Update sidebar navigation (dashboard pages only)

If this is a dashboard page, update `web/src/shared/components/layouts/sidebar.tsx`:

1. Read the current sidebar file
2. Add a `NavItem` to the correct role array (`agencyNav`, `enterpriseNav`, or `providerNav`)
3. Choose an appropriate icon from the already-imported `lucide-react` icons, or add a new import
4. Set `href` to `/dashboard/{role}/{name}`
5. Set `label` to the French-language page name
6. Place the item in a logical position (group related items together)

Example addition:
```ts
{ label: "Contrats", href: "/dashboard/provider/contracts", icon: FileText },
```

---

## STEP 6 — Verify

1. The page imports only from `@/features/` and `@/shared/` — never from another page or `@/config/`
2. The page has no business logic — it is a thin composition layer
3. If sidebar was updated, the `href` matches the actual file path
4. No `"use client"` on the page unless absolutely required (it should not be)

---

## Output

When finished, report:
1. Page file(s) created with their paths
2. Route URL(s) the page will be accessible at
3. Sidebar changes (if any)
4. Any loading/error boundaries created
