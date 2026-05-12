package response

import (
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
)

// ReferralResponse is the public DTO for a single referral. BEFORE activation
// the `rate_pct` is ONLY visible to the referrer and the provider — the
// client must never see it (Modèle A). After activation, full identities are
// visible through the associated conversation; the DTO exposes the same
// shape regardless but the handler layer decides whether to redact or not.
type ReferralResponse struct {
	ID               uuid.UUID              `json:"id"`
	ReferrerID       uuid.UUID              `json:"referrer_id"`
	ProviderID       uuid.UUID              `json:"provider_id"`
	ClientID         uuid.UUID              `json:"client_id"`
	// ProviderDisplayName / ClientDisplayName carry the human label
	// (org name when the user owns an agency/enterprise org, "First
	// Last" otherwise). They are populated for the apporteur viewer
	// only — other viewers see the anonymised intro snapshot and the
	// reveal happens through conversation activation, not the DTO.
	ProviderDisplayName string                `json:"provider_display_name,omitempty"`
	ClientDisplayName   string                `json:"client_display_name,omitempty"`
	RatePct          *float64               `json:"rate_pct,omitempty"` // omitted when viewer is client pre-activation
	DurationMonths   int16                  `json:"duration_months"`
	Status           referral.Status        `json:"status"`
	Version          int                    `json:"version"`
	IntroSnapshot    referral.IntroSnapshot `json:"intro_snapshot"`
	IntroMessageForMe string                `json:"intro_message_for_me,omitempty"`
	ActivatedAt      *time.Time             `json:"activated_at,omitempty"`
	ExpiresAt        *time.Time             `json:"expires_at,omitempty"`
	LastActionAt    time.Time               `json:"last_action_at"`
	RejectionReason string                  `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// ReferralDisplayNames carries the optional human-readable labels the
// handler resolves at request time (via the app service's party
// display-name resolver). Both fields default to the empty string when
// the resolver is not wired or the lookup fails. Only attached to the
// DTO when the viewer is the apporteur (referrer) — the other parties
// see the anonymised snapshot and the names are exchanged through the
// activated conversation, not the DTO.
type ReferralDisplayNames struct {
	Provider string
	Client   string
}

// NewReferralResponse formats a referral for the given viewer, applying the
// rate-redaction rule: the client sees no rate until the intro is active.
// The intro_message_for_me field picks the right variant (provider or client)
// based on the viewer.
func NewReferralResponse(r *referral.Referral, viewerID uuid.UUID) ReferralResponse {
	return NewReferralResponseWithNames(r, viewerID, ReferralDisplayNames{})
}

// NewReferralResponseWithNames is the apporteur-aware variant. When the
// viewer is the referrer, the provider/client display names are
// included in the DTO so the page can render the simplified identity
// cards without an extra fetch. Other viewers receive the same DTO
// shape but with the names omitted (Modèle A confidentiality).
func NewReferralResponseWithNames(r *referral.Referral, viewerID uuid.UUID, names ReferralDisplayNames) ReferralResponse {
	out := ReferralResponse{
		ID:             r.ID,
		ReferrerID:     r.ReferrerID,
		ProviderID:     r.ProviderID,
		ClientID:       r.ClientID,
		DurationMonths: r.DurationMonths,
		Status:         r.Status,
		Version:        r.Version,
		IntroSnapshot:  r.IntroSnapshot,
		ActivatedAt:    r.ActivatedAt,
		ExpiresAt:      r.ExpiresAt,
		LastActionAt:   r.LastActionAt,
		RejectionReason: r.RejectionReason,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	// Rate visibility: referrer and provider always, client only after
	// activation (for the historical view once the intro has moved past the
	// decision phase). Client during pending phases must NEVER see the rate.
	if viewerID == r.ReferrerID || viewerID == r.ProviderID {
		rate := r.RatePct
		out.RatePct = &rate
	} else if viewerID == r.ClientID && r.Status == referral.StatusActive {
		rate := r.RatePct
		out.RatePct = &rate
	}

	// Pick the right pitch for the viewer.
	switch viewerID {
	case r.ProviderID:
		out.IntroMessageForMe = r.IntroMessageProvider
	case r.ClientID:
		out.IntroMessageForMe = r.IntroMessageClient
	case r.ReferrerID:
		// Referrer sees both pitches? Expose provider-side by default; the UI
		// can also read intro_snapshot for context.
		out.IntroMessageForMe = r.IntroMessageProvider
		// The apporteur (owner) view also gets the human-readable
		// provider + client names so the detail page can render a
		// purely informational, minimalist identity card instead of
		// the masked snapshot reserved for the other two viewers.
		out.ProviderDisplayName = names.Provider
		out.ClientDisplayName = names.Client
	}

	return out
}

// ReferralListResponse wraps a page of referrals with the project's cursor
// pagination envelope.
type ReferralListResponse struct {
	Items      []ReferralResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// NewReferralListResponse builds the paginated list DTO from a slice of
// domain referrals.
func NewReferralListResponse(rows []*referral.Referral, nextCursor string, viewerID uuid.UUID) ReferralListResponse {
	items := make([]ReferralResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, NewReferralResponse(r, viewerID))
	}
	return ReferralListResponse{Items: items, NextCursor: nextCursor}
}

// ReferralNegotiationResponse is the timeline row the dashboard renders.
type ReferralNegotiationResponse struct {
	ID        uuid.UUID                 `json:"id"`
	Version   int                       `json:"version"`
	ActorID   uuid.UUID                 `json:"actor_id"`
	ActorRole referral.ActorRole        `json:"actor_role"`
	Action    referral.NegotiationAction `json:"action"`
	RatePct   float64                   `json:"rate_pct"`
	Message   string                    `json:"message"`
	CreatedAt time.Time                 `json:"created_at"`
}

// NewNegotiationList formats a slice of negotiations for JSON output.
func NewNegotiationList(rows []*referral.Negotiation) []ReferralNegotiationResponse {
	out := make([]ReferralNegotiationResponse, 0, len(rows))
	for _, n := range rows {
		out = append(out, ReferralNegotiationResponse{
			ID:        n.ID,
			Version:   n.Version,
			ActorID:   n.ActorID,
			ActorRole: n.ActorRole,
			Action:    n.Action,
			RatePct:   n.RatePct,
			Message:   n.Message,
			CreatedAt: n.CreatedAt,
		})
	}
	return out
}

// AttributionResponse is the projection of one attribution for the
// referral detail page's "Missions pendant cette mise en relation"
// section. Includes the parent proposal's title + status and the
// aggregate commission stats. Commission amounts (paid, pending,
// escrow, clawed-back) are all stripped when the viewer is the client
// (Modèle A — the client never sees a rate or any commission number).
//
// milestones_total is the authoritative count of milestones the
// proposal has (≥ 1 by domain invariant). The legacy
// milestones_pending field is kept for backward compatibility but
// the UI now renders {paid}/{total} instead of {paid}/{paid+pending}
// because pending milestones that have not generated a commission
// row yet were invisible to the old math.
type AttributionResponse struct {
	ID                        uuid.UUID `json:"id"`
	ProposalID                uuid.UUID `json:"proposal_id"`
	ProposalTitle             string    `json:"proposal_title,omitempty"`
	ProposalStatus            string    `json:"proposal_status,omitempty"`
	// TotalAmountCents is the gross proposal amount (sum of milestones)
	// in cents. Surfaced on the apporteur detail page next to the
	// mission title so each row reads "1 230 € — Mission alpha" rather
	// than just the title. Visible to every viewer — it's the public
	// mission price, not a commission number, so Modèle A does not
	// require redaction. Defaults to 0 when the proposal lookup failed.
	TotalAmountCents          int64     `json:"total_amount_cents"`
	RatePctSnapshot           *float64  `json:"rate_pct_snapshot,omitempty"`
	AttributedAt              time.Time `json:"attributed_at"`
	// EndedAt is the RFC3339-formatted timestamp at which the
	// apporteur explicitly terminated this attribution via
	// `POST /referrals/attributions/{id}/end`. Nil when the
	// attribution is still active. WALLET-UNIFY Run D — surfaced
	// in the DTO so the web/mobile UI can render the "Intro
	// terminée" badge persistently after a page reload (the
	// mutation already returns it, but the list endpoint must too).
	EndedAt                   *string   `json:"ended_at,omitempty"`
	TotalCommissionCents      *int64    `json:"total_commission_cents,omitempty"`
	PendingCommissionCents    *int64    `json:"pending_commission_cents,omitempty"`
	EscrowCommissionCents     *int64    `json:"escrow_commission_cents,omitempty"`
	ClawedBackCommissionCents *int64    `json:"clawed_back_commission_cents,omitempty"`
	MilestonesPaid            int       `json:"milestones_paid"`
	MilestonesPending         int       `json:"milestones_pending"`
	MilestonesTotal           int       `json:"milestones_total"`
}

// NewAttributionListFromStats formats a slice of attribution+stats for
// JSON output. Hides commission amounts and rate from the client, since
// Modèle A confidentiality extends to the post-activation historical
// view as well.
func NewAttributionListFromStats(rows []attributionWithStats, viewerID uuid.UUID, clientID uuid.UUID) []AttributionResponse {
	out := make([]AttributionResponse, 0, len(rows))
	isClient := viewerID == clientID
	for _, r := range rows {
		row := AttributionResponse{
			ID:                r.Attribution.ID,
			ProposalID:        r.Attribution.ProposalID,
			ProposalTitle:     r.ProposalTitle,
			ProposalStatus:    r.ProposalStatus,
			TotalAmountCents:  r.ProposalAmountCents,
			AttributedAt:      r.Attribution.AttributedAt,
			MilestonesPaid:    r.MilestonesPaid,
			MilestonesPending: r.MilestonesPending,
			MilestonesTotal:   r.MilestonesTotal,
		}
		// WALLET-UNIFY Run D — expose ended_at so the per-attribution
		// "Intro terminée" badge persists across reloads on web+mobile.
		// The domain field is *time.Time; project to a string pointer
		// so JSON `omitempty` distinguishes "active" from "ended".
		if r.Attribution.EndedAt != nil {
			endedAt := r.Attribution.EndedAt.UTC().Format(time.RFC3339)
			row.EndedAt = &endedAt
		}
		if !isClient {
			rate := r.Attribution.RatePctSnapshot
			row.RatePctSnapshot = &rate
			paid := r.TotalCommissionCents
			pending := r.PendingCommissionCents
			escrow := r.EscrowCommissionCents
			clawed := r.ClawedBackCommissionCents
			row.TotalCommissionCents = &paid
			row.PendingCommissionCents = &pending
			row.EscrowCommissionCents = &escrow
			row.ClawedBackCommissionCents = &clawed
		}
		out = append(out, row)
	}
	return out
}

// attributionWithStats is a local alias for the enriched attribution
// row returned by the app service. Mirrors the app-layer struct shape
// without pulling in the app package — the handler layer just maps
// field-by-field.
type attributionWithStats = struct {
	Attribution               *referral.Attribution
	ProposalTitle             string
	ProposalStatus            string
	ProposalAmountCents       int64
	TotalCommissionCents      int64
	PendingCommissionCents    int64
	ClawedBackCommissionCents int64
	EscrowCommissionCents     int64
	MilestonesPaid            int
	MilestonesPending         int
	MilestonesTotal           int
}

// CommissionResponse is one commission row for the /commissions
// endpoint. Only the apporteur and the provider party can read it
// (handler enforces). Commission amount is the NET for the apporteur
// (already computed from gross × rate_pct at commission creation).
type CommissionResponse struct {
	ID               uuid.UUID                  `json:"id"`
	AttributionID    uuid.UUID                  `json:"attribution_id"`
	MilestoneID      uuid.UUID                  `json:"milestone_id"`
	GrossAmountCents int64                      `json:"gross_amount_cents"`
	CommissionCents  int64                      `json:"commission_cents"`
	Currency         string                     `json:"currency"`
	Status           referral.CommissionStatus  `json:"status"`
	StripeTransferID string                     `json:"stripe_transfer_id,omitempty"`
	StripeReversalID string                     `json:"stripe_reversal_id,omitempty"`
	FailureReason    string                     `json:"failure_reason,omitempty"`
	PaidAt           *time.Time                 `json:"paid_at,omitempty"`
	ClawedBackAt     *time.Time                 `json:"clawed_back_at,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
}

// NewCommissionList formats a slice of commissions for JSON output.
func NewCommissionList(rows []*referral.Commission) []CommissionResponse {
	out := make([]CommissionResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, CommissionResponse{
			ID:               c.ID,
			AttributionID:    c.AttributionID,
			MilestoneID:      c.MilestoneID,
			GrossAmountCents: c.GrossAmountCents,
			CommissionCents:  c.CommissionCents,
			Currency:         c.Currency,
			Status:           c.Status,
			StripeTransferID: c.StripeTransferID,
			StripeReversalID: c.StripeReversalID,
			FailureReason:    c.FailureReason,
			PaidAt:           c.PaidAt,
			ClawedBackAt:     c.ClawedBackAt,
			CreatedAt:        c.CreatedAt,
		})
	}
	return out
}
