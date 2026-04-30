<!--
Thanks for opening a PR! Keep this template filled in — reviewers
need it. Delete the comment markers but leave the section headings.
-->

## Summary

<!--
1-3 bullets explaining the WHY, not the what. The diff already
shows the what.

Example:
- Adds brute-force lockout to /auth/login (SEC-07).
- 5 failures per email per 15min triggers a 30min lockout via Redis.
- Returns 429 with Retry-After so the client can back off.
-->

-

## Test plan

<!--
Checklist of what you ran or what a reviewer should run. Keep entries
specific (commands, paths) — "ran tests" is not enough. Tick what is
already green; leave unchecked what reviewers should run themselves.
-->

- [ ] `cd backend && go vet ./... && go test ./... -count=1 -race`
- [ ] `cd web && npx tsc --noEmit && npx vitest run`
- [ ] `cd admin && npx tsc --noEmit && npx vitest run` (if admin touched)
- [ ] `cd mobile && flutter analyze && flutter test` (if mobile touched)
- [ ] Manual smoke (describe what you clicked through):

## Linked issues

<!--
Reference any issues this PR closes. GitHub will auto-close them on
merge.

Examples:
Closes #123
Refs #456
-->

Closes #

## Risk and rollback

<!--
What can break and how to undo. Keep it short.

- DB migration: yes / no — and if yes, can it be reverted forward?
- Feature flag: yes / no — and if yes, what is the kill switch?
- External service touched: list (Stripe / LiveKit / Resend / etc.)
- Roll back via: revert this PR / disable feature flag / re-deploy
  previous tag.
-->

-

## Screenshots / recording (web + mobile changes)

<!--
Drag-drop screenshots or a short Loom for visual changes. Required
for any PR that affects pixels. Skip otherwise.
-->

---

<!--
Reminders before submitting:
- Conventional commits (feat / fix / refactor / chore / test / docs /
  perf / style / revert) — title matches the squash-merge subject.
- File size budget: 600 lines per file, 50 per function, 4 params.
- "Delete the folder = compiles" feature isolation invariant — see
  CONTRIBUTING.md §3 if your PR adds cross-feature imports.
- For security issues, do NOT open a public PR — see SECURITY.md.
-->
