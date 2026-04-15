package search_test

import (
	"os"
	"testing"
)

// golden_test.go hosts the live-OpenAI semantic search suite. Each
// test here exercises a real vector embedding path against a real
// Typesense cluster, so they are EXPENSIVE by the standards of the
// rest of the search package tests (~200 tokens per query through
// the OpenAI API).
//
// The entire suite is skipped by default to keep CI runs free. Set
// OPENAI_EMBEDDINGS_LIVE=true in the local environment to opt in.
// Phase 1 ships only the skeleton; phase 3 will fill the three
// fake queries below with real expected top-3 profile IDs once the
// test fixtures are loaded.
//
// Cost budget: ~$0.05 total across hundreds of dev runs.

// goldenEnabled reads the gating env var once and returns whether
// the live suite should run.
func goldenEnabled() bool {
	return os.Getenv("OPENAI_EMBEDDINGS_LIVE") == "true"
}

func TestGolden_NextjsDeveloperParis(t *testing.T) {
	if !goldenEnabled() {
		t.Skip("set OPENAI_EMBEDDINGS_LIVE=true to run live golden tests")
	}
	// Phase 3 will:
	//   1. Seed the testcontainer Typesense with the 200-fixture profiles.
	//   2. Build a real OpenAI embeddings client from OPENAI_API_KEY.
	//   3. Issue a hybrid query "nextjs développeur paris".
	//   4. Assert that the top-3 hits match the expected profile IDs.
	t.Fatal("golden test not yet implemented; phase 3 fills in expected IDs")
}

func TestGolden_BusinessReferrerSaaS(t *testing.T) {
	if !goldenEnabled() {
		t.Skip("set OPENAI_EMBEDDINGS_LIVE=true to run live golden tests")
	}
	t.Fatal("golden test not yet implemented; phase 3 fills in expected IDs")
}

func TestGolden_AIEngineerLLM(t *testing.T) {
	if !goldenEnabled() {
		t.Skip("set OPENAI_EMBEDDINGS_LIVE=true to run live golden tests")
	}
	t.Fatal("golden test not yet implemented; phase 3 fills in expected IDs")
}
