// Package system carries cross-cutting context helpers for code
// paths that legitimately run outside the per-request, per-tenant
// authentication envelope.
//
// Today the only such caller class is background schedulers
// (proposal auto-approve / fund-reminder / auto-close, dispute
// auto-resolve, AI summary worker, payout retry). Every other
// path is user-driven and MUST go through the standard
// middleware.MustGetOrgID gate before touching an RLS-protected
// repository read.
//
// The intent is twofold:
//
//  1. **Mark the boundary explicitly.** A scheduler entrypoint
//     wraps its goroutine context with WithSystemActor before
//     calling into the application services. Repository methods
//     that bypass the tenant gate (the legacy GetByID on RLS
//     tables) gate themselves on IsSystemActor — accidentally
//     calling them from a user-facing handler will panic at
//     boot time during the wiring phase rather than silently
//     leak rows in production.
//
//  2. **Pave the way for a privileged DB pool.** Today the
//     migration role bypasses RLS. Once production rotates to a
//     non-superuser application role, system-actor connections
//     need a *separate* pool with BYPASSRLS or a co-resident
//     superuser. The boundary marker is already in place; only
//     the pool selection inside the adapter needs to change.
package system

import "context"

// systemActorKey is the unexported context key. Pure type — no
// instances are ever constructed; the address-of-the-zero-value
// is stable so context.WithValue / Value match correctly.
type systemActorKey struct{}

// WithSystemActor returns a child context tagged as a
// system-actor caller. Repository methods that legitimately run
// without a tenant gate (the legacy GetByID on RLS tables) will
// honor the tag and skip the tenant context setter; every other
// caller is required to go through the GetByIDForOrg variant.
//
// Schedulers and background workers MUST wrap their goroutine
// context with WithSystemActor at startup. User-facing handlers
// MUST NOT — calling this from a request path silently disables
// the RLS guardrail and is a bug.
func WithSystemActor(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemActorKey{}, true)
}

// IsSystemActor reports whether the context was tagged with
// WithSystemActor at some ancestor scope. Repository code uses
// this as an explicit guard before honoring the legacy
// non-tenant-aware code path.
//
// Safe to call on any context, including the background root —
// returns false when the key was never set.
func IsSystemActor(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(systemActorKey{}).(bool)
	return v
}
