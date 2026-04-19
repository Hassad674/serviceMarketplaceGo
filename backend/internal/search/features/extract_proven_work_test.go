package features

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractProvenWork — the four worked examples from §3.2-4 table must match.
func TestExtractProvenWork(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		persona  Persona
		projects int32
		clients  int32
		repeat   float64
		wantMin  float64
		wantMax  float64
	}{
		// Values computed exactly from §3.2-4 formula (spec table is rounded).
		//   0.40 × log(1+P) + 0.35 × log(1+C) + 0.25 × sqrt(R), / log(101)
		{
			name:     "senior 5 projects 5 clients 20%",
			persona:  PersonaFreelance,
			projects: 5,
			clients:  5,
			repeat:   0.20,
			wantMin:  0.30,
			wantMax:  0.33, // exact ≈ 0.3154
		},
		{
			name:     "junior 50/30/15%",
			persona:  PersonaFreelance,
			projects: 50,
			clients:  30,
			repeat:   0.15,
			wantMin:  0.60,
			wantMax:  0.65, // exact ≈ 0.6222
		},
		{
			name:     "client farming 30 projects 2 clients 100% repeat",
			persona:  PersonaAgency,
			projects: 30,
			clients:  2,
			repeat:   1.0,
			wantMin:  0.40,
			wantMax:  0.45, // exact ≈ 0.4351
		},
		{
			name:     "established senior 10/7/40%",
			persona:  PersonaFreelance,
			projects: 10,
			clients:  7,
			repeat:   0.40,
			wantMin:  0.38,
			wantMax:  0.42, // exact ≈ 0.3998
		},
		{
			name:     "zero state",
			persona:  PersonaFreelance,
			projects: 0,
			clients:  0,
			repeat:   0,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "saturation at 100 projects",
			persona:  PersonaAgency,
			projects: 100,
			clients:  100,
			repeat:   1.0,
			// 0.40×log(101) + 0.35×log(101) + 0.25×1 = 3.4613 + 0.25 ≈ 3.711 / log(101) ≈ 0.804
			wantMin: 0.78,
			wantMax: 0.82,
		},
		{
			name:     "referrer persona -> 0",
			persona:  PersonaReferrer,
			projects: 100,
			clients:  100,
			repeat:   1.0,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "extreme attack 1000 projects",
			persona:  PersonaFreelance,
			projects: 1000,
			clients:  1000,
			repeat:   1.0,
			wantMin:  1,
			wantMax:  1,
		},
		{
			name:     "negative values defensively coerced",
			persona:  PersonaFreelance,
			projects: -5,
			clients:  -5,
			repeat:   -0.5,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "repeat rate above 1 clamped",
			persona:  PersonaFreelance,
			projects: 10,
			clients:  10,
			repeat:   5.0,
			wantMin:  0.4,
			wantMax:  0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{
				CompletedProjects:  tt.projects,
				UniqueClientsCount: tt.clients,
				RepeatClientRate:   tt.repeat,
			}
			got := ExtractProvenWork(tt.persona, doc, cfg)
			assert.GreaterOrEqual(t, got, tt.wantMin,
				"expected %s >= %f got %f", tt.name, tt.wantMin, got)
			assert.LessOrEqual(t, got, tt.wantMax,
				"expected %s <= %f got %f", tt.name, tt.wantMax, got)
			assert.False(t, math.IsNaN(got))
		})
	}
}

// Cap of 0 short-circuits to 0 (defensive).
func TestExtractProvenWork_InvalidCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProjectCountCap = 0
	doc := SearchDocumentLite{CompletedProjects: 10, UniqueClientsCount: 5, RepeatClientRate: 0.5}
	assert.Equal(t, 0.0, ExtractProvenWork(PersonaFreelance, doc, cfg))
}
