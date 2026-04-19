package search

import (
	"time"

	"marketplace-backend/internal/search"
	"marketplace-backend/internal/search/features"
)

// document_adapter.go is the boundary between the Typesense response
// shape (search.SearchDocument) and the ranking pipeline's
// feature-extractor input (features.SearchDocumentLite).
//
// Why live here and not inside features/ :
//   - features/ is dependency-free — pulling search.SearchDocument in
//     would couple it to the Typesense client + embeddings packages.
//   - This package (app/search) is already the single owner of the
//     candidate retrieval path. Adapting there keeps the contract
//     narrow and the reverse dependency clean.
//
// All fields map 1:1 to the declared contract in features/types.go
// §121+. Missing fields silently default to their zero value — the
// feature extractors are written to handle zero defensively
// (RatingScoreDiverse returns ColdStartFloor on n == 0, etc.).

// TypesenseHit bundles the SearchDocument returned by Typesense with
// the per-hit match score used by the text-match extractor. The raw
// Typesense response returns `_text_match` as an integer (non-bucketed
// raw BM25 score) and `text_match_info.score` as a stringified number
// — we keep only what the ranking pipeline actually needs.
//
// TextMatchBucket is the 0-10 bucketed value the ExtractTextMatch
// extractor consumes (docs/ranking-v1.md §3.2-1). The retrieval layer
// is responsible for deriving the bucket from Typesense's raw score
// because the raw value alone is not meaningful across queries — a
// BM25 score of 100 is high for one query and low for another.
//
// See computeTextMatchBuckets below for the normalisation rule used
// when the retrieval layer has not pre-computed the bucket.
type TypesenseHit struct {
	Document        search.SearchDocument
	TextMatchBucket int
}

// ToSearchDocumentLite projects a TypesenseHit onto the lite struct
// the feature extractors operate on. Never allocates a clone of the
// input slices — the lite struct only references them — so callers
// must treat the hit as read-only after this call.
//
// nowUnix is injected (rather than read from the clock) so the
// extractor stays pure. The pipeline fills it in from the caller's
// `in.Now` so every candidate in a single rerank uses the same
// reference instant.
func (h TypesenseHit) ToSearchDocumentLite(nowUnix int64) features.SearchDocumentLite {
	doc := h.Document
	return features.SearchDocumentLite{
		OrganizationID: doc.OrganizationID,
		Persona:        features.Persona(doc.Persona),

		Skills:     doc.Skills,
		SkillsText: doc.SkillsText,
		// The SearchDocument schema does not carry an `about` field
		// yet — reserve the attribution for the day it lands. The
		// stuffing rule still receives SkillsText through the Text
		// field of RawSignals so the 0-value here is harmless.
		About: "",

		RatingAverage:          doc.RatingAverage,
		RatingCount:            doc.RatingCount,
		CompletedProjects:      doc.CompletedProjects,
		ProfileCompletionScore: doc.ProfileCompletionScore,
		LastActiveAt:           doc.LastActiveAt,
		ResponseRate:           doc.ResponseRate,
		IsVerified:             doc.IsVerified,

		UniqueClientsCount:   doc.UniqueClientsCount,
		RepeatClientRate:     doc.RepeatClientRate,
		UniqueReviewersCount: doc.UniqueReviewersCount,
		MaxReviewerShare:     doc.MaxReviewerShare,
		ReviewRecencyFactor:  doc.ReviewRecencyFactor,
		LostDisputesCount:    doc.LostDisputesCount,
		AccountAgeDays:       doc.AccountAgeDays,

		NowUnix:         nowUnix,
		TextMatchBucket: h.TextMatchBucket,
	}
}

// rankingNow is a thin wrapper around time.Now so tests can stub the
// clock without reaching into package state. Kept unexported to keep
// the public surface minimal.
var rankingNow = time.Now
