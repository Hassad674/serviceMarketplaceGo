# Architecture Decision Records

This directory holds the project's [Architecture Decision Records](https://github.com/joelparkerhenderson/architecture-decision-record)
in the [Michael Nygard format](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions).

An ADR captures **one** significant architectural decision: the
context that forced the choice, the decision itself, the
consequences good and bad, and the alternatives that were considered
and rejected. Future contributors who wonder "why is this *that*
way?" should find their answer here without spelunking through
commit history.

## When to write a new ADR

Write an ADR when a decision:

- Affects more than one feature or layer (cross-cutting concern).
- Has consequences that will outlive a single sprint.
- Was contentious and has rejected alternatives worth documenting.
- Reverses or supersedes a previous ADR.

A new ADR is **not** required for tactical choices (which lib for
a one-off util, which color for the primary button). Those live in
code reviews and component specs.

## File naming

```
NNNN-kebab-case-title.md
```

`NNNN` is the next free 4-digit number. Numbers are never reused —
even if an ADR is superseded, its file stays.

## Status values

- `Proposed` — under discussion, not yet adopted.
- `Accepted` — in force; the codebase reflects this.
- `Deprecated` — the codebase no longer follows this, but the file
  documents what was once true.
- `Superseded by NNNN` — a newer ADR replaced this one. The link is
  bidirectional (the newer ADR cites this one as supersedes).

## Index

| #    | Title                                                   | Status   | Date       |
|------|---------------------------------------------------------|----------|------------|
| 0001 | [Hexagonal architecture (ports and adapters)](0001-hexagonal-architecture.md) | Accepted | 2026-04-30 |
| 0002 | [Org-scoped business state](0002-org-scoped-business-state.md) | Accepted | 2026-04-30 |
| 0003 | [PostgreSQL Row-Level Security with system-actor split](0003-postgresql-rls-with-system-actor.md) | Accepted | 2026-04-30 |
| 0004 | [Async Stripe webhooks via the pending\_events outbox](0004-stripe-webhooks-async-via-outbox.md) | Accepted | 2026-04-30 |
| 0005 | [OpenTelemetry OTLP exporter, no vendor lock-in](0005-opentelemetry-otlp-no-vendor-lock-in.md) | Accepted | 2026-04-30 |
| 0006 | [Feature isolation enforced by ESLint, not convention](0006-feature-isolation-no-cross-feature-imports.md) | Accepted | 2026-04-30 |
| 0007 | [Soft-delete with a 30-day RGPD recovery window](0007-soft-delete-30-days-rgpd-window.md) | Accepted | 2026-04-30 |
| 0008 | [Cursor pagination over OFFSET](0008-cursor-pagination-no-offset.md) | Accepted | 2026-04-30 |

## Template

When adding an ADR, copy this skeleton:

```markdown
# NNNN. Title in imperative mood

Date: YYYY-MM-DD

## Status

Accepted

## Context

The forces at play. What is the problem? What constraints (technical,
business, legal) shape the decision? Cite real code paths, real
metrics, real bugs that drove the discussion.

## Decision

The decision in imperative voice. "We will use X, gated by Y, with
Z as the escape hatch." One paragraph. Specific.

## Consequences

What becomes easier? What becomes harder? What new operational
costs are accepted? Cite any compensating controls.

### Positive

- ...

### Negative

- ...

## Alternatives considered

What else was on the table, why it lost. One paragraph per option,
ending with the disqualifier.

## References

- Links to code, migrations, prior PRs, external articles.
```
