package search_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

// TestScopedClient_ForcesPersonaFilter verifies every scoped client
// constructor injects its persona filter into the outgoing request,
// regardless of what the caller passes in filter_by.
func TestScopedClient_ForcesPersonaFilter(t *testing.T) {
	cases := []struct {
		name       string
		build      func(*search.Client) *search.PersonaScopedClient
		userFilter string
		wantFilter string
	}{
		{
			name:       "freelance empty user filter",
			build:      search.NewFreelanceClient,
			userFilter: "",
			wantFilter: "persona:freelance && is_published:true",
		},
		{
			name:       "agency with rating filter",
			build:      search.NewAgencyClient,
			userFilter: "rating_average:>=4",
			wantFilter: "persona:agency && is_published:true && (rating_average:>=4)",
		},
		{
			name:       "referrer with complex filter",
			build:      search.NewReferrerClient,
			userFilter: "skills:[React] && city:Paris",
			wantFilter: "persona:referrer && is_published:true && (skills:[React] && city:Paris)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotFilter string
			client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotFilter = r.URL.Query().Get("filter_by")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"hits":[]}`))
			}))

			scoped := c.build(client)
			_, err := scoped.Query(context.Background(), search.SearchParams{
				Q:        "anything",
				QueryBy:  "title",
				FilterBy: c.userFilter,
			})
			require.NoError(t, err)
			assert.Equal(t, c.wantFilter, gotFilter)
		})
	}
}

// TestScopedClient_CannotLeakViaOverride is the critical security
// test. A caller tries to bypass the freelance scope by passing
// `persona:agency` in their own filter_by. The scoped client MUST
// still enforce persona:freelance, so the effective filter is
// `persona:freelance && is_published:true && (persona:agency)`.
// That composition is unsatisfiable — no document has two personas —
// so Typesense returns zero results. The key test here is that the
// scoped client NEVER drops its own persona clause.
func TestScopedClient_CannotLeakViaOverride(t *testing.T) {
	var gotFilter string
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter_by")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":[]}`))
	}))

	scoped := search.NewFreelanceClient(client)
	_, err := scoped.Query(context.Background(), search.SearchParams{
		Q:        "malicious",
		QueryBy:  "title",
		FilterBy: "persona:agency", // attempt to bypass
	})
	require.NoError(t, err)

	// The freelance clause is still present and comes FIRST, so
	// Typesense evaluates both conditions in conjunction.
	assert.Contains(t, gotFilter, "persona:freelance && is_published:true")
	assert.Contains(t, gotFilter, "persona:agency")
	// Position matters: the injected clause must come BEFORE the
	// user clause so it cannot be accidentally dropped by future
	// query-parsing bugs.
	idxLegit := indexOf(gotFilter, "persona:freelance")
	idxBogus := indexOf(gotFilter, "persona:agency")
	assert.Less(t, idxLegit, idxBogus)
}

// TestScopedClient_ORCannotEscapePersonaClause guards against an
// attacker who understands precedence and tries `persona:agency ||
// true` to OR their way around. Wrapping the user filter in
// parentheses via composeFilter neutralises this.
func TestScopedClient_ORCannotEscapePersonaClause(t *testing.T) {
	var gotFilter string
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter_by")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":[]}`))
	}))

	scoped := search.NewFreelanceClient(client)
	_, err := scoped.Query(context.Background(), search.SearchParams{
		Q:        "x",
		QueryBy:  "title",
		FilterBy: "persona:agency || rating_average:>=0",
	})
	require.NoError(t, err)

	// The user clause MUST be wrapped in parentheses so the OR
	// binds tighter than the outer AND — otherwise a crafted OR
	// could short-circuit the persona guard.
	assert.Contains(t, gotFilter, "&& (persona:agency || rating_average:>=0)")
}

func TestScopedClient_PersonaAccessor(t *testing.T) {
	c, err := search.NewClient("http://localhost:8108", "k")
	require.NoError(t, err)

	assert.Equal(t, search.PersonaFreelance, search.NewFreelanceClient(c).Persona())
	assert.Equal(t, search.PersonaAgency, search.NewAgencyClient(c).Persona())
	assert.Equal(t, search.PersonaReferrer, search.NewReferrerClient(c).Persona())
}

func TestScopedClient_PanicOnNil(t *testing.T) {
	assert.Panics(t, func() { search.NewFreelanceClient(nil) })
}

// indexOf is a trivial helper used in the leak tests; inlined here
// to keep the test file self-contained.
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
