package features

import "strings"

// ExtractSkillsOverlap computes the skills-overlap ratio described in
// `docs/ranking-v1.md` §3.2-2.
//
//	query_skills   = tokenize(query_text) ∪ filter.skills
//	profile_skills = canonical(doc.skills)
//	overlap        = |query_skills ∩ profile_skills|
//	skills_overlap_ratio = (|query_skills| == 0) ? 0 : overlap / |query_skills|
//
// Rules specific to V1 :
//   - Referrers never compete on skills — the feature returns 0 regardless of
//     the query / doc content (its weight in the referrer table is 0% but we
//     still emit a stable 0 so the composite scorer can multiply cleanly).
//   - Tokens are lowercased and trimmed at the boundary ; canonicalisation
//     deeper than that (lemmatisation, synonyms…) is a later-round concern.
//   - An empty query-side skill set yields 0 — returning 0 is the
//     cold-start-safe choice because the composite scorer redistributes the
//     unused weight via §5.2.
func ExtractSkillsOverlap(q Query, doc SearchDocumentLite, _ Config) float64 {
	if q.Persona == PersonaReferrer {
		return 0
	}

	querySet := buildQuerySkillSet(q)
	if len(querySet) == 0 {
		return 0
	}

	profileSet := buildProfileSkillSet(doc.Skills)
	if len(profileSet) == 0 {
		return 0
	}

	overlap := 0
	for token := range querySet {
		if _, ok := profileSet[token]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(querySet))
}

// buildQuerySkillSet merges normalised tokens + filter skills into a
// deduplicated set. We keep both sources normalised identically (lower +
// trimmed) so the intersection is symmetric.
func buildQuerySkillSet(q Query) map[string]struct{} {
	out := make(map[string]struct{}, len(q.NormalisedTokens)+len(q.FilterSkills))
	for _, t := range q.NormalisedTokens {
		if key := normaliseSkill(t); key != "" {
			out[key] = struct{}{}
		}
	}
	for _, s := range q.FilterSkills {
		if key := normaliseSkill(s); key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

// buildProfileSkillSet mirrors buildQuerySkillSet for the document side.
func buildProfileSkillSet(skills []string) map[string]struct{} {
	out := make(map[string]struct{}, len(skills))
	for _, s := range skills {
		if key := normaliseSkill(s); key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

// normaliseSkill is the single source of truth for how a skill token is
// canonicalised. Kept minimal in V1 : lowercasing + trimming of whitespace.
// Future rounds may plug in a synonym map here without touching the
// extraction logic.
func normaliseSkill(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
