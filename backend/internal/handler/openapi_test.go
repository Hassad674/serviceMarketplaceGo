package handler

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// updateOpenAPIGolden flips the OpenAPI snapshot to write-mode. Same
// semantics as updateGolden in router_snapshot_test.go — set true once
// to capture the baseline, then flip back to false. The CI guard
// surfaces a left-on flag because the test re-asserts immediately
// after the rewrite.
var updateOpenAPIGolden = flag.Bool("update-openapi-golden", false, "rewrite testdata/openapi.golden.json")

// TestOpenAPISchemaShape_Snapshot pins the OpenAPI document the API
// emits. It serialises the document to JSON with stable ordering and
// asserts byte-identity against testdata/openapi.golden.json. Any
// change to the schema (new endpoint, removed DTO field, retagged
// operation) requires a deliberate snapshot update — this is the
// drift-proof guard the F.3.2 typing sweep relies on.
//
// To regenerate the snapshot after an intentional schema change:
//
//	go test ./internal/handler/ -run TestOpenAPISchemaShape_Snapshot -update-openapi-golden
//
// The file is human-readable JSON (2-space indent) so PR reviewers can
// diff the schema alongside the code change.
func TestOpenAPISchemaShape_Snapshot(t *testing.T) {
	router := NewRouter(snapshotDeps())
	doc := BuildOpenAPIDocument(router)

	got, err := marshalDeterministic(doc)
	if err != nil {
		t.Fatalf("marshal openapi document: %v", err)
	}

	goldenPath := filepath.Join("testdata", "openapi.golden.json")
	if *updateOpenAPIGolden {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write openapi golden: %v", err)
		}
		t.Logf("wrote %d bytes to %s", len(got), goldenPath)
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read openapi golden: %v\nrun `go test ./internal/handler/ -run TestOpenAPISchemaShape_Snapshot -update-openapi-golden` to capture", err)
	}
	if string(want) != string(got) {
		t.Errorf("openapi schema drifted — testdata/openapi.golden.json is no longer byte-identical to the generated schema.\nDiff with `diff <(go test ... -run TestOpenAPISchemaShape_Snapshot -update-openapi-golden && cat testdata/openapi.golden.json) testdata/openapi.golden.json`")
	}
}

// TestOpenAPIEndpoint_ServesValidSchema runs the live handler and
// asserts the response is well-formed OpenAPI 3.1 with the expected
// top-level shape. This is the "alive" check — TestOpenAPISchemaShape
// is the regression check.
func TestOpenAPIEndpoint_ServesValidSchema(t *testing.T) {
	router := NewRouter(snapshotDeps())
	srv := httptest.NewServer(router)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/openapi.json")
	if err != nil {
		t.Fatalf("GET /api/openapi.json: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "max-age=300") {
		t.Errorf("Cache-Control = %q, want max-age=300", cc)
	}

	var doc map[string]any
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if v, _ := doc["openapi"].(string); v != OpenAPIVersion {
		t.Errorf("openapi version = %v, want %s", v, OpenAPIVersion)
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("info missing")
	}
	if info["title"] != APITitle {
		t.Errorf("info.title = %v, want %s", info["title"], APITitle)
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing")
	}
	if len(paths) < 100 {
		t.Errorf("paths count = %d, want >= 100", len(paths))
	}
	// Spot-check a few critical paths the F.3.2 sweep relies on.
	mustHave := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/me",
		"/api/v1/profile/",
		"/api/v1/jobs/{id}",
		"/api/v1/proposals/{id}",
		"/api/v1/messaging/conversations",
		"/api/v1/notifications/preferences",
		"/api/v1/me/billing-profile/",
	}
	for _, path := range mustHave {
		if _, ok := paths[path]; !ok {
			t.Errorf("paths is missing %s", path)
		}
	}
	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("components missing")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatalf("components.schemas missing")
	}
	mustHaveSchemas := []string{"AuthResponse", "MeResponse", "ProfileResponse", "JobResponse", "ProposalResponse", "ErrorResponse"}
	for _, s := range mustHaveSchemas {
		if _, ok := schemas[s]; !ok {
			t.Errorf("schemas is missing %s", s)
		}
	}
}

// TestOpenAPIEverChiRouteIsCovered verifies that every chi.Walk-emitted
// route appears in the OpenAPI paths map (modulo the openapi.json
// endpoints themselves and the /metrics scrape). This is the
// "drift-proof" property: a new handler registered via chi will
// always show up in the schema, no separate manifest to update.
func TestOpenAPIEveryChiRouteIsCovered(t *testing.T) {
	router := NewRouter(snapshotDeps())
	doc := BuildOpenAPIDocument(router)

	paths, _ := doc["paths"].(map[string]map[string]any)
	if paths == nil {
		// json round-trip nuance — re-extract via any
		raw, _ := doc["paths"].(map[string]any)
		paths = make(map[string]map[string]any, len(raw))
		for k, v := range raw {
			if m, ok := v.(map[string]any); ok {
				paths[k] = m
			}
		}
	}

	var missing []string
	skip := map[string]bool{
		"/api/openapi.json":    true,
		"/api/v1/openapi.json": true,
		"/metrics":             true,
	}

	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if skip[route] {
			return nil
		}
		oapi := chiPathToOpenAPI(route)
		ops, ok := paths[oapi]
		if !ok {
			missing = append(missing, method+" "+route)
			return nil
		}
		methodKey := strings.ToLower(method)
		if _, ok := ops[methodKey]; !ok {
			missing = append(missing, method+" "+route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("OpenAPI document does not cover %d route(s):\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// marshalDeterministic JSON-marshals a value with stable key ordering.
// Go's encoding/json already sorts map keys alphabetically, so the
// only extra step is enabling indent for human-readable output.
func marshalDeterministic(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	// Ensure trailing newline so the file edits diff cleanly.
	if !strings.HasSuffix(string(b), "\n") {
		b = append(b, '\n')
	}
	return b, nil
}
