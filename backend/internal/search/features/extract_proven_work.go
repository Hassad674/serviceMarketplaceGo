package features

import "math"

// ExtractProvenWork computes the proven_work_score described in
// `docs/ranking-v1.md` §3.2-4.
//
//	raw = 0.40 × log(1 + completed_projects)
//	    + 0.35 × log(1 + unique_clients)
//	    + 0.25 × sqrt(repeat_client_rate)
//	proven_work_score = min(1.0, raw / log(1 + project_count_cap))
//
// Default normaliser : log(1 + 100) ≈ 4.62 so a profile with 100 completed
// projects + 100 unique clients + 100% repeat rate saturates at ≈ 1.0.
//
// Referrers always score 0 — they don't complete projects themselves. Their
// weight for this feature in the persona table is 0% anyway ; returning 0
// keeps the composite clean.
//
// The repeat-rate input is clamped to [0, 1] : the indexer is expected to
// emit values in that range but we defensively pin them.
func ExtractProvenWork(persona Persona, doc SearchDocumentLite, cfg Config) float64 {
	if persona == PersonaReferrer {
		return 0
	}

	projects := float64(doc.CompletedProjects)
	clients := float64(doc.UniqueClientsCount)
	repeatRate := clamp01(doc.RepeatClientRate)

	if projects < 0 {
		projects = 0
	}
	if clients < 0 {
		clients = 0
	}

	raw := 0.40*math.Log1p(projects) + 0.35*math.Log1p(clients) + 0.25*math.Sqrt(repeatRate)

	cap := cfg.ProjectCountCap
	if cap <= 0 {
		return 0
	}
	den := math.Log1p(float64(cap))
	if den == 0 {
		return 0
	}
	return clamp01(raw / den)
}
