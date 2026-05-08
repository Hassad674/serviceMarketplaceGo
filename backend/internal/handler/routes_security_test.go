package handler

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestMountSecurityRoutes_Registered guards the wiring of the
// /me/security/activity endpoint. The original report (account/security
// tab "Impossible de charger l'activité") came from a stale local
// backend that did not include this route — chi then fell through to
// the /me sub-router mounted by the billing helper and the request
// inherited its RequirePermission(billing.view) middleware, returning
// 403 no_organization to the user.
//
// This test catches the regression in two flavors:
//
//  1. mountSecurityRoutes alone registers exactly GET
//     /me/security/activity (and nothing else). A zero count means the
//     wiring short-circuited (deps.Security == nil) and the symptom
//     above will reappear in production once a build forgets the
//     wireSecurity() call.
//
//  2. The full router builds the route with strictly fewer middlewares
//     than /me/invoices (which carries RequirePermission(billing.view)
//     on top of the same auth + nocache + global stack). If the count
//     ever matches /me/invoices, someone has accidentally moved the
//     security route under the /me sub-router and the org-less feature
//     is broken for every user without an organization (most providers).
func TestMountSecurityRoutes_Registered(t *testing.T) {
	deps := snapshotDeps()
	auth := func(next http.Handler) http.Handler { return next }

	r := chi.NewRouter()
	mountSecurityRoutes(r, deps, auth)

	got := collectRoutes(t, r)
	if len(got) != 1 {
		t.Fatalf("mountSecurityRoutes registered %d routes, want exactly 1: %v", len(got), got)
	}
	if want := "GET /me/security/activity"; got[0] != want {
		t.Errorf("mountSecurityRoutes registered %q, want %q", got[0], want)
	}
}

// TestMountSecurityRoutes_NotInheritingMeMiddleware is the regression
// guard for the chi sub-router middleware inheritance bug that caused
// the original /fr/account?section=security 403. It builds the full
// router (so the /me sub-router from the billing helper coexists with
// /me/security) and asserts the security route's middleware count is
// strictly smaller than /me/invoices's. /me/invoices carries:
//
//	global middlewares + auth + nocache + RequirePermission(billing.view)
//
// /me/security/activity must NOT carry RequirePermission. If the two
// counts ever match, the feature is broken for every user who is not
// part of an organization — which is the default state for a fresh
// signup and for solo providers.
func TestMountSecurityRoutes_NotInheritingMeMiddleware(t *testing.T) {
	r := NewRouter(snapshotDeps())

	mwCount := map[string]int{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		mwCount[method+" "+route] = len(mws)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	const securityKey = "GET /api/v1/me/security/activity"
	const invoicesKey = "GET /api/v1/me/invoices"

	securityMW, secOK := mwCount[securityKey]
	invoicesMW, invOK := mwCount[invoicesKey]

	if !secOK {
		t.Fatalf("%s is not registered — security feature is unwired", securityKey)
	}
	if !invOK {
		t.Fatalf("%s is not registered — billing feature is unwired (test setup issue)", invoicesKey)
	}
	if securityMW >= invoicesMW {
		t.Errorf("/me/security/activity has %d middlewares, /me/invoices has %d — "+
			"security route is inheriting the billing /me sub-router's RequirePermission. "+
			"This breaks the feature for every user without an organization.",
			securityMW, invoicesMW)
	}
}

// collectRoutes returns the registered (METHOD path) tuples for the
// given router. Sister to countRoutes in routes_helpers_test.go but
// preserves the route strings so the security-routing assertion above
// can pin the exact path.
func collectRoutes(t *testing.T, r chi.Router) []string {
	t.Helper()
	var routes []string
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes = append(routes, method+" "+route)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	return routes
}
