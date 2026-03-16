---
name: check-web
description: Verify web app architecture compliance — feature isolation, component quality, performance, accessibility, and data fetching patterns. Run to detect violations before they accumulate.
user-invocable: true
allowed-tools: Read, Bash, Grep, Glob, Agent
---

# Check Web — Architecture & Quality Verification

Target: **$ARGUMENTS**

If `$ARGUMENTS` is empty, check the ENTIRE web app. Otherwise, check only the specified feature(s) or area(s).

You are the architecture guardian for the Next.js web app. Run every check below and produce a clear pass/fail report.

---

## CHECK 1 — Feature isolation

### 1a. No cross-feature imports

Verify that no file in `web/src/features/{X}/` imports from `web/src/features/{Y}/`.

**How to check:**
Use Grep to search all files in `web/src/features/` for import patterns containing `@/features/`. For each match, verify the import stays within the SAME feature folder.

**FAIL if:** `features/invoice/components/foo.tsx` imports from `@/features/mission/...`

### 1b. Features only import from allowed sources

Each feature file can ONLY import from:
- Its own feature directory (`../api/`, `../hooks/`, `./sub-component`)
- `@/shared/` (components, hooks, lib, types)
- External npm packages (react, next, @tanstack/react-query, zod, lucide-react, etc.)

**FAIL if:** A feature imports from `@/app/`, `@/config/`, or another feature.

### 1c. No feature exports consumed by other features

For each feature, grep the entire `web/src/features/` directory for imports referencing that feature. Any import from OUTSIDE that feature = **FAIL**.

Only `web/src/app/` pages are allowed to import from features.

---

## CHECK 2 — Component quality

### 2a. File length

No file in `web/src/features/` or `web/src/shared/` should exceed 600 lines.

**How to check:**
```bash
cd /home/hassad/Documents/marketplaceServiceGo
find web/src/features web/src/shared -name "*.tsx" -o -name "*.ts" | xargs wc -l | sort -rn | head -20
```

**FAIL if:** Any file exceeds 600 lines. **WARN if:** Any file exceeds 400 lines.

### 2b. No unnecessary "use client"

Check files with `"use client"` directive. A component needs `"use client"` ONLY if it:
- Uses hooks (`useState`, `useEffect`, `useQuery`, `useMutation`, custom hooks)
- Has event handlers (`onClick`, `onChange`, `onSubmit`)
- Uses browser APIs (`window`, `document`, `localStorage`)

**How to check:**
For each `"use client"` file, verify it actually uses one of the above. If it only renders JSX with props, it should be a Server Component.

**FAIL if:** A `"use client"` file contains none of the above patterns.

### 2c. Named exports only in features

All exports in `web/src/features/` must be named exports: `export function`, `export type`, `export const`.

**FAIL if:** Any file in `features/` uses `export default`.

### 2d. No `any` type

Search all `.ts` and `.tsx` files in `web/src/` for `: any`, `as any`, `<any>`.

**FAIL if:** `any` is used without an explicit `// eslint-disable` comment explaining why.

### 2e. No `@ts-ignore` or `@ts-expect-error`

Search for `// @ts-ignore` and `// @ts-expect-error` in all source files.

**FAIL if:** Any occurrence found without a justification comment.

---

## CHECK 3 — Performance

### 3a. Images use next/image

Search all `.tsx` files for raw `<img` tags.

**FAIL if:** Any `<img` tag is used instead of `next/image` `<Image` component.

### 3b. No heavy imports in client components

For files with `"use client"`, check for imports of large libraries that should not be in client bundles:
- `moment` (use date-fns or Intl)
- `lodash` (use individual lodash-es imports or native methods)
- `axios` (the project uses apiClient)

**FAIL if:** Any of these are imported in a `"use client"` file.

### 3c. Dynamic imports for heavy components

Check if any `"use client"` component imports other large components (modals, editors, charts) statically. These should use `next/dynamic` or `React.lazy`.

**WARN if:** Large interactive components are statically imported in frequently rendered pages.

---

## CHECK 4 — Accessibility

### 4a. Images have alt text

Search all `.tsx` files for `<Image` (next/image) and `<img` tags. Verify each has an `alt` prop.

**FAIL if:** Any image element is missing the `alt` prop.

### 4b. Buttons have accessible text

Search for `<button` elements. Verify each has:
- Text content inside the button, OR
- An `aria-label` prop, OR
- An `aria-labelledby` prop

**FAIL if:** A `<button` has no accessible text (icon-only buttons without `aria-label`).

### 4c. Form inputs have labels

Search for `<input`, `<select`, `<textarea` elements. Verify each is associated with a label via:
- A wrapping `<label>` element, OR
- An `id` + matching `htmlFor` on a `<label>`, OR
- An `aria-label` or `aria-labelledby` prop

**WARN if:** Any form input lacks a label association.

### 4d. Interactive elements are keyboard accessible

Verify that `onClick` handlers on non-button/non-link elements also have:
- `role="button"` and `tabIndex={0}`
- `onKeyDown` or `onKeyPress` handler for Enter/Space

**FAIL if:** A `<div onClick=` or `<span onClick=` lacks keyboard accessibility attributes.

---

## CHECK 5 — Data fetching patterns

### 5a. No useEffect for data fetching

Search `"use client"` files for the pattern: `useEffect` combined with `fetch`, `apiClient`, or state setters for data loading.

**FAIL if:** Any component uses `useEffect` + `useState` for data fetching instead of TanStack Query `useQuery` or Server Components.

### 5b. All mutations use useMutation

Search for direct `apiClient` calls with `method: "POST"`, `"PUT"`, `"PATCH"`, `"DELETE"` inside components (not in api/ files). These should go through `useMutation`.

**FAIL if:** A component calls `apiClient` directly for mutations instead of using a `useMutation` hook.

### 5c. Loading states for async operations

For each `useQuery` call, verify the component handles the `isLoading` or `isPending` state with a loading indicator.

**WARN if:** A query result is used without checking `isLoading` / `isPending`.

### 5d. Error states for async operations

For each `useQuery` and `useMutation` call, verify the component handles the `isError` or `error` state.

**WARN if:** A query/mutation result is used without checking `isError`.

---

## CHECK 6 — Page thinness

Verify that files in `web/src/app/` are thin routing layers:
- No `useState`, `useEffect`, or event handlers in page files
- Pages should be under 20 lines (warn if over 30)
- Pages import from `@/features/` and compose, nothing else

**FAIL if:** A page file contains business logic, hooks, or exceeds 30 lines.

---

## Report format

Output a structured report:

```
# Web Architecture Check Report

## Summary
- Total checks: X
- Passed: X
- Failed: X
- Warnings: X

## Results

### CHECK 1 — Feature isolation
- [PASS] 1a. No cross-feature imports
- [FAIL] 1b. Allowed sources — features/invoice/hooks/use-invoice.ts imports from @/config/site
  -> Fix: move needed config to @/shared/lib/ or pass as prop

### CHECK 2 — Component quality
- [PASS] 2a. File length — all files under 600 lines (longest: sidebar.tsx at 134 lines)
- [WARN] 2b. Unnecessary "use client" — features/mission/components/mission-status.tsx has "use client" but only renders JSX
  -> Fix: remove "use client" directive
...

### CHECK 5 — Data fetching patterns
- [FAIL] 5a. useEffect data fetching — features/messaging/components/chat.tsx uses useEffect + fetch
  -> Fix: replace with useQuery from @tanstack/react-query
```

For each failure, provide:
1. Exact file path
2. What rule it violates
3. How to fix it
