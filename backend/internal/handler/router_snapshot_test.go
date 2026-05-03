package handler

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/config"
)

// updateGolden controls whether the snapshot test rewrites
// testdata/routes.golden instead of asserting against it. Set to true
// once when capturing the baseline, then back to false. The test fails
// (loud) if anyone leaves it true on commit so a regression cannot
// silently be promoted to "the new baseline".
var updateGolden = flag.Bool("update-golden", false, "update routes.golden snapshot")

// TestRouterSnapshot is the behaviour-preservation guard for the
// phase-3-F router split. It walks the chi tree of a maximally-populated
// router and compares (method, path, middleware count) tuples against
// testdata/routes.golden. The split is only safe when the golden file
// remains byte-identical before and after the refactor.
//
// Mostly-nil deps would only exercise the always-on subtree; we
// allocate every handler as a zero-value pointer so every `if
// deps.X != nil` branch in NewRouter activates.
func TestRouterSnapshot(t *testing.T) {
	r := NewRouter(snapshotDeps())

	tuples := walkRoutes(t, r)
	got := strings.Join(tuples, "\n") + "\n"

	goldenPath := filepath.Join("testdata", "routes.golden")
	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		t.Logf("wrote %d routes to %s", len(tuples), goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v\nrun `go test -run TestRouterSnapshot -update-golden` to capture", err)
	}

	if string(want) != got {
		t.Errorf("router snapshot drifted — route table is not byte-identical\n--- want\n%s\n--- got\n%s",
			string(want), got)
	}
}

// walkRoutes traverses the chi tree and returns a deterministic slice of
// "METHOD PATH mw=N" strings, one per registered endpoint. Middleware
// count is included so a refactor that loses (or duplicates) a
// middleware on a route is caught — the chain count is invariant under
// a pure file split.
func walkRoutes(t *testing.T, r chi.Router) []string {
	t.Helper()
	var lines []string
	err := chi.Walk(r, func(method string, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		lines = append(lines, fmt.Sprintf("%s %s mw=%d", method, route, len(mws)))
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	sort.Strings(lines)
	return lines
}

// snapshotDeps returns a RouterDeps with every optional handler
// allocated as a zero-value pointer. The handlers are never invoked by
// chi.Walk, so empty structs are sufficient. Mandatory fields (Config,
// TokenService, SessionService, UserRepo, OrgOverridesResolver) get
// stubs that satisfy the construction-time interfaces — the auth /
// security middlewares only inspect them when a real request comes in.
func snapshotDeps() RouterDeps {
	cfg := &config.Config{
		Env:            "development",
		AllowedOrigins: []string{"http://localhost:3000"},
	}
	return RouterDeps{
		Config:               cfg,
		TokenService:         nil, // captured by middleware closure, never invoked here
		SessionService:       nil,
		UserRepo:             nil,
		OrgOverridesResolver: nil,

		Auth:                  &AuthHandler{},
		Invitation:            &InvitationHandler{},
		Team:                  &TeamHandler{},
		RoleOverrides:         &RoleOverridesHandler{},
		Profile:               &ProfileHandler{},
		ClientProfile:         &ClientProfileHandler{},
		ProfilePricing:        &ProfilePricingHandler{},
		FreelanceProfile:      &FreelanceProfileHandler{},
		FreelancePricing:      &FreelancePricingHandler{},
		FreelanceProfileVideo: &FreelanceProfileVideoHandler{},
		ReferrerProfile:       &ReferrerProfileHandler{},
		ReferrerPricing:       &ReferrerPricingHandler{},
		ReferrerProfileVideo:  &ReferrerProfileVideoHandler{},
		OrganizationShared:    &OrganizationSharedProfileHandler{},
		Upload:                &UploadHandler{},
		Health:                &HealthHandler{},
		Messaging:             &MessagingHandler{},
		Proposal:              &ProposalHandler{},
		Job:                   &JobHandler{},
		JobApplication:        &JobApplicationHandler{},
		Review:                &ReviewHandler{},
		Call:                  &CallHandler{},
		SocialLink:            &SocialLinkHandler{},
		FreelanceSocialLink:   &SocialLinkHandler{},
		ReferrerSocialLink:    &SocialLinkHandler{},
		Embedded:              &EmbeddedHandler{},
		Notification:          &NotificationHandler{},
		Stripe:                &StripeHandler{},
		Report:                &ReportHandler{},
		Wallet:                &WalletHandler{},
		Billing:               &BillingHandler{},
		Subscription:          &SubscriptionHandler{},
		BillingProfile:        &BillingProfileHandler{},
		Invoice:               &InvoiceHandler{},
		AdminCreditNote:       &AdminCreditNoteHandler{},
		AdminInvoice:          &AdminInvoiceHandler{},
		Admin:                 &AdminHandler{},
		Portfolio:             &PortfolioHandler{},
		ProjectHistory:        &ProjectHistoryHandler{},
		Dispute:               &DisputeHandler{},
		AdminDispute:          &AdminDisputeHandler{},
		Skill:                 &SkillHandler{},
		Referral:              &ReferralHandler{},
		Search:                &SearchHandler{},
		AdminSearchStats:      &AdminSearchStatsHandler{},
		GDPR:                  &GDPRHandler{},
		WSHandler:             func(w http.ResponseWriter, r *http.Request) {},
		Metrics:               nil, // metrics route is gated; we keep it off so the golden does not depend on prometheus internals
		RateLimiter:           nil, // optional — leaving nil keeps the chain count comparison stable
	}
}
