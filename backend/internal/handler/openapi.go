package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	res "marketplace-backend/pkg/response"
)

// OpenAPIVersion is the OpenAPI specification version emitted by the
// schema builder. We target 3.1 because openapi-typescript fully supports
// it and 3.1 aligns the schema dialect with JSON Schema 2020-12 — letting
// us reuse the reflection-driven generator (see openapi_schemas.go).
const OpenAPIVersion = "3.1.0"

// APITitle and APIVersion are exposed via the OpenAPI `info` block. The
// version is bumped explicitly when we cut a new public release of the
// contract; non-breaking additions do not require a bump per the
// CLAUDE.md versioning rules.
const (
	APITitle       = "Marketplace API"
	APIDescription = "Open-source B2B marketplace API connecting agencies, freelancers, enterprises, and business referrers."
	APIVersion     = "1.0.0"
)

// OpenAPIDocument is the top-level OpenAPI 3.1 document we serve from
// GET /api/openapi.json. Built ONCE at boot time from the live chi
// router (so the path list is drift-proof — every registered route is
// described, no separate manifest to keep in sync) plus a curated
// catalogue of DTO references (see openapi_routes.go).
//
// We deliberately model the document as `map[string]any` instead of
// importing the heavyweight getkin/kin-openapi library. Reasons:
//   - the schema we emit is small enough that a typed tree adds noise
//     without value (every leaf is one of: string, int, []string, map);
//   - we already serialize Go structs to JSON everywhere — so the
//     reflection-driven schema generator (openapi_schemas.go) already
//     understands the same encoding rules openapi-typescript expects;
//   - keeping the dep list small matches the brief's "no new
//     dependencies unless absolutely required" rule.
type OpenAPIDocument map[string]any

// BuildOpenAPIDocument walks the chi tree on the provided router and
// composes the OpenAPI 3.1 document. It is deterministic — calling it
// twice on the same router returns the same document — so the
// schema can be cached at boot time and served as-is on every
// /api/openapi.json hit.
//
// The function never panics: a route the catalogue does not know
// about is still emitted with a generic shape (any/any). This means
// every chi.Walk path appears in the schema, which is exactly what
// the F.3.2 typing sweep needs to refer to via `paths["..."]`.
func BuildOpenAPIDocument(router chi.Router) OpenAPIDocument {
	paths := map[string]map[string]any{}
	registry := newSchemaRegistry()

	_ = chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// Skip the OpenAPI endpoint itself — describing the schema
		// inside the schema is recursive and not useful to consumers.
		if route == "/api/openapi.json" || route == "/api/v1/openapi.json" {
			return nil
		}
		// Skip the Prometheus scrape endpoint (text/plain, not JSON).
		if route == "/metrics" {
			return nil
		}
		// Normalize chi's `{param}` to OpenAPI's `{param}` (already the
		// same syntax) — kept here as a no-op so any future divergence
		// is caught in one place.
		oapiPath := chiPathToOpenAPI(route)
		op := operationFor(method, route, registry)

		methodKey := strings.ToLower(method)
		registerPath := func(p string) {
			if _, ok := paths[p]; !ok {
				paths[p] = map[string]any{}
			}
			if _, ok := paths[p]["parameters"]; !ok {
				if params := pathParamsFor(route); len(params) > 0 {
					paths[p]["parameters"] = params
				}
			}
			paths[p][methodKey] = op
		}
		registerPath(oapiPath)
		// chi's `r.Route("/x", func(r) { r.Get("/", h) })` registers
		// the canonical path as `/x/` (trailing slash) but ALSO accepts
		// requests to `/x` (no slash) — both return 200. Expose the
		// alias in the OpenAPI document so consumers that historically
		// hit the un-slashed form (the entire web app does) get a
		// matching path entry. Without this, `paths["/api/v1/profile"]`
		// would be undefined even though the runtime call works.
		if alt := alternateSlashPath(oapiPath); alt != "" {
			registerPath(alt)
		}
		return nil
	})

	return OpenAPIDocument{
		"openapi": OpenAPIVersion,
		"info": map[string]any{
			"title":       APITitle,
			"description": APIDescription,
			"version":     APIVersion,
			"license": map[string]any{
				"name": "MIT",
				"url":  "https://opensource.org/license/mit/",
			},
		},
		"servers": []map[string]any{
			{"url": "/", "description": "Same-origin (proxy in production, direct in dev)"},
			{"url": "http://localhost:8083", "description": "Local development backend"},
		},
		"tags":  buildTags(),
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
					"description":  "Mobile/admin clients send Authorization: Bearer <access_token>",
				},
				"sessionCookie": map[string]any{
					"type":        "apiKey",
					"in":          "cookie",
					"name":        "session",
					"description": "Web browser session — server-set, http-only, same-site=Lax",
				},
			},
			"schemas": registry.export(),
		},
	}
}

// alternateSlashPath returns a sibling path with the trailing slash
// flipped — `/foo/` ↔ `/foo`. Used to expose both forms in the
// OpenAPI document because chi's r.Route group + r.Get("/") accepts
// callers using either form. Returns "" when no alternate makes sense
// (root paths, paths with no slash to flip, or query-string-bearing
// paths the OpenAPI doc never describes).
//
// We do NOT flip when the path is the bare root "/" — there is no
// alternate.
func alternateSlashPath(p string) string {
	if p == "" || p == "/" {
		return ""
	}
	if strings.HasSuffix(p, "/") {
		return strings.TrimRight(p, "/")
	}
	return p + "/"
}

// chiPathToOpenAPI converts a chi route template to an OpenAPI path
// template. chi uses `{name}` natively which is also OpenAPI's syntax,
// so the function is a pass-through today — kept so that any future
// divergence (e.g. chi regex constraints `{id:[0-9]+}`) is centralized.
func chiPathToOpenAPI(route string) string {
	if !strings.Contains(route, "{") {
		return route
	}
	// Strip chi regex constraints: {id:[0-9]+} -> {id}
	var b strings.Builder
	i := 0
	for i < len(route) {
		ch := route[i]
		if ch != '{' {
			b.WriteByte(ch)
			i++
			continue
		}
		end := strings.IndexByte(route[i:], '}')
		if end < 0 {
			b.WriteString(route[i:])
			break
		}
		segment := route[i+1 : i+end]
		if colon := strings.IndexByte(segment, ':'); colon >= 0 {
			segment = segment[:colon]
		}
		b.WriteByte('{')
		b.WriteString(segment)
		b.WriteByte('}')
		i += end + 1
	}
	return b.String()
}

// pathParamsFor collects every `{param}` substring of a chi route and
// emits an OpenAPI `parameters` entry (path / required / string).
// chi never types its path params beyond "string" so the OpenAPI
// schema mirrors that — UUIDs, slugs, numeric ids alike are described
// as strings; the handler is responsible for parsing them.
func pathParamsFor(route string) []map[string]any {
	var out []map[string]any
	i := 0
	for i < len(route) {
		ch := route[i]
		if ch != '{' {
			i++
			continue
		}
		end := strings.IndexByte(route[i:], '}')
		if end < 0 {
			break
		}
		segment := route[i+1 : i+end]
		if colon := strings.IndexByte(segment, ':'); colon >= 0 {
			segment = segment[:colon]
		}
		out = append(out, map[string]any{
			"name":     segment,
			"in":       "path",
			"required": true,
			"schema":   map[string]any{"type": "string"},
		})
		i += end + 1
	}
	return out
}

// buildTags returns a deterministic, alphabetised list of tags. Each
// tag is documented once here so the consumer sees a clean grouping in
// generated docs (Stoplight, Redoc, Swagger UI).
func buildTags() []map[string]any {
	tags := []map[string]any{
		{"name": "auth", "description": "Authentication, session, and role management."},
		{"name": "team", "description": "Organizations, members, invitations, transfers."},
		{"name": "profile", "description": "Agency profile, pricing, social links, skills."},
		{"name": "freelance-profile", "description": "Provider personal profile (split-profile model)."},
		{"name": "referrer-profile", "description": "Business referrer (apporteur) profile."},
		{"name": "client-profile", "description": "Client (enterprise) profile facet."},
		{"name": "organization-shared", "description": "Org-level shared fields (location, languages, photo)."},
		{"name": "search", "description": "Typesense-backed search and click tracking."},
		{"name": "messaging", "description": "Conversations and messages between principals."},
		{"name": "proposal", "description": "Proposals, milestones, completion lifecycle."},
		{"name": "job", "description": "Jobs, applications, credits, viewing history."},
		{"name": "review", "description": "Reviews and review averages per organization."},
		{"name": "report", "description": "Abuse / spam reports for moderation."},
		{"name": "social-link", "description": "Public social links displayed on profiles."},
		{"name": "portfolio", "description": "Portfolio items (case studies)."},
		{"name": "notification", "description": "In-app notifications and preferences."},
		{"name": "billing", "description": "Wallet, subscriptions, invoices, payouts, KYC."},
		{"name": "billing-profile", "description": "Customer billing profile (VAT, address, default payment method)."},
		{"name": "referral", "description": "Apporteur referrals and commission negotiation."},
		{"name": "dispute", "description": "Dispute opening, counter-proposals, AI mediator."},
		{"name": "gdpr", "description": "RGPD: data export, account deletion request/confirm/cancel."},
		{"name": "skill", "description": "Skill catalog and autocomplete."},
		{"name": "upload", "description": "Direct-to-MinIO uploads (photo/video/portfolio)."},
		{"name": "call", "description": "LiveKit voice call signaling."},
		{"name": "stripe", "description": "Stripe Connect Embedded Components."},
		{"name": "admin", "description": "Admin endpoints — moderation, dashboard, dispute resolution."},
		{"name": "health", "description": "Liveness / readiness probes."},
		{"name": "test", "description": "Development-only health and word endpoints."},
		{"name": "websocket", "description": "WebSocket upgrade endpoint for real-time messaging."},
	}
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i]["name"].(string) < tags[j]["name"].(string)
	})
	return tags
}

// errorResponseRef returns the standard error envelope reference so
// every operation reuses the same component. The envelope shape is
// pinned to the production handler (pkg/response.Error / NotFound /
// ValidationError) — see registerErrorEnvelope() in openapi_schemas.go.
func errorResponseRef() map[string]any {
	return map[string]any{
		"description": "Error envelope",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/ErrorResponse"},
			},
		},
	}
}

// successJSONContent wraps a schema reference into an OpenAPI 200
// response envelope. The wrapper is small but repeated on every
// successful operation; centralising it keeps the operation builder
// terse.
func successJSONContent(ref string) map[string]any {
	return map[string]any{
		"description": "OK",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/" + ref},
			},
		},
	}
}

// emptyResponse is used for 204 / "no content" responses — the
// JSON content stanza is omitted entirely so generated clients don't
// expect a body.
func emptyResponse(description string) map[string]any {
	return map[string]any{"description": description}
}

// rawJSONResponse is for operations whose response is a JSON object
// without a curated DTO yet — the schema is left "open" (unconstrained
// object) so generated clients still get a valid shape and downstream
// callers can refine via TypeScript intersections.
func rawJSONResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"type": "object", "additionalProperties": true},
			},
		},
	}
}

// rawJSONRequestBody mirrors rawJSONResponse for un-curated request
// bodies. The handler still validates the body — this entry is just
// the contract description.
func rawJSONRequestBody() map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"type": "object", "additionalProperties": true},
			},
		},
	}
}

// jsonRequestBody references a curated request DTO by component name.
func jsonRequestBody(ref string) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/" + ref},
			},
		},
	}
}

// multipartFormRequestBody describes file uploads. We declare the
// `file` part explicitly so generated clients know to use FormData
// instead of JSON.
func multipartFormRequestBody() map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"multipart/form-data": map[string]any{
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file": map[string]any{
							"type":   "string",
							"format": "binary",
						},
					},
					"required": []string{"file"},
				},
			},
		},
	}
}

// pdfBinaryResponse describes endpoints that return application/pdf
// (invoice PDF redirect / passthrough). The body is binary; clients
// stream it.
func pdfBinaryResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/pdf": map[string]any{
				"schema": map[string]any{"type": "string", "format": "binary"},
			},
		},
	}
}

// noContent is shorthand for a 204 with no body.
func noContent() map[string]any {
	return map[string]any{"description": "No content"}
}

// errorResponses returns the standard error response set. Keeping the
// list short and consistent across endpoints simplifies client
// generation — every error envelope is the same shape.
func errorResponses() map[string]any {
	return map[string]any{
		"400": errorResponseRef(),
		"401": errorResponseRef(),
		"403": errorResponseRef(),
		"404": errorResponseRef(),
		"409": errorResponseRef(),
		"422": errorResponseRef(),
		"429": errorResponseRef(),
		"500": errorResponseRef(),
	}
}

// withSecurity attaches the bearer + cookie security requirement to an
// operation. We always emit BOTH (alternative requirements) — a client
// must satisfy at least one. Public operations skip this entirely.
func withSecurity(op map[string]any) map[string]any {
	op["security"] = []map[string]any{
		{"bearerAuth": []string{}},
		{"sessionCookie": []string{}},
	}
	return op
}

// ServeOpenAPIHandler returns an http.Handler that serves the cached
// OpenAPI document as JSON. The document is built ONCE on the first
// call (lazily) so the cost of reflection-driven schema discovery is
// paid out of the critical path; subsequent calls are pure
// `json.Marshal` (cached even further inside the handler).
//
// The endpoint is public, no auth, cache-control 5min — matching the
// brief.
func ServeOpenAPIHandler(router chi.Router) http.HandlerFunc {
	doc := BuildOpenAPIDocument(router)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		res.JSON(w, http.StatusOK, doc)
	}
}
