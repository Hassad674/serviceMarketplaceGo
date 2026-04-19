package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractSkillsOverlap returns |query ∩ profile| / |query|, 0 for referrers
// or empty query, with tokens lowercased on both sides.
func TestExtractSkillsOverlap(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name   string
		q      Query
		skills []string
		want   float64
	}{
		{
			name:   "zero state -> 0 (empty query + empty skills)",
			q:      Query{Persona: PersonaFreelance},
			skills: nil,
			want:   0,
		},
		{
			name: "perfect overlap",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"react", "typescript"},
			},
			skills: []string{"React", "TypeScript"},
			want:   1,
		},
		{
			name: "partial overlap (1/3)",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"react", "kubernetes", "terraform"},
			},
			skills: []string{"React", "Vue", "Angular"},
			want:   1.0 / 3.0,
		},
		{
			name: "filter skills combine with tokens",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"senior"},
				FilterSkills:     []string{"Go", "Postgres"},
			},
			skills: []string{"go", "postgres", "typescript"},
			want:   2.0 / 3.0,
		},
		{
			name: "referrer persona always 0",
			q: Query{
				Persona:          PersonaReferrer,
				NormalisedTokens: []string{"marketing"},
			},
			skills: []string{"marketing", "sales"},
			want:   0,
		},
		{
			name: "agency persona still computes overlap",
			q: Query{
				Persona:          PersonaAgency,
				NormalisedTokens: []string{"devops"},
			},
			skills: []string{"devops"},
			want:   1,
		},
		{
			name: "profile skills empty -> 0 (no way to match)",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"react"},
			},
			skills: nil,
			want:   0,
		},
		{
			name: "no overlap at all",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"java"},
			},
			skills: []string{"python"},
			want:   0,
		},
		{
			name: "duplicates on both sides deduplicated",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"react", "react", "react"},
				FilterSkills:     []string{"React"},
			},
			skills: []string{"React", "React"},
			want:   1,
		},
		{
			name: "case insensitive + whitespace trimmed",
			q: Query{
				Persona:          PersonaFreelance,
				NormalisedTokens: []string{"  REACT  ", "Typescript"},
			},
			skills: []string{"react", "typescript"},
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{Skills: tt.skills}
			got := ExtractSkillsOverlap(tt.q, doc, cfg)
			assert.InDelta(t, tt.want, got, 1e-9)
			assert.GreaterOrEqual(t, got, 0.0)
			assert.LessOrEqual(t, got, 1.0)
		})
	}
}

// normaliseSkill is the single source of truth — regression-pin so a refactor
// doesn't silently break tokenisation.
func TestNormaliseSkill(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"react", "react"},
		{"REACT", "react"},
		{"  React  ", "react"},
		{"", ""},
		{"TypeScript", "typescript"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, normaliseSkill(tt.in))
		})
	}
}
