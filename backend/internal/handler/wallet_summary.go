package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// commissionProjector is the narrow contract the wallet summary
// depends on so worktrees without the referral feature still boot —
// when the projector is nil, the summary endpoint degrades to "no
// projections" rather than 500. The real *referralapp.Service
// satisfies the interface natively.
type commissionProjector interface {
	ProjectedCommissions(ctx context.Context, orgID uuid.UUID) ([]referralapp.ProjectedCommission, error)
}

// commissionRecorder reads the apporteur's recent commission rows so
// the unified timeline can interleave them with mission earnings. The
// real *referralapp.Service satisfies this via its existing
// ReferralWalletReader contract.
type commissionRecorder interface {
	RecentCommissions(ctx context.Context, referrerID uuid.UUID, limit int) ([]portservice.ReferralCommissionRecord, error)
}

// proposalTitleResolver returns the human-readable title for a proposal
// id so the unified timeline rows can render the mission title instead
// of "Sans titre". Narrow contract — only Title is consumed.
//
// The real *proposalapp.Service satisfies this via its GetProposalByID
// method (the returned proposal exposes Title). Wrapped in this narrow
// port so tests can drive the title-lookup path without standing up
// the full proposal stack, and so a worktree without the proposal
// feature wired still boots (titles degrade to empty).
type proposalTitleResolver interface {
	TitleForProposal(ctx context.Context, proposalID uuid.UUID) (string, error)
}

// missionWalletLoader is the narrow contract Summary depends on for
// the mission-side overview. Defined here so handler tests can drive
// the mission-leg + mission-title path without instantiating the full
// *paymentapp.Service. The real service satisfies it via its existing
// GetWalletOverview method.
type missionWalletLoader interface {
	GetWalletOverview(ctx context.Context, userID, orgID uuid.UUID) (*paymentapp.WalletOverview, error)
}

// WithCommissionProjector wires the projection reader so
// /wallet/summary surfaces "à venir" commissions. Builder pattern
// matches WithCommissionRetrier / WithKYCOnboardingURLResolver so the
// constructor signature stays stable when the referral feature is
// disabled.
func (h *WalletHandler) WithCommissionProjector(p commissionProjector) *WalletHandler {
	h.commissionProjector = p
	return h
}

// WithCommissionRecorder wires the apporteur's recent commission
// reader. The current production service satisfies both this and
// commissionProjector — they're split into two narrow ports so unit
// tests can drive each branch independently.
func (h *WalletHandler) WithCommissionRecorder(r commissionRecorder) *WalletHandler {
	h.commissionRecorder = r
	return h
}

// summaryBreakdownLeg is one side (missions or commissions) of the
// unified wallet summary. The wallet UI renders each leg as its own
// card with the same grammar.
type summaryBreakdownLeg struct {
	TotalCents       int64 `json:"total_cents"`
	AvailableCents   int64 `json:"available_cents"`
	EscrowedCents    int64 `json:"escrowed_cents"`
	TransmittedCents int64 `json:"transmitted_cents"`
}

// summaryBreakdown groups both legs of the unified summary.
type summaryBreakdown struct {
	Missions    summaryBreakdownLeg `json:"missions"`
	Commissions summaryBreakdownLeg `json:"commissions"`
}

// summaryTransaction is one row of the unified transaction timeline.
// Type is "mission" or "commission" — the UI uses it to pick the
// right icon and the deep-link target.
type summaryTransaction struct {
	Type         string    `json:"type"`
	AmountCents  int64     `json:"amount_cents"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	MissionTitle string    `json:"mission_title,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	ReferenceID  string    `json:"reference_id"`
}

// summaryResponse is the envelope returned by GET /wallet/summary.
type summaryResponse struct {
	Currency           string               `json:"currency"`
	TotalCents         int64                `json:"total_cents"`
	AvailableCents     int64                `json:"available_cents"`
	EscrowedCents      int64                `json:"escrowed_cents"`
	TransmittedCents   int64                `json:"transmitted_cents"`
	Breakdown          summaryBreakdown     `json:"breakdown"`
	RecentTransactions []summaryTransaction `json:"recent_transactions"`
	NextCursor         string               `json:"next_cursor,omitempty"`
}

// summaryCursor is the opaque base64-JSON cursor format used to
// paginate recent_transactions. Mirrors the rest of the codebase
// (cursor-based, never offset).
type summaryCursor struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
}

const (
	defaultSummaryTxLimit = 20
	maxSummaryTxLimit     = 100
)

// Summary returns the unified wallet view: mission earnings +
// apporteur commissions in a single envelope with a shared breakdown
// (total / available / escrowed / transmitted) and a merged
// recent_transactions timeline. The composition is read-only —
// no DB writes, no Stripe calls, no audit.
//
// Composition (wired via builders):
//   - paymentSvc.GetWalletOverview          — mission side
//   - commissionRecorder.RecentCommissions  — commission history
//   - commissionProjector.ProjectedCommissions — escrowed projections
//
// Any of the three can be nil; the response degrades gracefully
// (zero amounts / empty slices) rather than 500-ing.
func (h *WalletHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	limit := parseSummaryLimit(r.URL.Query().Get("limit"))
	cursor, cursorErr := decodeSummaryCursor(r.URL.Query().Get("cursor"))
	if cursorErr != nil {
		res.Error(w, http.StatusBadRequest, "invalid_cursor", "cursor format invalid")
		return
	}

	missions := h.loadMissionSide(r.Context(), userID, orgID)
	commissions := h.loadCommissionSide(r.Context(), userID, orgID)

	response := summaryResponse{
		Currency:  pickCurrency(missions, commissions),
		Breakdown: composeBreakdown(missions, commissions),
	}
	response.TotalCents = response.Breakdown.Missions.TotalCents + response.Breakdown.Commissions.TotalCents
	response.AvailableCents = response.Breakdown.Missions.AvailableCents + response.Breakdown.Commissions.AvailableCents
	response.EscrowedCents = response.Breakdown.Missions.EscrowedCents + response.Breakdown.Commissions.EscrowedCents
	response.TransmittedCents = response.Breakdown.Missions.TransmittedCents + response.Breakdown.Commissions.TransmittedCents

	transactions := buildTransactionTimeline(missions, commissions)
	page, next := paginateTransactions(transactions, cursor, limit)
	h.enrichWithProposalTitles(r.Context(), page, missions, commissions)
	response.RecentTransactions = page
	response.NextCursor = next

	res.JSON(w, http.StatusOK, map[string]any{"data": response})
}

// enrichWithProposalTitles walks the page of timeline entries and
// fills `mission_title` from the proposal service. Performs ONE
// lookup per unique proposal id (the natural deduplication keeps the
// fan-out bounded by the page size — typically 20). Errors are
// swallowed individually: a row whose title cannot be resolved keeps
// an empty title, and the UI falls back to the i18n "Sans titre".
//
// To avoid a fan-out on the proposal repo per page, we resolve
// proposal IDs from the source overview/commission slices BEFORE
// pagination — but only fetch the proposals that map to the page
// we're about to return, so missions with hundreds of records but
// limit=20 never hit 100 lookups.
func (h *WalletHandler) enrichWithProposalTitles(
	ctx context.Context,
	page []summaryTransaction,
	missions *paymentapp.WalletOverview,
	commissions commissionSideView,
) {
	if h.proposalTitles == nil || len(page) == 0 {
		return
	}
	// Build a reference_id → proposal_id map from the source slices.
	// Mission rows use record ID as ReferenceID; commission rows use
	// the commission UUID. Either way the proposal id is sourced from
	// the original entity, not parsed out of the timeline row.
	proposalByRef := map[string]uuid.UUID{}
	for _, r := range missions.Records {
		if r.ProposalID == "" {
			continue
		}
		pid, err := uuid.Parse(r.ProposalID)
		if err != nil {
			continue
		}
		proposalByRef[r.ID] = pid
	}
	for _, r := range commissions.records {
		if r.ProposalID == uuid.Nil {
			continue
		}
		proposalByRef[r.ID.String()] = r.ProposalID
	}
	// Resolve titles only for the proposals on the current page so a
	// long history never blows up the request — page size is bounded
	// by `limit` (default 20, hard cap 100).
	titlesByProposal := map[uuid.UUID]string{}
	for i, tx := range page {
		pid, ok := proposalByRef[tx.ReferenceID]
		if !ok {
			continue
		}
		title, seen := titlesByProposal[pid]
		if !seen {
			resolved, err := h.proposalTitles.TitleForProposal(ctx, pid)
			if err != nil {
				slog.Warn("wallet summary: proposal title lookup failed",
					"proposal_id", pid, "error", err)
				// Cache the empty result so a repeated id on the same
				// page does not re-hit the failing proposal lookup.
				titlesByProposal[pid] = ""
				continue
			}
			title = resolved
			titlesByProposal[pid] = title
		}
		page[i].MissionTitle = title
	}
}

// loadMissionSide fetches the provider wallet overview through the
// narrow missionWalletLoader port. Failures degrade to an empty
// overview — the commissions side still renders. Production wires
// *paymentapp.Service into the loader in NewWalletHandler; tests
// inject a fake via WithMissionWalletLoader.
func (h *WalletHandler) loadMissionSide(ctx context.Context, userID, orgID uuid.UUID) *paymentapp.WalletOverview {
	if h.missionWallet == nil {
		return &paymentapp.WalletOverview{}
	}
	wallet, err := h.missionWallet.GetWalletOverview(ctx, userID, orgID)
	if err != nil || wallet == nil {
		if err != nil {
			slog.Warn("wallet summary: mission side load failed",
				"user_id", userID, "org_id", orgID, "error", err)
		}
		return &paymentapp.WalletOverview{}
	}
	return wallet
}

// commissionSideView captures the four bits the breakdown / timeline
// builders need. Built from RecentCommissions + ProjectedCommissions
// so the rest of the file does not depend on either contract directly.
type commissionSideView struct {
	records     []portservice.ReferralCommissionRecord
	projections []referralapp.ProjectedCommission
}

// loadCommissionSide composes the apporteur commission view. Both
// the recorder and the projector are optional — when one is nil the
// other still contributes; when both are nil the view is empty.
func (h *WalletHandler) loadCommissionSide(ctx context.Context, userID, orgID uuid.UUID) commissionSideView {
	out := commissionSideView{}
	if h.commissionRecorder != nil {
		recs, err := h.commissionRecorder.RecentCommissions(ctx, userID, 100)
		if err != nil {
			slog.Warn("wallet summary: recent commissions load failed",
				"user_id", userID, "error", err)
		} else {
			out.records = recs
		}
	}
	if h.commissionProjector != nil {
		projections, err := h.commissionProjector.ProjectedCommissions(ctx, orgID)
		if err != nil {
			slog.Warn("wallet summary: projected commissions load failed",
				"org_id", orgID, "error", err)
		} else {
			out.projections = projections
		}
	}
	return out
}

// composeBreakdown derives the per-leg totals from the wallet
// overview + commission view.
func composeBreakdown(missions *paymentapp.WalletOverview, commissions commissionSideView) summaryBreakdown {
	return summaryBreakdown{
		Missions:    missionLeg(missions),
		Commissions: commissionLeg(commissions),
	}
}

// missionLeg flattens the mission-side overview into the leg shape.
func missionLeg(w *paymentapp.WalletOverview) summaryBreakdownLeg {
	if w == nil {
		return summaryBreakdownLeg{}
	}
	return summaryBreakdownLeg{
		TotalCents:       w.AvailableAmount + w.EscrowAmount + w.TransferredAmount,
		AvailableCents:   w.AvailableAmount,
		EscrowedCents:    w.EscrowAmount,
		TransmittedCents: w.TransferredAmount,
	}
}

// commissionLeg derives the commission-side totals from the
// projection stream — the canonical source of truth for commission
// aggregates.
//
// History: the previous implementation summed BOTH the DB records
// AND the SourceProjection projections, which double-counted any
// commission whose row had been persisted while a SourceProjection
// entry was emitted in parallel (e.g. attribution timing race, or an
// `approved` milestone whose row was missed by the per-milestone
// lookup → safety-net ProjectionPending). Symptom: a single 1298 €
// pending_kyc commission showed as 1298 € in BOTH the "séquestre"
// AND "disponible" cards.
//
// Fix: projections already cover EVERY state via dispatchMilestone:
//
//   - funded/submitted/disputed (active escrow) → ProjectionEscrowed
//     (source=projection).
//   - approved/released + commission row → SourceRow row carrying the
//     row's status (paid → ProjectionPaid, failed → ProjectionFailed,
//     pending_kyc/pending → ProjectionPending, clawed_back/cancelled
//     → ProjectionFailed bucket).
//   - approved/released + missing row → safety-net ProjectionPending
//     (source=projection).
//   - pending_funding / cancelled / refunded → SKIP.
//
// So the records loop is redundant — and worse, it double-counts.
// The records slice still feeds the recent_transactions timeline
// (records are the human-facing history), but aggregates derive only
// from the projection stream.
//
// Bucket mapping:
//   - ProjectionPaid       → TransmittedCents (money already at the apporteur)
//   - ProjectionEscrowed   → EscrowedCents     (locked, awaiting milestone release)
//   - ProjectionPending    → AvailableCents    (drainable via withdraw — UI shows "Retirer")
//   - ProjectionFailed     → AvailableCents    (retire-eligible too, same drain path)
//   - TotalCents = sum of all three.
func commissionLeg(view commissionSideView) summaryBreakdownLeg {
	leg := summaryBreakdownLeg{}
	for _, p := range view.projections {
		switch p.Status {
		case referralapp.ProjectionPaid:
			leg.TransmittedCents += p.ProjectedCents
		case referralapp.ProjectionEscrowed:
			leg.EscrowedCents += p.ProjectedCents
		case referralapp.ProjectionPending, referralapp.ProjectionFailed:
			leg.AvailableCents += p.ProjectedCents
		}
	}
	leg.TotalCents = leg.AvailableCents + leg.EscrowedCents + leg.TransmittedCents
	return leg
}

// pickCurrency picks the wallet's currency for the response envelope.
// Falls back to "EUR" — the marketplace is EUR-only for V1; the field
// is exposed so the contract is forward-compatible.
func pickCurrency(missions *paymentapp.WalletOverview, commissions commissionSideView) string {
	if len(commissions.records) > 0 && commissions.records[0].Currency != "" {
		return commissions.records[0].Currency
	}
	_ = missions
	return "EUR"
}

// buildTransactionTimeline merges mission + commission entries and
// sorts by OccurredAt DESC.
func buildTransactionTimeline(missions *paymentapp.WalletOverview, commissions commissionSideView) []summaryTransaction {
	out := make([]summaryTransaction, 0, len(missions.Records)+len(commissions.records))
	for _, r := range missions.Records {
		out = append(out, missionTransaction(r))
	}
	for _, r := range commissions.records {
		out = append(out, commissionTransaction(r))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].OccurredAt.After(out[j].OccurredAt)
	})
	return out
}

// missionTransaction maps a payment_record DTO onto the unified
// timeline shape.
func missionTransaction(r paymentapp.WalletRecord) summaryTransaction {
	occurred, _ := time.Parse("2006-01-02T15:04:05Z", r.CreatedAt)
	status := r.TransferStatus
	if r.MissionStatus != "" {
		status = r.MissionStatus
	}
	return summaryTransaction{
		Type:        "mission",
		AmountCents: r.ProviderPayout,
		Currency:    "EUR",
		Status:      status,
		OccurredAt:  occurred,
		ReferenceID: r.ID,
	}
}

// commissionTransaction maps a commission record onto the unified
// timeline shape, using PaidAt when available (more precise than
// CreatedAt for the "occurred" semantic) and falling back to CreatedAt
// otherwise.
func commissionTransaction(r portservice.ReferralCommissionRecord) summaryTransaction {
	occurred := r.CreatedAt
	if r.PaidAt != nil {
		occurred = *r.PaidAt
	}
	return summaryTransaction{
		Type:        "commission",
		AmountCents: r.CommissionCents,
		Currency:    fallbackCurrency(r.Currency),
		Status:      r.Status,
		OccurredAt:  occurred,
		ReferenceID: r.ID.String(),
	}
}

// fallbackCurrency returns EUR when c is empty (older rows might
// have been seeded without a currency).
func fallbackCurrency(c string) string {
	if c == "" {
		return "EUR"
	}
	return c
}

// parseSummaryLimit clamps the limit query param to [1, maxSummaryTxLimit].
// Defaults to defaultSummaryTxLimit when missing or invalid.
func parseSummaryLimit(raw string) int {
	if raw == "" {
		return defaultSummaryTxLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultSummaryTxLimit
	}
	if n > maxSummaryTxLimit {
		return maxSummaryTxLimit
	}
	return n
}

// decodeSummaryCursor decodes the opaque base64-JSON cursor. Empty
// input is a valid "first page" signal.
func decodeSummaryCursor(raw string) (*summaryCursor, error) {
	if raw == "" {
		return nil, nil
	}
	bytes, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	var c summaryCursor
	if err := json.Unmarshal(bytes, &c); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}
	return &c, nil
}

// encodeSummaryCursor produces the opaque cursor string from the
// last-emitted transaction on the page.
func encodeSummaryCursor(tx summaryTransaction) string {
	c := summaryCursor{
		OccurredAt: tx.OccurredAt,
		ID:         tx.ReferenceID,
	}
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

// paginateTransactions slices `all` according to cursor + limit and
// returns the page + the next_cursor (empty when no more rows).
// Stable + deterministic: a duplicate ID on the cursor boundary is
// skipped so paginating never duplicates rows.
func paginateTransactions(all []summaryTransaction, cursor *summaryCursor, limit int) ([]summaryTransaction, string) {
	if cursor != nil {
		// Skip everything strictly newer than the cursor (already shown
		// on a previous page) plus the cursor entry itself (matched by
		// reference id).
		filtered := make([]summaryTransaction, 0, len(all))
		for _, tx := range all {
			if tx.OccurredAt.After(cursor.OccurredAt) {
				continue
			}
			if tx.OccurredAt.Equal(cursor.OccurredAt) && tx.ReferenceID >= cursor.ID {
				continue
			}
			filtered = append(filtered, tx)
		}
		all = filtered
	}
	if len(all) <= limit {
		return all, ""
	}
	page := all[:limit]
	return page, encodeSummaryCursor(page[len(page)-1])
}
