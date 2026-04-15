package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// scoped_client.go exposes a persona-scoped wrapper around *Client that
// forces every query to include a `persona:<value> && is_published:true`
// filter by construction. This is the backend-side equivalent of the
// scoped Typesense API key we will hand to the frontend in phase 2 —
// both layers enforce the same invariant, because defense in depth
// means never trusting a single choke point.
//
// The contract: given a PersonaScopedClient for freelancers, even a
// caller who explicitly passes `filter_by: persona:agency` in the
// search params can NEVER reach an agency document. The scoped
// client prepends its own persona filter in front of the caller's
// string, and Typesense evaluates them in conjunction — a document
// must satisfy BOTH clauses to be returned.
//
// This file is covered by scoped_client_test.go with leak scenarios.

// PersonaScopedClient is a read-only view into a single persona's
// slice of the index. Constructed once at wiring time and passed
// down into handlers; callers never hold a bare *Client reference
// on the query path.
type PersonaScopedClient struct {
	client  *Client
	persona Persona
}

// NewFreelanceClient wraps a raw client into a freelance-only
// scoped view. Panics if the inner client is nil to fail fast at
// boot — there is no sensible "empty" scoped client.
func NewFreelanceClient(c *Client) *PersonaScopedClient {
	return newScopedClient(c, PersonaFreelance)
}

// NewAgencyClient wraps a raw client into an agency-only scoped view.
func NewAgencyClient(c *Client) *PersonaScopedClient {
	return newScopedClient(c, PersonaAgency)
}

// NewReferrerClient wraps a raw client into a referrer-only scoped view.
func NewReferrerClient(c *Client) *PersonaScopedClient {
	return newScopedClient(c, PersonaReferrer)
}

// newScopedClient is the private factory so each persona constructor
// fails fast on a nil client without duplicating the check.
func newScopedClient(c *Client, persona Persona) *PersonaScopedClient {
	if c == nil {
		panic("search: cannot create scoped client on nil *Client")
	}
	if !persona.IsValid() {
		// Safety net: the package-level constructors pass valid
		// constants, so a panic here only triggers on a
		// programmer error adding a new persona without updating
		// IsValid.
		panic(fmt.Sprintf("search: cannot create scoped client for invalid persona %q", persona))
	}
	return &PersonaScopedClient{client: c, persona: persona}
}

// Persona returns the persona this client is scoped to. Useful for
// logging and for phase-2 handlers that want to echo the scope back
// to the frontend.
func (p *PersonaScopedClient) Persona() Persona { return p.persona }

// Query runs the search against the alias, prepending the mandatory
// `persona:X && is_published:true` filter clause to the caller's
// own filter_by. Even if the caller tries to override persona, the
// AND composition guarantees only documents matching BOTH clauses
// come back.
func (p *PersonaScopedClient) Query(ctx context.Context, params SearchParams) (json.RawMessage, error) {
	scoped := params
	scoped.FilterBy = p.composeFilter(params.FilterBy)
	return p.client.Query(ctx, AliasName, scoped)
}

// composeFilter returns `persona:<persona> && is_published:true &&
// (user_filter)`. The user_filter is wrapped in parentheses so its
// own ORs cannot sneak around the persona clause via operator
// precedence (`A && B || C` evaluates as `(A && B) || C`, which
// would let a `persona:agency || …` clause escape).
//
// We intentionally do NOT try to parse the user's filter — any
// attempt at whitelisting is both fragile and unnecessary, because
// the && composition is mathematically sufficient.
func (p *PersonaScopedClient) composeFilter(userFilter string) string {
	base := fmt.Sprintf("persona:%s && is_published:true", p.persona)
	trimmed := strings.TrimSpace(userFilter)
	if trimmed == "" {
		return base
	}
	return fmt.Sprintf("%s && (%s)", base, trimmed)
}
