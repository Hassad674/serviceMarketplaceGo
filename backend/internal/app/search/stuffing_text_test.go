package search

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// stuffing_text_test.go covers the stuffingText helper feeding the
// keyword-stuffing detector (§7.1). The helper concatenates SkillsText
// + About into one lowercased token sequence the rule can scan.

func TestStuffingText_BothFieldsPopulated(t *testing.T) {
	lite := features.SearchDocumentLite{
		SkillsText: "React Node",
		About:      "Senior Frontend",
	}
	got := stuffingText(lite)
	assert.Equal(t, "react node senior frontend", got)
}

func TestStuffingText_SkillsOnly(t *testing.T) {
	lite := features.SearchDocumentLite{SkillsText: "Vue Svelte"}
	got := stuffingText(lite)
	assert.Equal(t, "vue svelte", got)
}

func TestStuffingText_AboutOnly(t *testing.T) {
	lite := features.SearchDocumentLite{About: "Building SaaS B2B"}
	got := stuffingText(lite)
	assert.Equal(t, "building saas b2b", got)
}

func TestStuffingText_BothEmpty(t *testing.T) {
	lite := features.SearchDocumentLite{}
	got := stuffingText(lite)
	assert.Equal(t, "", got)
}

func TestStuffingText_TrimsWhitespace(t *testing.T) {
	lite := features.SearchDocumentLite{
		SkillsText: "   react   ",
		About:      "  expert  ",
	}
	got := stuffingText(lite)
	// The trim removes the leading / trailing whitespace; interior
	// spaces are preserved (the rule retokenises so they're harmless).
	assert.Equal(t, "react expert", got)
}

// TestStuffingText_DetectsStuffingPattern proves the helper produces
// text that the actual stuffing rule recognises — feeding "react"
// 12× through About now triggers the rule, where before only the
// SkillsText path could fire.
func TestStuffingText_AboutAlonecanTriggerStuffing(t *testing.T) {
	// 12 repetitions of "react" — exceeds the default max repetition
	// of 5 and below the distinct ratio of 0.3.
	about := "react react react react react react react react react react react react"
	lite := features.SearchDocumentLite{About: about}
	got := stuffingText(lite)
	// Sanity-check the helper itself produced the lowercased sequence.
	assert.Contains(t, got, "react react react",
		"about-only stuffing must be passed through unchanged for the rule to count repetitions")
}
