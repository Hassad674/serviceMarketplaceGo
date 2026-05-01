package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/config"
	paymentapp "marketplace-backend/internal/app/payment"
	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/internal/search/antigaming"
	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/rules"
	"marketplace-backend/internal/search/scorer"
)

// paymentProcessor returns the payment service as PaymentProcessor if
// Stripe is configured, nil otherwise. Wraps the typed *paymentapp.Service
// in the narrower port interface so the proposal feature receives the
// minimum surface needed for milestone payments.
func paymentProcessor(svc *paymentapp.Service, cfg *config.Config) service.PaymentProcessor {
	if cfg.StripeConfigured() {
		return svc
	}
	return nil
}

// orgOwnerLookupAdapter implements handler.OrgOwnerLookup on top of
// the existing OrganizationRepository. Lives in the wiring layer
// because it is a one-line bridge that should not bloat the handler
// package nor the organization domain.
type orgOwnerLookupAdapter struct {
	orgs repository.OrganizationRepository
}

func (a *orgOwnerLookupAdapter) OwnerUserIDForOrg(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error) {
	org, err := a.orgs.FindByID(ctx, orgID)
	if err != nil {
		return uuid.Nil, err
	}
	return org.OwnerUserID, nil
}

// wsOriginPatterns converts full origin URLs (e.g. "https://example.com")
// to hostname patterns (e.g. "example.com") for coder/websocket
// OriginPatterns, and adds a wildcard for local development.
func wsOriginPatterns(origins []string) []string {
	patterns := make([]string, 0, len(origins)+1)
	for _, o := range origins {
		// Strip scheme — coder/websocket matches on hostname only.
		host := strings.TrimPrefix(o, "https://")
		host = strings.TrimPrefix(host, "http://")
		if host != "" {
			patterns = append(patterns, host)
		}
	}
	// Always allow localhost for dev.
	patterns = append(patterns, "localhost:*")
	return patterns
}

// buildRankingPipeline composes the four Stage 2-5 ranking packages
// into the RankingPipeline consumed by app/search.Service. All knobs
// live in RANKING_* environment variables; see docs/ranking-tuning.md
// for the operator playbook. Missing env vars fall back to the safe
// public defaults published in docs/ranking-v1.md §11.
//
// Boot-time fail-loud policy: scorer + rules configs return an error
// on malformed values so a typo in a weight raises slog.Error +
// os.Exit(1) rather than silently zeroing the ranking.
//
// Extract-time configs (features + antigaming) swallow malformed
// values by design — their individual extractors handle zero values
// gracefully, so a mistyped threshold just falls back to the default
// rather than taking down the search path.
func buildRankingPipeline() *appsearch.RankingPipeline {
	fcfg := features.LoadConfigFromEnv()
	agCfg := antigaming.LoadConfigFromEnv()
	scCfg, scErr := scorer.LoadConfigFromEnv()
	if scErr != nil {
		slog.Error("ranking: scorer config invalid", "error", scErr)
		os.Exit(1)
	}
	rlCfg, rlErr := rules.LoadConfigFromEnv()
	if rlErr != nil {
		slog.Error("ranking: rules config invalid", "error", rlErr)
		os.Exit(1)
	}

	ext := features.NewDefaultExtractor(fcfg)
	ag := antigaming.NewPipeline(agCfg, antigaming.NoopLinkedReviewersDetector{}, antigaming.SlogLogger{})
	rer := scorer.NewWeightedScorer(scCfg)
	br := rules.NewBusinessRules(rlCfg)

	return appsearch.NewRankingPipeline(ext, ag, rer, br)
}
