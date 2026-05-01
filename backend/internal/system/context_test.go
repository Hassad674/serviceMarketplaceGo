package system_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/system"
)

// TestIsSystemActor_DefaultFalse verifies an unmarked context is
// considered user-driven. This is the safe default — repository
// code that gates on IsSystemActor must NOT bypass the tenant
// gate when the marker is missing.
func TestIsSystemActor_DefaultFalse(t *testing.T) {
	assert.False(t, system.IsSystemActor(context.Background()),
		"unmarked context must be classified as user-driven")
}

// TestIsSystemActor_NilContext is the paranoia test — a nil
// context must not panic and must return false.
func TestIsSystemActor_NilContext(t *testing.T) {
	assert.False(t, system.IsSystemActor(nil),
		"nil context must default to false rather than panic")
}

// TestWithSystemActor_TagsContext verifies the tag is observable
// through IsSystemActor on the returned context.
func TestWithSystemActor_TagsContext(t *testing.T) {
	ctx := system.WithSystemActor(context.Background())
	assert.True(t, system.IsSystemActor(ctx),
		"context returned by WithSystemActor must be classified as system-actor")
}

// TestWithSystemActor_DoesNotAffectParent confirms the parent
// context is not mutated. Critical for safety: a child request
// branching off a tagged parent (rare but possible during
// goroutine fan-out) MUST be able to reason about its own scope
// without the parent's tag leaking unrelated.
func TestWithSystemActor_DoesNotAffectParent(t *testing.T) {
	parent := context.Background()
	_ = system.WithSystemActor(parent)
	assert.False(t, system.IsSystemActor(parent),
		"parent context must remain unmarked after WithSystemActor on a child")
}

// TestWithSystemActor_PropagatesThroughCancellation verifies the
// tag survives wrapping in context.WithCancel — schedulers
// commonly do this so they can stop the goroutine on shutdown.
func TestWithSystemActor_PropagatesThroughCancellation(t *testing.T) {
	parent := system.WithSystemActor(context.Background())
	child, cancel := context.WithCancel(parent)
	t.Cleanup(cancel)

	require.True(t, system.IsSystemActor(child),
		"WithSystemActor tag must propagate through context.WithCancel")
}

// TestWithSystemActor_PropagatesThroughValueChain verifies the
// tag survives subsequent context.WithValue calls. Schedulers
// often add request_id, trace_id, etc. to the system-actor
// context after the initial wrap.
func TestWithSystemActor_PropagatesThroughValueChain(t *testing.T) {
	type k string
	parent := system.WithSystemActor(context.Background())
	child := context.WithValue(parent, k("trace_id"), "abc123")

	assert.True(t, system.IsSystemActor(child),
		"WithSystemActor tag must propagate through context.WithValue")
	assert.Equal(t, "abc123", child.Value(k("trace_id")),
		"sibling context value must coexist with the system-actor tag")
}
