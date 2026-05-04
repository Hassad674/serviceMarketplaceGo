# Agent brief template — Web batch (desktop + responsive)

> Copy this file to `design/batches/BATCH-XXX-web-<topic>.md`, fill the variables marked `{{...}}`, then dispatch.

---

## Batch identity

- **ID**: `BATCH-XXX`
- **Surface**: Web (Next.js 16 — desktop + responsive in the same files via Tailwind breakpoints)
- **Topic**: `{{topic}}`
- **Worktree**: `/tmp/mp-design-web-{{batch-id}}`
- **Branch**: `feat/design-web-{{batch-id}}-{{topic}}`
- **Base**: `origin/main`

## Screens in this batch (from `design/inventory.md`)

- `{{inventory-id-1}}` — `{{screen-name-1}}` (desktop) + responsive variant
- `{{inventory-id-2}}` — `{{screen-name-2}}` (desktop) + responsive variant
- ...

## Source assets

For each screen, study:
- The Soleil JSX implementation: `design/assets/sources/phase1/{{file}}.jsx` lines `{{range}}`
- The desktop layout: `design/assets/pdf/web-desktop.pdf` page `{{N}}`
- The mobile/responsive layout: `design/assets/pdf/web-responsive.pdf` page `{{N}}`

The JSX is **inspiration only** — re-implement using the repo's idiomatic Tailwind classes. Inline styles in the JSX are NOT to be ported as inline styles in `.tsx`.

## Repo mapping

For each screen, the inventory entry lists:
- **Route existante**: `{{route}}`
- **Fichier principal**: `{{path-to-page.tsx}}`
- **Components touchables**: `{{whitelist}}`
- **API hooks (OFF-LIMITS)**: `{{list — these MUST NOT be edited}}`
- **Features design absentes du repo**: `{{list — SKIP these sections, FLAG in report}}`

---

# RULES

[Include verbatim from `design/agent-templates/shared-rules.md`]

# WEB-SPECIFIC RULES

## Tooling for this batch

- Tailwind 4 utility classes only. Tokens already defined in `web/src/styles/globals.css` (Phase 0 batch lands tokens — if your batch is a screen batch, the tokens are already there).
- `cn()` for conditional classes. `cva()` for component variants.
- Use the `Portrait` primitive at `web/src/shared/components/ui/portrait.tsx` (created in Phase 0). Never roll your own avatar.
- Icons: `lucide-react`. Mapping table in `design/DESIGN_SYSTEM.md` §8.
- Fonts: already loaded via `next/font`. Use Tailwind class `font-serif`, `font-sans`, `font-mono`.

## Server vs Client Components

- Default to Server Components.
- Add `"use client"` only when the screen genuinely needs `useState`, event handlers, or browser APIs. If your screen has a tabs control or modal: wrap ONLY that subtree in a client island, leave the surrounding page server.
- Do NOT downgrade an existing Server Component to a Client Component because the design "feels interactive". Most editorial/static surfaces stay server.

## Responsive

The same `.tsx` ships desktop + responsive. Use Tailwind breakpoints:
- Default = mobile-first (390px target)
- `md:` = 768px+
- `lg:` = 1024px+
- `xl:` = 1280px+
- `2xl:` = 1440px+ (the desktop maquette target)

The web-responsive PDF shows the 390px adaptation. Layout reorg is allowed (sidebar collapses to bottom tab bar, columns stack to single column, etc.) — but the visual identity stays Soleil.

## i18n

Every visible French string lives in `web/messages/fr.json`. Reuse keys when they exist. Add new keys at the end of the relevant namespace.

```tsx
const t = useTranslations("dashboard");
<h1>{t("greeting")}</h1>     // "Bonjour Nova, belle journée en perspective."
```

## OFF-LIMITS for web (in addition to shared list)

- `web/src/features/*/api/**.ts`
- `web/src/features/*/hooks/use-*.ts`
- `web/src/features/*/schemas/**.ts`
- `web/src/shared/lib/api-client.ts`, `api-paths.ts`
- `web/src/shared/types/api.d.ts`
- `web/middleware.ts`
- `web/next.config.ts`
- `web/package.json`, `package-lock.json`
- `web/e2e/**`
- All `*.test.ts*` files

## TOUCHABLE for web (typical batch)

- `web/src/app/<routegroup>/<route>/page.tsx`, `layout.tsx`, `loading.tsx`, `error.tsx`
- `web/src/features/<feature>/components/**.tsx` (UI components only — they may consume hooks but not redefine them)
- `web/src/styles/globals.css` (Phase 0 only — screen batches do NOT touch this)
- `web/messages/fr.json` (add i18n keys for new strings)

## Validation pipeline (web batch)

```bash
cd /tmp/mp-design-web-{{batch-id}}

# Build + types + tests
cd web && npx tsc --noEmit && npx vitest run --changed origin/main && cd ..

# Verify backend untouched
cd backend && go build ./... && go vet ./... && cd ..

# Run the design guardrails
design/scripts/validate-no-regression.sh

# Smoke test — start dev server and load the screens
cd web && npx next dev &
sleep 5
curl -s http://localhost:3000/{{route}} > /dev/null && echo "OK" || echo "FAIL"
kill %1
```

ALL of these must pass before you commit.

## Visual diff

For each screen:
1. Before you start, checkout `origin/main`, run dev server, screenshot at `design/diffs/{{screen-id}}/before-desktop.png` (1440x900) and `before-mobile.png` (390x844).
2. After your work, on your branch, screenshot the same routes → `after-desktop.png` and `after-mobile.png`.
3. Add `notes.md` describing intentional differences.

If you can't run the dev server (DB not seeded), document it in the report. The orchestrator will capture the screenshots.

## Push + PR

```bash
git push -u origin feat/design-web-{{batch-id}}-{{topic}}
gh pr create \
  --title "[design/web/{{batch-id}}] {{topic}}" \
  --body "$(cat <<'EOF'
## Batch summary
{{1-paragraph}}

## Screens shipped
- {{inventory-id-1}} {{screen-name-1}}
- ...

## Out-of-scope flagged
- ...

## Validation pipeline output
\`\`\`
{{paste full output}}
\`\`\`

## Files changed (whitelist check)
- Allowed: ...
- Off-limits: NONE

## Visual diffs
- design/diffs/{{screen-id}}/before-desktop.png ↔ after-desktop.png
- ...
EOF
)"
```

## Final report (in this batch file, after PR is open)

```markdown
## Outcome
- PR: #{{pr-number}}
- Status: in-review / merged / changes-requested
- Validation pipeline: PASS / FAIL
- Visual diffs reviewed by orchestrator: yes / no
- Screens marked done in tracking.md: yes / no
```
