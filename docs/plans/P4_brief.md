# P4 — 27 raw `<img>` → `next/image` (LCP hardening)

**Phase:** F.1 CRITICAL #4
**Source audit:** PERF-FINAL-W-XX (`auditperf.md` web section)
**Effort:** ½j mécanique
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p4-img-to-next-image`

## Problem

27 raw `<img>` tags survive in the web codebase (regression from 7 in the previous audit). Every site that listing-renders profile/avatar/portfolio thumbnails currently uses `<img>`, which:
- Skips the `next/image` AVIF/WebP auto-negotiation (5-10× larger payload than necessary)
- Skips lazy-loading defaults (impacts CLS + bandwidth on listings)
- Doesn't reserve space (CLS regression)
- LCP suffers on `/agencies`, `/freelances`, `/projects` — the SEO-critical surfaces

## Discovery

```bash
cd /tmp/mp-p4-img/web
grep -rn '<img' src/ --include="*.tsx" --include="*.jsx" | grep -v "node_modules\|.next/\|\.test\.\|//.*<img"
```

Expected: ~27 sites. Inventory them and classify each:
1. **Avatar / icon** (≤64px) → `next/image` with `width={64} height={64}`, eager only if above-fold
2. **Card thumbnail** (300-500px) → `next/image` with appropriate dims, `priority` only on hero/above-fold
3. **Portfolio media large** (>720px) → `next/image` `sizes="(max-width: 768px) 100vw, 50vw"`, lazy by default
4. **Logo SVG / inline** → keep `<img>` if it's an `.svg` from local assets (next/image SVG support is limited); confirm tree-shaking instead. Annotate with `// reason: SVG inline keeps vector quality`.
5. **Mailtemplate / email-rendered** → keep `<img>` (email clients don't run JS)

## Fix pattern

```tsx
// BEFORE
<img src={user.avatarUrl} alt={user.name} className="w-12 h-12 rounded-full" />

// AFTER
import Image from "next/image"
<Image
  src={user.avatarUrl}
  alt={user.name}
  width={48}
  height={48}
  className="rounded-full"
/>
```

For external (R2 / arbitrary URL) sources, ensure `next.config.ts` has the appropriate `images.remotePatterns` entry. Don't use `domains` — deprecated.

For dynamic images where dimensions are unknown:
```tsx
<Image src={url} alt={alt} fill sizes="(max-width: 768px) 100vw, 50vw" />
```
With a parent `<div className="relative aspect-square">`.

## Hard constraints (paranoid mode)

- **Tests BEFORE bulk migration**: write a `tests/no-raw-img.spec.ts` ESLint test (or vitest snapshot) that asserts `<img` count = 0 in `src/` (excluding annotated SVG/email exceptions). It must FAIL on `main` and PASS on the branch.
- **Lighthouse perf check on 3 listing routes**: run `npx lighthouse http://localhost:3001/fr/agencies --only-categories=performance` and `freelances` and `projects`. LCP must be ≤ 2.5s (or improved vs main baseline by ≥ 20%).
- **Visual regression**: rendering must be byte-identical (same dims, same border-radius). For each migrated site, add a `tests/components/__tests__/<component>.test.tsx` that renders the component and asserts the expected rendered classes/dims are stable.
- **Validation pipeline before every commit**:
  ```bash
  cd /tmp/mp-p4-img/web
  npx tsc --noEmit
  npx vitest run --changed
  npx eslint src/  # 0 errors
  ```
- **One commit per logical group** (~5 commits expected): avatars, card thumbnails, portfolio media, logos exceptions documented, ESLint rule + tests.

## Tests required

1. **`web/scripts/check-no-raw-img.mjs`** (or use ESLint `@next/next/no-img-element` rule promoted to `error` and remove from any legacy override):
   ```bash
   $ npm run check:no-raw-img
   ✓ 0 violations (was 27 on main)
   ```
2. **Component tests** for each significantly-migrated component (asserting Image renders with correct width/height props).
3. **Lighthouse CLI snapshot in CI** (optional, but document the baseline numbers in PR description).

## OFF-LIMITS

- LiveKit / call: `web/src/features/call/` including `call-slot.tsx` + `call-runtime.tsx` — never touch
- `.github/workflows/*` — token can't push
- Other plans (P1, P2, P3, P5 etc.) — never touch
- Backend / admin / mobile — out of scope

## Branch ownership

Agent creates `fix/p4-img-to-next-image` from clean `main` via `git worktree add`. Never touches another branch.

## Final report (under 500 words)

Lead with PR URL.

1. Inventory: 27 sites → migrated 24 / kept-as-img 3 (with reasons)
2. Lighthouse delta on 3 listings (LCP before / after)
3. ESLint rule promoted to error (zero new `<img>` will pass CI)
4. Tests added (count + names)
5. Validation pipeline output
6. Branch ownership confirmed
