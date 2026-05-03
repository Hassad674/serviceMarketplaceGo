package search

import (
	"context"
	"fmt"
	"strings"
)

// collectResults drains the results channel and returns the first
// non-nil error, wrapped with the step name for observability.
func collectResults(results chan loadResult, expected int) error {
	var firstErr error
	for k := 0; k < expected; k++ {
		r := <-results
		if r.err != nil && firstErr == nil {
			firstErr = fmt.Errorf("indexer %s: %w", r.name, r.err)
		}
	}
	return firstErr
}

// embedActor builds the text input passed to the embeddings API and
// returns the resulting vector. Extracted so BuildDocument stays
// short and so the text-composition rules can be unit-tested.
func (i *Indexer) embedActor(ctx context.Context, s *RawActorSignals, skills []string) ([]float32, error) {
	text := ComposeEmbeddingText(s, skills...)
	if text == "" {
		// An entirely empty profile has no meaningful vector;
		// rather than send blank text to OpenAI we skip the
		// embedding. The document will still index — just without
		// semantic search coverage — and the next profile update
		// will re-trigger the reindex with a populated text.
		return nil, nil
	}
	return i.embedder.Embed(ctx, text)
}

// MaxEmbeddingInputChars caps the composed text sent to the
// embeddings API. The limit controls per-document cost: at ~4 chars
// per token × 2k chars the payload lands near 500 tokens, which on
// `text-embedding-3-small` costs ~$0.00001. Truncating keeps a
// verbose 10k-char about field from spiking cost per profile.
const MaxEmbeddingInputChars = 2000

// ComposeEmbeddingText is the exported helper that converts raw
// profile signals into the text input for the embeddings API. Kept
// exported + pure so golden tests can replay the exact same input.
//
// Field order matches the phase 3 spec: display_name, title,
// skills_text (derived from the caller's skill slice), about. Each
// field is trimmed and skipped when empty. Result is truncated at
// MaxEmbeddingInputChars to bound cost even on profiles with very
// long about sections.
func ComposeEmbeddingText(s *RawActorSignals, skills ...string) string {
	if s == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if v := strings.TrimSpace(s.DisplayName); v != "" {
		parts = append(parts, v)
	}
	if v := strings.TrimSpace(s.Title); v != "" {
		parts = append(parts, v)
	}
	if skillsText := strings.TrimSpace(strings.Join(skills, " ")); skillsText != "" {
		parts = append(parts, skillsText)
	}
	if v := strings.TrimSpace(s.About); v != "" {
		parts = append(parts, v)
	}
	if len(s.ExpertiseDomains) > 0 {
		parts = append(parts, strings.Join(s.ExpertiseDomains, ", "))
	}
	combined := strings.TrimSpace(strings.Join(parts, ". "))
	if len(combined) > MaxEmbeddingInputChars {
		combined = combined[:MaxEmbeddingInputChars]
	}
	return combined
}

// assembleDocument builds the SearchDocument from the populated
// aggregate. Every numerical feature goes through the ranking.go
// helpers so the ranking formulas have a single source of truth.
func (i *Indexer) assembleDocument(agg *indexAggregate, persona Persona) (*SearchDocument, error) {
	s := agg.signals
	if s == nil {
		return nil, fmt.Errorf("search indexer: signals are nil")
	}

	completion := ProfileCompletionScore(CompletionInput{
		HasPhoto:         s.PhotoURL != "",
		HasAbout:         strings.TrimSpace(s.About) != "",
		HasTitle:         strings.TrimSpace(s.Title) != "",
		HasVideo:         s.VideoURL != "",
		ExpertiseCount:   len(s.ExpertiseDomains),
		SkillsCount:      len(agg.skills),
		HasPricing:       agg.pricing != nil && agg.pricing.HasPricing,
		HasLocation:      s.City != "" && s.CountryCode != "",
		SocialLinksCount: s.SocialLinksCount,
		LanguagesCount:   len(s.LanguagesProfessional),
	})

	doc := &SearchDocument{
		// Composite ID keeps one org + multiple personas isolated
		// so the agency upsert cannot overwrite the freelance
		// document (or vice versa). The frontend uses the
		// OrganizationID field to build profile links.
		ID:                      s.OrganizationID.String() + ":" + string(persona),
		OrganizationID:          s.OrganizationID.String(),
		Persona:                 persona,
		IsPublished:             s.IsPublished,
		DisplayName:             s.DisplayName,
		Title:                   s.Title,
		PhotoURL:                s.PhotoURL,
		City:                    s.City,
		CountryCode:             s.CountryCode,
		WorkMode:                nilToEmpty(s.WorkMode),
		LanguagesProfessional:   nilToEmpty(s.LanguagesProfessional),
		LanguagesConversational: nilToEmpty(s.LanguagesConversational),
		AvailabilityStatus:      s.AvailabilityStatus,
		AvailabilityPriority:    AvailabilityPriority(s.AvailabilityStatus),
		ExpertiseDomains:        nilToEmpty(s.ExpertiseDomains),
		Skills:                  nilToEmpty(agg.skills),
		SkillsText:              strings.Join(agg.skills, " "),
		// completion is bounded 0..100 by ProfileCompletionScore — the
		// int32 narrowing is provably overflow-free.
		ProfileCompletionScore: int32(completion), // #nosec G115 -- bounded 0..100
		LastActiveAt:           s.LastActiveAt.Unix(),
		IsFeatured:             i.isFeatured(s.OrganizationID),
	}

	applyLocation(doc, s)
	applyPricing(doc, agg.pricing)
	applyRating(doc, agg.rating)
	applyEarnings(doc, agg.earnings)
	applyMessaging(doc, agg.messaging)
	applyClientHistory(doc, agg.clientHistory)
	applyReviewDiversity(doc, agg.reviewDiversity)
	applyAccountAge(doc, agg.accountAge)
	doc.IsVerified = agg.kyc

	if agg.embed != nil {
		doc.Embedding = agg.embed
	}
	doc.SetTimestamps(s.CreatedAt, s.UpdatedAt)

	if err := doc.Validate(); err != nil {
		return nil, fmt.Errorf("search indexer: assembled document invalid: %w", err)
	}
	return doc, nil
}

// applyLocation copies the geopoint into the document if both
// coordinates are present. Typesense expects `[lat, lng]` — any
// other order breaks the geo filter.
func applyLocation(doc *SearchDocument, s *RawActorSignals) {
	if s.Latitude == nil || s.Longitude == nil {
		return
	}
	doc.Location = []float64{*s.Latitude, *s.Longitude}
}

// applyPricing flattens a RawPricing into the document's pricing
// fields. When no pricing exists, the fields stay zero/empty and
// the JSON omitempty tags keep them out of the payload.
func applyPricing(doc *SearchDocument, p *RawPricing) {
	if p == nil || !p.HasPricing {
		return
	}
	doc.PricingType = p.Type
	doc.PricingMinAmount = p.MinAmount
	doc.PricingMaxAmount = p.MaxAmount
	doc.PricingCurrency = p.Currency
	doc.PricingNegotiable = p.Negotiable
}

// applyRating runs the Bayesian rating through the ranking helper
// and records the derived top-rated badge.
func applyRating(doc *SearchDocument, r *RawRatingAggregate) {
	if r == nil {
		return
	}
	doc.RatingAverage = r.Average
	// review counts are at most a few thousand per actor — the int32
	// narrowing is provably overflow-free for any realistic actor.
	doc.RatingCount = int32(r.Count) // #nosec G115 -- bounded by review-table size per actor
	doc.RatingScore = BayesianRatingScore(r.Average, r.Count)
	doc.IsTopRated = IsTopRated(r.Average, r.Count)
}

// applyEarnings records the two earnings signals.
func applyEarnings(doc *SearchDocument, e *RawEarningsAggregate) {
	if e == nil {
		return
	}
	doc.TotalEarned = e.TotalAmount
	// CompletedProjects counts proposals; even the most active actor
	// has fewer than 1M, well below int32 range.
	doc.CompletedProjects = int32(e.CompletedProjects) // #nosec G115 -- bounded by proposal-table size per actor
}

// applyMessaging records the messaging-driven response rate. Nil is
// valid (inactive account) and maps to zero.
func applyMessaging(doc *SearchDocument, m *RawMessagingSignals) {
	if m == nil {
		return
	}
	doc.ResponseRate = m.ResponseRate
}

// applyClientHistory copies the proven-work signals onto the document.
// Nil is treated as "no history" — the document's fields stay at zero
// which is the contract callers in the ranking pipeline rely on
// (see docs/ranking-v1.md §3.2-4).
func applyClientHistory(doc *SearchDocument, h *RawClientHistory) {
	if h == nil {
		return
	}
	doc.UniqueClientsCount = int32(h.UniqueClients) // #nosec G115 -- bounded by client-table size per actor
	doc.RepeatClientRate = h.RepeatClientRate
}

// applyReviewDiversity copies the reviewer-diversity signals onto the
// document. Nil means no reviews yet, which surfaces as zeros —
// downstream code interprets that as "cold-start floor" territory
// (see docs/ranking-v1.md §3.2-3 step 4).
func applyReviewDiversity(doc *SearchDocument, d *RawReviewDiversity) {
	if d == nil {
		return
	}
	doc.UniqueReviewersCount = int32(d.UniqueReviewers) // #nosec G115 -- bounded by reviewer-table size per actor
	doc.MaxReviewerShare = d.MaxReviewerShare
	doc.ReviewRecencyFactor = d.ReviewRecencyFactor
}

// applyAccountAge copies the dispute + age signals onto the document.
// Nil means "no traceable owner user" (test fixtures where the owner
// was never wired); zeros are a safe default that make the downstream
// is_verified_mature check fail and the account_age_bonus drop to 0.
func applyAccountAge(doc *SearchDocument, a *RawAccountAge) {
	if a == nil {
		return
	}
	// LostDisputes is bounded by total disputes (low thousands).
	// AccountAgeDays is bounded by service uptime (years × 365).
	doc.LostDisputesCount = int32(a.LostDisputes) // #nosec G115 -- bounded by dispute-table size per actor
	doc.AccountAgeDays = int32(a.AccountAgeDays)  // #nosec G115 -- bounded by service uptime in days
}

// nilToEmpty turns a nil slice into an empty slice so the serialised
// JSON payload uses `[]` instead of `null`. Typesense accepts both,
// but `[]` makes the wire format easier to diff in integration
// tests.
func nilToEmpty[T any](in []T) []T {
	if in == nil {
		return []T{}
	}
	return in
}
