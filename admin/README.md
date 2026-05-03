# Admin Panel

Vite 7 + React 19 + Tailwind 4 admin dashboard for the marketplace.

This app keeps its own `node_modules`. **There is no root install and no
hoisting from the monorepo** — you must run `npm install` from this
directory the first time you clone the repo. Failing to do so causes
`npm run dev`, `npm run build`, and `npx vitest run` to fail with errors
like "Failed to resolve import zustand" because the dependencies declared
in `admin/package.json` (zustand, others) are not yet present in
`admin/node_modules/`.

## Quickstart

```bash
# From the repo root
cd admin

# 1. First-time install — MANDATORY before any other command
npm install

# 2. Start the dev server (http://localhost:5174)
npm run dev
```

## Daily commands

```bash
npm run dev          # Dev server on port 5174
npm run build        # tsc -b + Vite production build
npm run preview      # Serve the production build locally
npm run lint         # ESLint
npm run generate-api # Regenerate types from backend OpenAPI schema
npx vitest run       # Run unit tests
npx tsc -b --noEmit  # Type-check without emitting
```

## Environment

The app reads `VITE_API_URL` (default `http://localhost:8083`) at build
time. Override it in `admin/.env.local` for non-default backend ports.

## Docs

- [`admin/CLAUDE.md`](./CLAUDE.md) — architecture, conventions, layered
  structure, and per-feature wiring rules.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — repo-wide validation
  pipeline, branch naming, and PR rules.
- [`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md) — full system
  architecture including how the admin app fits into the stack.
