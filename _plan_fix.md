# Fix prod-pipeline bugs — plan

## Bug 1 — `privacy/page.tsx` has both `metadata` and `generateMetadata`
Root cause: commit `94dcf308` left a static `metadata` export (line 9-11) AND a `generateMetadata` async export (line 13-25). Next.js 16 rejects this combination during build. The `generateMetadata` version is the richer one (sets localized `title` + `description` AND still keeps `robots: { index: false, follow: false }`). Fix: remove the redundant static `metadata` export; keep `generateMetadata` only. Verify with `npm run build` + a vitest snapshot if any exists for this page.

## Bug 2 — 3 vitest failures
Two on `decisions-automatisees.test.tsx`, one on `sessions-list.test.tsx`. Approach: read each failing test + the asserted component, run vitest with `--reporter=verbose` per file to see the exact diff, then determine if the test expectation is stale (component changed) or the component is wrong. Per CLAUDE.md, never skip a test to make the suite green — fix the underlying issue.

## Bug 3 — 6 tsc errors on e2e specs
- `job-applicant-thread.spec.ts:254` — `.body` on `never` → a `.find()` result wasn't narrowed; add a `if (!x) throw` guard.
- `search-tracking-handoff-real.spec.ts:176/186/187` — same pattern (`.find()` result destructured into `never`); add guard.
- `xss-jsonld.spec.ts:64/74` — two `@ts-expect-error` directives are unused (their underlying issue was fixed); remove them.
Verify by re-running `tsc --noEmit` and `playwright test <file> --list` to ensure specs are still discoverable.

## Validation
After all 3 fixes:
1. `cd web && npx tsc --noEmit`
2. `cd web && npx vitest run`
3. `cd web && npm run build`
4. `cd backend && go build ./... && go vet ./... && go test ./... -count=1 -timeout 180s`
