package handler

import (
	"strings"
)

// operationFor builds an OpenAPI 3.1 `operationObject` for a (method,
// route) pair. The catalogue (see catalogueLookup below) supplies the
// curated tag, summary, request shape and response shape for known
// routes; everything else gets a generic shape so the caller can
// still address it via `paths["..."]`.
//
// The function never panics — an unknown route falls back to a sane
// default operation. This is by design: chi.Walk discovers every
// registered route, the catalogue is best-effort enrichment.
func operationFor(method, route string, registry *schemaRegistry) map[string]any {
	spec, found := catalogueLookup(method, route)
	if !found {
		spec = defaultSpecFor(method, route)
	}

	op := map[string]any{
		"tags":     spec.Tags,
		"summary":  spec.Summary,
		"operationId": operationID(method, route),
		"responses": buildResponses(spec, registry),
	}
	if len(spec.PathParams) > 0 {
		op["parameters"] = spec.PathParams
	}
	if spec.RequestBody != nil {
		op["requestBody"] = spec.RequestBody
	}
	// Authenticated routes — by middleware-count convention every chi
	// route under /api/v1/* with an `auth` middleware applied gets
	// the bearer/cookie security requirement. Public routes (auth
	// endpoints, invitations validate, profile lookup by id, etc.)
	// skip it. The catalogue is the source of truth so one place
	// owns the auth annotation.
	if spec.AuthRequired {
		withSecurity(op)
	}
	if spec.Description != "" {
		op["description"] = spec.Description
	}
	return op
}

// buildResponses composes the responses map, defaulting unknown
// success shapes to `rawJSONResponse` so generated clients still see
// a well-typed `application/json` body.
func buildResponses(spec routeSpec, _ *schemaRegistry) map[string]any {
	out := map[string]any{}
	switch spec.SuccessKind {
	case successJSONRef:
		out[spec.SuccessStatus] = successJSONContent(spec.SuccessRef)
	case successJSONList:
		out[spec.SuccessStatus] = map[string]any{
			"description": "OK",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"type":  "array",
						"items": map[string]any{"$ref": "#/components/schemas/" + spec.SuccessRef},
					},
				},
			},
		}
	case successNoContent:
		out[spec.SuccessStatus] = noContent()
	case successRawJSON:
		out[spec.SuccessStatus] = rawJSONResponse("OK")
	case successPDF:
		out[spec.SuccessStatus] = pdfBinaryResponse("PDF document")
	case successRedirect:
		out[spec.SuccessStatus] = emptyResponse("Redirect")
	default:
		out[spec.SuccessStatus] = rawJSONResponse("OK")
	}
	// Standard error envelope — every operation can return any of these.
	for code, body := range errorResponses() {
		if _, exists := out[code]; exists {
			continue
		}
		out[code] = body
	}
	return out
}

// operationID is a snake-case-ish unique identifier per operation.
// openapi-typescript does not require it, but Stoplight / Redoc and
// other tools render it as the human anchor for an endpoint.
func operationID(method, route string) string {
	id := strings.ToLower(method) + route
	id = strings.ReplaceAll(id, "/api/v1/", "/")
	replacer := strings.NewReplacer(
		"/", "_",
		"{", "",
		"}", "",
		"-", "_",
		":", "_",
	)
	id = replacer.Replace(id)
	id = strings.Trim(id, "_")
	return id
}

// successKind enumerates how an operation's 2xx response is described.
// Adding a new value here is a deliberate API-shape extension, never a
// silent change.
type successKind int

const (
	successRawJSON successKind = iota
	successJSONRef
	successJSONList
	successNoContent
	successPDF
	successRedirect
)

// routeSpec is the curated description of one endpoint. Every field
// is optional except Tags + Summary + SuccessStatus. The catalogue
// in catalogueLookup is dense enough to cover the 146 unique paths
// the F.3.2 web sweep targets; uncurated routes inherit defaults
// from defaultSpecFor.
type routeSpec struct {
	Tags          []string
	Summary       string
	Description   string
	AuthRequired  bool
	RequestBody   map[string]any
	SuccessKind   successKind
	SuccessStatus string
	SuccessRef    string
	PathParams    []map[string]any
}

// defaultSpecFor synthesises a routeSpec from method + route alone.
// Used when the catalogue has no entry — keeps every chi.Walk-emitted
// path describable.
func defaultSpecFor(method, route string) routeSpec {
	tag := tagFromRoute(route)
	auth := !isPublicRoute(method, route)
	successStatus := defaultSuccessStatus(method)
	kind := successRawJSON
	if successStatus == "204" {
		kind = successNoContent
	}
	return routeSpec{
		Tags:          []string{tag},
		Summary:       method + " " + route,
		AuthRequired:  auth,
		SuccessKind:   kind,
		SuccessStatus: successStatus,
	}
}

// tagFromRoute infers a default tag from the URL prefix. Falls back
// to "misc" when the prefix is not recognised. The catalogue overrides
// this for every known route, so the inference only matters for
// uncurated leaves.
func tagFromRoute(route string) string {
	switch {
	case strings.HasPrefix(route, "/api/v1/admin/"):
		return "admin"
	case strings.HasPrefix(route, "/api/v1/auth/"):
		return "auth"
	case strings.HasPrefix(route, "/api/v1/billing/") || strings.HasPrefix(route, "/api/v1/wallet/") ||
		strings.HasPrefix(route, "/api/v1/me/invoices") || strings.HasPrefix(route, "/api/v1/me/invoicing") ||
		strings.HasPrefix(route, "/api/v1/subscriptions") || strings.HasPrefix(route, "/api/v1/payment-info/"):
		return "billing"
	case strings.HasPrefix(route, "/api/v1/me/billing-profile"):
		return "billing-profile"
	case strings.HasPrefix(route, "/api/v1/calls/"):
		return "call"
	case strings.HasPrefix(route, "/api/v1/disputes"):
		return "dispute"
	case strings.HasPrefix(route, "/api/v1/me/account") || strings.HasPrefix(route, "/api/v1/me/export"):
		return "gdpr"
	case strings.HasPrefix(route, "/api/v1/jobs"):
		return "job"
	case strings.HasPrefix(route, "/api/v1/messaging/"):
		return "messaging"
	case strings.HasPrefix(route, "/api/v1/notifications"):
		return "notification"
	case strings.HasPrefix(route, "/api/v1/portfolio") || strings.HasPrefix(route, "/api/v1/profiles/{orgId}/project-history"):
		return "portfolio"
	case strings.HasPrefix(route, "/api/v1/proposals/") || strings.HasPrefix(route, "/api/v1/projects/"):
		return "proposal"
	case strings.HasPrefix(route, "/api/v1/profile") || strings.HasPrefix(route, "/api/v1/profiles") ||
		strings.HasPrefix(route, "/api/v1/clients/"):
		return "profile"
	case strings.HasPrefix(route, "/api/v1/freelance-profile") || strings.HasPrefix(route, "/api/v1/freelance-profiles"):
		return "freelance-profile"
	case strings.HasPrefix(route, "/api/v1/referrer-profile") || strings.HasPrefix(route, "/api/v1/referrer-profiles"):
		return "referrer-profile"
	case strings.HasPrefix(route, "/api/v1/referrals"):
		return "referral"
	case strings.HasPrefix(route, "/api/v1/reports"):
		return "report"
	case strings.HasPrefix(route, "/api/v1/reviews"):
		return "review"
	case strings.HasPrefix(route, "/api/v1/search"):
		return "search"
	case strings.HasPrefix(route, "/api/v1/skills"):
		return "skill"
	case strings.HasPrefix(route, "/api/v1/stripe"):
		return "stripe"
	case strings.HasPrefix(route, "/api/v1/test/"):
		return "test"
	case strings.HasPrefix(route, "/api/v1/upload/"):
		return "upload"
	case strings.HasPrefix(route, "/api/v1/invitations") ||
		strings.HasPrefix(route, "/api/v1/organizations/role-definitions") ||
		strings.HasPrefix(route, "/api/v1/organizations/{orgID}") ||
		strings.HasPrefix(route, "/api/v1/organization/"):
		return "team"
	case strings.HasPrefix(route, "/api/v1/organization/shared"):
		return "organization-shared"
	case route == "/api/v1/ws":
		return "websocket"
	case route == "/health" || route == "/ready":
		return "health"
	default:
		return "misc"
	}
}

// isPublicRoute returns true when the route is reachable without an
// access token. The set is small and explicit — every entry is a
// route the auth middleware does NOT touch.
func isPublicRoute(method, route string) bool {
	switch method + " " + route {
	case "POST /api/v1/auth/register",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/forgot-password",
		"POST /api/v1/auth/reset-password",
		"GET /api/v1/invitations/validate",
		"POST /api/v1/invitations/accept",
		"POST /api/v1/stripe/webhook",
		"GET /health",
		"GET /ready",
		"GET /api/v1/test/health-check",
		"GET /api/v1/test/words",
		"POST /api/v1/test/words",
		"GET /api/v1/ws":
		return false
	}
	// Public read profile / search routes (no auth middleware applied).
	publicPrefixes := []string{
		"GET /api/v1/profiles/",
		"GET /api/v1/clients/",
		"GET /api/v1/freelance-profiles/",
		"GET /api/v1/referrer-profiles/",
		"GET /api/v1/portfolio/org/",
		"GET /api/v1/portfolio/{id}",
		"GET /api/v1/reviews/average/",
		"GET /api/v1/reviews/org/",
		"GET /api/v1/skills/autocomplete",
		"GET /api/v1/skills/catalog",
	}
	full := method + " " + route
	for _, p := range publicPrefixes {
		if strings.HasPrefix(full, p) {
			return false
		}
	}
	return true
}

// defaultSuccessStatus picks the typical successful status for a
// method. This lines up with chi's idiomatic responses across the
// codebase: GET 200, POST 201/200 (catalogue can override), PUT 200,
// PATCH 200, DELETE 204.
func defaultSuccessStatus(method string) string {
	switch method {
	case "GET":
		return "200"
	case "POST":
		return "201"
	case "PUT", "PATCH":
		return "200"
	case "DELETE":
		return "204"
	}
	return "200"
}

// catalogueLookup returns the curated routeSpec for a (method, route)
// pair. Hits the dense map produced by buildCatalogue() — the cost is
// one map lookup per route at boot time.
func catalogueLookup(method, route string) (routeSpec, bool) {
	if spec, ok := catalogue[method+" "+route]; ok {
		return spec, true
	}
	return routeSpec{}, false
}

// catalogue is the densely-populated curated route → spec map. Built
// lazily once the package initializes (see init.go).
var catalogue = buildCatalogue()
