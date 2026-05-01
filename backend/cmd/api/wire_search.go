package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/adapter/worker"
	"marketplace-backend/internal/adapter/worker/handlers"
	appsearch "marketplace-backend/internal/app/search"
	"marketplace-backend/internal/app/searchanalytics"
	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/search"
)

// wireSearchPublisher allocates the search-index outbox publisher
// (Postgres-backed) used by every service that mutates actor signals
// (freelance profile, referrer profile, pricing, skills, etc.) so
// they can emit a `search.reindex` event without re-wiring the whole
// chain. The publisher debounces rapid repeats so a storm of profile
// updates does not translate to a storm of index rebuilds.
//
// Returns nil when Typesense is not configured — services receive
// the nil publisher and silently skip publishing.
func wireSearchPublisher(cfg *config.Config, pendingEventsRepo *postgres.PendingEventRepository) *searchindex.Publisher {
	if !cfg.TypesenseConfigured() {
		return nil
	}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{
		Events: pendingEventsRepo,
	})
	if err != nil {
		slog.Error("search: failed to build publisher", "error", err)
		os.Exit(1)
	}
	return pub
}

// wireSearchIndexer brings up the Typesense client, EnsureSchema, the
// search-only HMAC parent key, the embeddings client (with retry), and
// registers the indexer + delete handlers on the pending-events worker.
// Returns the typesense client so the query path can reuse it; nil
// when Typesense is not configured.
func wireSearchIndexer(
	cfg *config.Config,
	db *sql.DB,
	pendingEventsWorker *worker.Worker,
) *search.Client {
	if !cfg.TypesenseConfigured() {
		slog.Warn("search: typesense not configured — the listing pages will return 503 until TYPESENSE_* env vars are set")
		return nil
	}
	tsClient, err := search.NewClient(cfg.TypesenseHost, cfg.TypesenseAPIKey)
	if err != nil {
		slog.Error("search: invalid typesense configuration", "error", err)
		os.Exit(1)
	}
	if err := search.EnsureSchema(context.Background(), search.EnsureSchemaDeps{
		Client: tsClient,
		Logger: slog.Default(),
	}); err != nil {
		slog.Warn("search: ensure schema failed, continuing without indexing", "error", err)
	}
	// Bootstrap the search-only parent key used as the HMAC parent
	// for scoped search keys. Typesense refuses to derive scoped keys
	// from the master admin key — we MUST use a key whose `actions`
	// list contains `documents:search`. We cycle the key on every
	// startup because Typesense only exposes the full value on
	// creation.
	if err := tsClient.EnsureSearchAPIKey(context.Background()); err != nil {
		slog.Error("search: failed to bootstrap search API key", "error", err)
		os.Exit(1)
	}
	slog.Info("search: search-only parent key bootstrapped")

	embedder := buildIndexerEmbedder(cfg)

	searchDataRepo := postgres.NewSearchDocumentRepository(db)
	searchIndexer, err := search.NewIndexer(searchDataRepo, embedder)
	if err != nil {
		slog.Error("search: failed to build indexer", "error", err)
		os.Exit(1)
	}
	searchIndexSvc, err := searchindex.NewService(searchindex.Config{
		Client:  tsClient,
		Indexer: searchIndexer,
		Logger:  slog.Default(),
	})
	if err != nil {
		slog.Error("search: failed to build indexing service", "error", err)
		os.Exit(1)
	}

	pendingEventsWorker.Register(pendingevent.TypeSearchReindex, handlers.NewSearchReindexHandler(searchIndexSvc))
	pendingEventsWorker.Register(pendingevent.TypeSearchDelete, handlers.NewSearchDeleteHandler(searchIndexSvc))
	slog.Info("search: typesense indexer wired")
	return tsClient
}

// buildIndexerEmbedder selects the live OpenAI embeddings client when
// OPENAI_API_KEY is set, falling back to the deterministic mock when
// it is absent. The live client is wrapped in a RetryingEmbeddings
// adapter so transient 5xx / 429 responses retry with exponential
// backoff (500ms / 1s / 2s) — matching the spec.
//
// Phase 3: with OPENAI_API_KEY set, live embeddings become MANDATORY —
// a transient failure no longer silently falls back to the mock (which
// would ship near-duplicate vectors and destroy semantic ranking). The
// retry wrapper is the only safety net.
func buildIndexerEmbedder(cfg *config.Config) search.EmbeddingsClient {
	if cfg.OpenAIAPIKey == "" {
		slog.Warn("search: OPENAI_API_KEY not set, using mock embeddings — search quality will be degraded")
		return search.NewMockEmbeddings()
	}
	openaiClient, err := search.NewOpenAIEmbeddings(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingsModel)
	if err != nil {
		slog.Error("search: OPENAI_API_KEY set but client invalid — aborting to surface config error",
			"error", err)
		os.Exit(1)
	}
	embedder := search.NewRetryingEmbeddings(openaiClient)
	slog.Info("search: live OpenAI embeddings enabled (with retry)",
		"model", cfg.OpenAIEmbeddingsModel)
	return embedder
}

// wireSearchQuery composes the query-side service: per-persona
// Typesense clients, optional hybrid embedder, analytics capture,
// ranking pipeline, LTR capture. Returns nil products when Typesense
// is not configured.
func wireSearchQuery(cfg *config.Config, db *sql.DB, tsClient *search.Client) (
	*handler.SearchHandler,
	*handler.AdminSearchStatsHandler,
) {
	if tsClient == nil {
		return nil, nil
	}
	// Phase 3: wire the analytics service so every search is captured
	// and the /search/track endpoint has somewhere to persist clicks.
	// Nil-safe — if the repo fails to build search keeps working
	// without analytics.
	analyticsRepo := postgres.NewSearchAnalyticsRepository(db)
	var analyticsSvc *searchanalytics.Service
	if svc, err := searchanalytics.NewService(searchanalytics.Config{
		Repository: analyticsRepo,
		Logger:     slog.Default(),
	}); err != nil {
		slog.Error("search: analytics service disabled", "error", err)
	} else {
		analyticsSvc = svc
	}

	// Phase 4: admin stats dashboard. Reuses the same repository
	// (which now implements both Repository and StatsRepository) so
	// there's no extra dependency. The handler is gated by
	// RequireAdmin at the router level.
	var adminStats *handler.AdminSearchStatsHandler
	if statsSvc, err := searchanalytics.NewStatsService(searchanalytics.StatsServiceConfig{
		Repository: analyticsRepo,
		Logger:     slog.Default(),
	}); err != nil {
		slog.Error("search: stats service disabled", "error", err)
	} else {
		adminStats = handler.NewAdminSearchStatsHandler(statsSvc)
	}

	// Phase 3: hybrid search needs a live embedder on the query path.
	// Reuse the same OpenAI client (with retry wrapper) we built for
	// indexing — the rate limits live on the API key, so sharing the
	// client matters.
	var queryEmbedder search.EmbeddingsClient
	if cfg.OpenAIAPIKey != "" {
		openaiClient, err := search.NewOpenAIEmbeddings(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingsModel)
		if err == nil {
			queryEmbedder = search.NewRetryingEmbeddings(openaiClient)
		} else {
			slog.Warn("search: query-time embedder disabled", "error", err)
		}
	}

	analyticsAdapter := newSearchAnalyticsRecorder(analyticsSvc)

	// Ranking V1 pipeline wiring (phase 6F) — composition of the four
	// Stage 2-5 packages. Every knob lives in RANKING_* environment
	// variables (see docs/ranking-tuning.md). Boot fails loud on
	// malformed env: a typo in a float weight must never limp into
	// prod with a silent zero.
	rankingPipeline := buildRankingPipeline()

	// LTR capture wiring — the repo is the same
	// SearchAnalyticsRepository already built above. The service
	// holds the goroutine that writes result_features_json; the repo
	// runs the UPDATE under a 3s deadline.
	var ltrRepo searchanalytics.LTRRepository = analyticsRepo

	querySvc := appsearch.NewService(appsearch.ServiceDeps{
		Freelance:        search.NewFreelanceClient(tsClient),
		Agency:           search.NewAgencyClient(tsClient),
		Referrer:         search.NewReferrerClient(tsClient),
		Embedder:         queryEmbedder,
		Analytics:        analyticsAdapter,
		Logger:           slog.Default(),
		RankingPipeline:  rankingPipeline,
		LTRRepository:    ltrRepo,
		AnalyticsService: analyticsSvc,
	})
	searchHandler := handler.NewSearchHandler(handler.SearchHandlerDeps{
		Service:       querySvc,
		Client:        tsClient,
		TypesenseHost: cfg.TypesenseHost,
		// Use the bootstrapped search-only key as the HMAC parent for
		// scoped key generation. Typesense rejects scoped keys derived
		// from the master admin key.
		APIKey:       tsClient.SearchAPIKey(),
		ClickTracker: analyticsSvc,
		Logger:       slog.Default(),
	})
	slog.Info("search: query service wired",
		"hybrid_enabled", queryEmbedder != nil,
		"analytics_enabled", analyticsSvc != nil,
		"admin_stats_enabled", adminStats != nil,
		"ranking_enabled", rankingPipeline != nil,
		"ltr_capture_enabled", ltrRepo != nil && analyticsSvc != nil)
	return searchHandler, adminStats
}
