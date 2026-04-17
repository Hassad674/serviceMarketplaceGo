package search

import (
	"context"
	"encoding/json"
	"fmt"
)

// drift_detector.go compares the per-persona document counts in
// Postgres against Typesense. The phase 3 spec sets the alerting
// threshold at 0.5% drift — anything above that surfaces as a WARN
// in the structured logs.
//
// The comparison function (DetectDrift) is pure and I/O-free: it
// takes pre-counted snapshots from both sources so tests can swap
// in fakes. The CLI wiring (cmd/drift-check) fetches the counts
// and invokes this function.

// PersonaCount is a typed pair of (persona, count). Used instead
// of a map so the JSON serialisation (if we ever log the snapshot)
// stays stable.
type PersonaCount struct {
	Persona Persona
	Count   int64
}

// DriftReport is the typed result of a drift-detection run.
type DriftReport struct {
	Postgres   map[Persona]int64
	Typesense  map[Persona]int64
	Ratios     map[Persona]float64
	MaxRatio   float64
	IsCritical bool
}

// DriftThreshold is the fraction (0 <= x <= 1) of permissible drift
// between Postgres + Typesense counts. 0.005 = 0.5% per the phase 3
// spec.
const DriftThreshold = 0.005

// DetectDriftOpts is the optional tuning struct. Zero value applies
// the default threshold.
type DetectDriftOpts struct {
	Threshold float64
}

// DetectDrift compares the two count snapshots and returns a typed
// report. Pure function — no logging, no I/O.
func DetectDrift(postgres, typesense []PersonaCount, opts DetectDriftOpts) DriftReport {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = DriftThreshold
	}
	report := DriftReport{
		Postgres:  toPersonaMap(postgres),
		Typesense: toPersonaMap(typesense),
		Ratios:    make(map[Persona]float64, 3),
	}
	for _, persona := range []Persona{PersonaFreelance, PersonaAgency, PersonaReferrer} {
		ratio := computeDriftRatio(report.Postgres[persona], report.Typesense[persona])
		report.Ratios[persona] = ratio
		if ratio > report.MaxRatio {
			report.MaxRatio = ratio
		}
	}
	report.IsCritical = report.MaxRatio > threshold
	return report
}

// toPersonaMap flattens a slice of PersonaCount into a lookup map.
func toPersonaMap(entries []PersonaCount) map[Persona]int64 {
	m := make(map[Persona]int64, len(entries))
	for _, e := range entries {
		m[e.Persona] = e.Count
	}
	return m
}

// computeDriftRatio returns the absolute difference between a and b
// divided by the max.
//
//   - a==b==0 → ratio 0 (no drift)
//   - one side zero, other non-zero → ratio 1 (100% drift)
//   - otherwise diff/max
func computeDriftRatio(a, b int64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	if a == 0 || b == 0 {
		return 1
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	max := a
	if b > max {
		max = b
	}
	return float64(diff) / float64(max)
}

// CountDocumentsByPersona queries Typesense for document counts per
// persona. Runs three facet-limited searches (one per persona)
// because Typesense does not expose a dedicated count endpoint.
func (c *Client) CountDocumentsByPersona(ctx context.Context, collection string) ([]PersonaCount, error) {
	personas := []Persona{PersonaFreelance, PersonaAgency, PersonaReferrer}
	out := make([]PersonaCount, 0, len(personas))
	for _, p := range personas {
		count, err := c.countByFilter(ctx, collection, fmt.Sprintf("persona:%s", p))
		if err != nil {
			return nil, fmt.Errorf("count %s: %w", p, err)
		}
		out = append(out, PersonaCount{Persona: p, Count: count})
	}
	return out, nil
}

// countByFilter runs a 0-per-page search against the collection
// with the given filter_by and returns `found`.
func (c *Client) countByFilter(ctx context.Context, collection, filterBy string) (int64, error) {
	params := SearchParams{
		Q:        "*",
		QueryBy:  "display_name",
		FilterBy: filterBy,
		PerPage:  0,
	}
	raw, err := c.Query(ctx, collection, params)
	if err != nil {
		return 0, err
	}
	var envelope struct {
		Found int64 `json:"found"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return 0, fmt.Errorf("decode count: %w", err)
	}
	return envelope.Found, nil
}
