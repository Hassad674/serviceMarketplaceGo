# 0006. Feature isolation enforced by ESLint, not convention

Date: 2026-04-30

## Status

Accepted

## Context

The web app (`web/`) and admin app (`admin/`) follow a
feature-based layout: each feature lives under
`src/features/<name>/` with its own `api/`, `components/`,
`hooks/`, `actions/` (web only), and `types/` directories. This
mirrors the backend's module independence (ADR 0001) and is the
foundation of the project's "delete the folder" invariant
(`CONTRIBUTING.md §3`).

The rule is: **a feature never imports from another feature**.
Cross-feature data flows through `src/shared/` (primitives,
hooks, contexts) or via composition in `src/app/` pages — never
via direct `import { x } from "@/features/other-feature/..."`.

The rule is documented in `web/CLAUDE.md` and `admin/CLAUDE.md`.
For a long time we relied on convention + code review to enforce
it. This worked at first; by the third PR a violation had
slipped in (a billing page reaching into messaging hooks for a
minor styling reason). The reviewer caught it, but the lesson
was clear: convention is not enough.

A naive grep for `@/features/` in `src/features/` would catch
violations but generates false positives (a feature importing
from itself: `auth/components/login-form.tsx` importing
`@/features/auth/api/login.ts`).

## Decision

We will enforce the cross-feature-import rule **mechanically via
ESLint** using `import/no-restricted-paths` zones. Every feature
folder is its own zone; only same-feature imports cross the
fence.

Concrete configuration in `web/eslint.config.mjs` and
`admin/eslint.config.mjs`:

```js
{
  files: ["src/features/**"],
  rules: {
    "import/no-restricted-paths": ["error", {
      zones: [
        {
          target: "./src/features/auth",
          from: "./src/features",
          except: ["./auth"],
          message: "Features must not import from other features..."
        },
        // ... one zone per feature ...
      ]
    }]
  }
}
```

The zone definitions are generated from the feature directory
listing; adding a new feature means adding one zone block.

CI (`.github/workflows/ci.yml`) runs `npx eslint .` on both
`web/` and `admin/`. A cross-feature import fails the build with
the explicit message "Features must not import from other
features. Extract shared logic to src/shared/."

We also document the **legitimate composition pattern** in
`web/CLAUDE.md`:

- A page that needs both billing and profile components imports
  both from their respective features and composes — but the
  page lives in `src/app/`, not in either feature.
- Truly shared logic (e.g. an `<Avatar>` primitive used by 5
  features) lives in `src/shared/components/`.

## Consequences

### Positive

- The rule is mechanical, not human. A new contributor cannot
  accidentally violate it on day one.
- Feature deletions remain a folder rm. We have done this twice
  in the project (a deferred feature was scrapped, another was
  scoped down) — both times the deletion was clean because no
  other feature reached in.
- The mental model is clear and cheap to learn: "is the import
  inside my feature folder? then OK." Beats a 3-page CONTRIBUTING
  section.
- Reviewers spend their time on the actual change, not on
  policing imports.

### Negative

- The ESLint zone list is verbose — one block per feature
  (currently 25+ blocks). We accept the verbosity; the rule is
  worth the lines.
- Adding a new feature requires adding its block to the zones.
  We document this in the `add-web-feature` slash command's
  template so the agent does it as part of scaffolding.
- Some genuine "shared" code temporarily lives in a feature
  before being promoted to `shared/`. The linter forces an early
  promotion decision, which is generally good but occasionally
  feels heavy for one-off helpers. We accept this tradeoff.

## Alternatives considered

- **Convention + code review only** — what we had. Failed on
  the third PR. Rejected.
- **Module federation / package boundaries** (yarn workspaces,
  npm workspaces) — proper enforcement but adds tooling
  complexity and a publish step. Overkill for an in-app modular
  layout.
- **Custom Babel/SWC plugin** — same effect as the ESLint rule
  but harder to debug. Rejected; the ESLint rule is well
  supported and the error message points to the right line.
- **Run-time check** (a wrapper around `import()`) — performance
  hit on every import. Rejected.

## References

- `web/eslint.config.mjs` lines 90+ — the
  `noCrossFeatureImports` block.
- `admin/eslint.config.mjs` — same shape.
- `web/CLAUDE.md` — feature isolation rule for contributors.
- `web/e2e/refactor-isolation.spec.ts` — Playwright contract
  test that confirms the runtime composition pattern still works.
- `CONTRIBUTING.md §3` — the "delete the folder" invariant.
- P9 commit chain in `git log --grep="P9"`: `36771e2a`,
  `8cba8965`, `c27ad952`, `d6f6c73a`.
