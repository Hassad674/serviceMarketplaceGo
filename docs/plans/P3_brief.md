# P3 — Web shadcn primitives + migrate 309 buttons + 95 inputs

**Phase:** F.1 CRITICAL #3
**Source audit:** QUAL-FINAL (web DRY massif) + asymétrie web vs admin
**Effort:** 1 jour mécanique
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p3-web-shadcn-primitives`

## Problem

The admin app (`admin/src/components/ui/`) has shadcn primitives (Button/Input/Card/Modal/Select), but the web app does NOT. Result: **309 raw `<button>` sites + 95 raw `<input>` sites** across `web/src/`, each one re-declaring the same Tailwind class soup. DRY violation massive + asymmetric quality between web and admin (the audit calls admin "top-5%" and web "top-20%" partly because of this).

## Goal

Port the shadcn primitives from `admin/` to `web/`, keep them in sync visually with the design system tokens (rose primary, gradient hero/subtle/warm, shadow xs/sm/md/lg/xl/glow, animation classes), then migrate every raw button/input/card/modal/select site to the new primitive. **Zero visual regression.**

## Discovery (do first)

```bash
cd /tmp/mp-p3-shadcn/web
# Existing primitives (might already have some)
ls src/shared/components/ui/

# Raw sites to migrate
grep -rn "<button" src/ --include="*.tsx" --include="*.jsx" | grep -v "node_modules\|\.next/\|\.test\." | wc -l
grep -rn "<input" src/ --include="*.tsx" --include="*.jsx" | grep -v "node_modules\|\.next/\|\.test\." | wc -l

# Count per file (find hotspots)
grep -rn "<button" src/ --include="*.tsx" | grep -v "\.test\." | cut -d: -f1 | sort | uniq -c | sort -rn | head -20
```

Use the admin app's primitives as the reference template:
```bash
cat admin/src/components/ui/button.tsx
cat admin/src/components/ui/input.tsx
cat admin/src/components/ui/card.tsx
cat admin/src/components/ui/modal.tsx  # or dialog.tsx
cat admin/src/components/ui/select.tsx
```

## Primitives to port (5 minimum, more if admin has them)

1. **Button** — variants: `primary | secondary | outline | ghost | destructive` + sizes: `sm | md | lg`. Uses `gradient-primary` on `primary` variant. `hover:shadow-glow` + `active:scale-[0.98]` per CLAUDE.md.
2. **Input** — `h-10`, `rounded-lg`, `shadow-xs`, `focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10`. Error state: `border-red-500 ring-4 ring-red-500/10`. Supports `<label>` association via `htmlFor`.
3. **Card** — `bg-white rounded-2xl border border-slate-100 shadow-sm p-6`. Interactive variant: `hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5`.
4. **Modal / Dialog** — Radix Dialog wrapper, glass effect (`.glass-strong`), `animate-scale-in`, focus trap, escape-to-close, click-outside-to-close.
5. **Select** — Radix Select wrapper, same input styling, dropdown with `animate-fade-in`.

Optional (migrate if admin has them and they're used >5 times in web):
- Avatar
- Badge / role-badge
- Toast / sonner
- Tooltip
- Skeleton
- Switch / Checkbox / RadioGroup

Use **`class-variance-authority` (cva)** for variants — it's already a dep. Use **`clsx` + `tailwind-merge`** via the existing `cn()` helper in `src/shared/lib/utils.ts`.

## Fix pattern

### For Button:

```tsx
// BEFORE (one of 309)
<button
  type="submit"
  className="bg-rose-500 hover:bg-rose-600 text-white font-medium px-4 py-2 rounded-lg shadow-sm transition disabled:opacity-50"
  disabled={loading}
>
  Submit
</button>

// AFTER
<Button type="submit" variant="primary" size="md" disabled={loading}>
  Submit
</Button>
```

### For Input:

```tsx
// BEFORE
<input
  type="email"
  className="h-10 px-3 rounded-lg border border-slate-200 focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 shadow-xs"
  {...register("email")}
/>

// AFTER
<Input type="email" {...register("email")} aria-invalid={!!errors.email} />
```

### For Card / Modal / Select: similar pattern — wrap raw structure in primitive composition.

## Hard constraints (paranoid mode)

- **Tests BEFORE bulk migration**: write `web/src/shared/components/ui/__tests__/{button,input,card,modal,select}.test.tsx` for each new primitive. Cover variants, sizes, disabled state, error state, accessibility (ARIA), keyboard navigation. Each primitive ≥ 90% line coverage.
- **Visual regression sanity**: for components with existing tests, render-snapshot before/after to verify identical output. Add `data-testid` if needed for matching.
- **Migration approach** — DO NOT do a single mega-commit. Split:
  1. Commit 1: Add `button.tsx` primitive + tests (no migrations yet)
  2. Commit 2: Migrate top 5-10 hotspot files (most-used components) to Button. Run tests, verify visual.
  3. Commit 3: Migrate remaining ~290 sites in alphabetic order. One commit per `features/<X>/` if scope is large.
  4. Commit 4-7: Same pattern for Input, Card, Modal, Select.
  5. Final commit: ESLint rule promote `react/forbid-elements` for `button` + `input` outside `shared/components/ui/` (force future code to use primitives).
- **Validation pipeline before EVERY commit**:
  ```bash
  cd /tmp/mp-p3-shadcn/web
  npx tsc --noEmit
  npx vitest run --changed
  npx eslint src/  # 0 errors
  ```
- **Storybook NOT required** — but if you find time, add a `*.stories.tsx` per primitive (skip if not budget).
- **a11y mandatory** :
  - Every Button has `type` (button/submit/reset)
  - Every Input has either an associated `<label>` (via `htmlFor`) or `aria-label`
  - Modal has `aria-labelledby`, `aria-describedby`, focus trap, ESC key
  - Select supports keyboard nav (arrow keys + enter)

## Out-of-scope (flag, don't fix)

- Admin primitives — they exist already, leave them
- Mobile primitives — different framework (Flutter), not in scope
- Backend — out of scope
- LiveKit / call: `web/src/features/call/` — never touch
- `.github/workflows/*` — token can't push
- New components beyond Button/Input/Card/Modal/Select unless heavily-used

## Branch ownership

Agent creates `fix/p3-web-shadcn-primitives` from clean `main` via `git worktree add`. Never touches another branch.

## Final report (under 800 words)

Lead with PR URL.

1. Primitives ported (count + list with line counts)
2. Sites migrated per primitive (Button N/309, Input M/95, etc.)
3. Tests added (count per primitive)
4. ESLint rule promoted (yes/no, what's now blocked)
5. Visual regression check (snapshot count, all green)
6. Validation pipeline output
7. "Branch ownership confirmed"
8. Out-of-scope items found (if any)

GO.
