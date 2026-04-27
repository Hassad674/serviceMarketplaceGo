// Package embedded contains the notification dispatching logic for the
// Stripe Connect Embedded Components integration. It consumes webhook
// snapshots (built by adapter/stripe/webhook.go) and emits user-facing
// notifications via the shared notification service.
//
// This package is intentionally isolated from internal/app/payment so the
// classic custom-KYC path keeps working during the migration window.
package embedded

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/organization"
	portservice "marketplace-backend/internal/port/service"
)

// NotificationSink is the minimal contract the Notifier needs from the
// notification service. Tests inject a mock.
type NotificationSink interface {
	Send(ctx context.Context, userID uuid.UUID, t notifdomain.NotificationType, title, body string, metadata map[string]any) error
}

// OrgStore gives the Notifier the org-keyed Stripe state operations it
// needs. Since phase R5 the Stripe Connect account is owned by the
// organization, so the notifier resolves incoming webhooks against an
// org id and persists the diffing snapshot under the same key.
//
// Production: wire with repository.OrganizationRepository.
// Tests: implement with an in-memory fake.
type OrgStore interface {
	// FindByStripeAccountID returns enough of the org to identify its
	// owner (so a notification can be sent to a real user id) plus the
	// id itself (so state reads / writes know which row to touch).
	FindByStripeAccountID(ctx context.Context, accountID string) (*organization.Organization, error)
	GetStripeLastState(ctx context.Context, orgID uuid.UUID) ([]byte, error)
	SaveStripeLastState(ctx context.Context, orgID uuid.UUID, state []byte) error
}

// LastAccountState mirrors the fields of StripeAccountSnapshot we compare on.
//
// HasEverActivated is a sticky one-way latch: once we've observed the
// account at ChargesEnabled=true OR PayoutsEnabled=true, this stays true
// forever. It gates the requirement-class notifications (currently_due,
// eventually_due, past_due, document errors) so they don't spam the user
// during the very first onboarding pass — Stripe always returns a long
// list of pending requirements for a freshly-created account, and the
// KYC page itself already surfaces them. After the first activation,
// subsequent requirement changes ARE meaningful and resume normal
// dispatch.
//
// HasPayoutsEverActivated is the same idea but specifically for the
// payouts capability — used to fire the positive "Virements sortants
// activés" notification exactly once on the first PayoutsEnabled=true.
type LastAccountState struct {
	ChargesEnabled          bool
	PayoutsEnabled          bool
	DetailsSubmitted        bool
	CurrentlyDueHash        string
	PastDueHash             string
	EventuallyDueHash       string
	ErrorCodes              []string
	DisabledReason          string
	HasEverActivated        bool
	HasPayoutsEverActivated bool
}

// Notifier dispatches user-facing notifications for Stripe account lifecycle
// events. Thread-safe: holds an in-memory cooldown map shared across calls.
type Notifier struct {
	notifications NotificationSink
	orgs          OrgStore

	cooldown sync.Map // key: orgID|type, val: time.Time (last sent)
	ttl      time.Duration

	// Referral KYC listener — wired post-construction. Fires whenever the
	// account transitions to a payable state so the referral feature can
	// drain parked pending_kyc commissions for the owner user.
	referralKYCListener portservice.ReferralKYCListener
}

// SetReferralKYCListener plugs the referral KYC listener in post-construction.
// Safe to call at app startup. Passing nil disables the hook.
func (n *Notifier) SetReferralKYCListener(l portservice.ReferralKYCListener) {
	n.referralKYCListener = l
}

// NewNotifier wires the notifier. ttl sets the minimum interval between
// two identical notifications for the same org (to avoid spam when
// Stripe fires multiple account.updated webhooks back-to-back).
func NewNotifier(sink NotificationSink, orgs OrgStore, ttl time.Duration) *Notifier {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Notifier{
		notifications: sink,
		orgs:          orgs,
		ttl:           ttl,
	}
}

// HandleAccountSnapshot is the main entry point. It:
//  1. resolves the user from account_id
//  2. loads the previous state
//  3. computes the diff and emits one notification per meaningful change
//  4. persists the new state
//
// Called from the stripe webhook handler for account.updated +
// capability.updated events.
func (n *Notifier) HandleAccountSnapshot(ctx context.Context, snap *portservice.StripeAccountSnapshot) error {
	if snap == nil || snap.AccountID == "" {
		return fmt.Errorf("nil or empty snapshot")
	}

	org, err := n.orgs.FindByStripeAccountID(ctx, snap.AccountID)
	if err != nil {
		return fmt.Errorf("resolve org for account %s: %w", snap.AccountID, err)
	}

	prev := n.loadPrevState(ctx, org.ID)

	// Notifications still target a user (the notification inbox is
	// per-operator). V1: route to the org's current owner. R6+ can
	// fan out to all members with the wallet.view permission.
	recipientUserID := org.OwnerUserID

	// hasEverActivated is the sticky onboarding-complete latch: true
	// once we've observed the account active in any prior webhook.
	// Requirement-class notifs are gated behind it so a freshly-created
	// Stripe Connect account doesn't trigger a flood of "11 informations
	// requises" alerts before the user has even started filling forms.
	hasEverActivated := prev != nil && prev.HasEverActivated
	hasPayoutsEverActivated := prev != nil && prev.HasPayoutsEverActivated

	events := n.diff(prev, snap, hasEverActivated, hasPayoutsEverActivated)
	for _, ev := range events {
		if !n.shouldSend(org.ID, ev.Key) {
			slog.Debug("embedded: notification skipped (cooldown)",
				"org_id", org.ID, "key", ev.Key)
			continue
		}
		if err := n.notifications.Send(ctx, recipientUserID, ev.Type, ev.Title, ev.Body, ev.Meta); err != nil {
			slog.Warn("embedded: send notification failed",
				"org_id", org.ID, "type", ev.Type, "error", err)
			continue
		}
		n.markSent(org.ID, ev.Key)
	}

	// Referral KYC drain — trigger when the account transitions to payable.
	// "Payable" is defined as PayoutsEnabled=true AND ChargesEnabled=true.
	// Prev state nil means this is the first webhook we've seen, which in
	// practice happens right after onboarding completion, so we fire then too.
	if n.referralKYCListener != nil && snap.PayoutsEnabled && snap.ChargesEnabled {
		prevWasPayable := prev != nil && prev.PayoutsEnabled && prev.ChargesEnabled
		if !prevWasPayable {
			if err := n.referralKYCListener.OnStripeAccountReady(ctx, org.OwnerUserID); err != nil {
				slog.Warn("referral: kyc drain failed",
					"org_id", org.ID, "owner_user_id", org.OwnerUserID, "error", err)
			}
		}
	}

	// Persist the new state on the org row so the next webhook can
	// diff against it. The HasEverActivated / HasPayoutsEverActivated
	// latches survive across snapshots — once true, always true.
	newState := snapshotToState(snap)
	if hasEverActivated || snap.ChargesEnabled || snap.PayoutsEnabled {
		newState.HasEverActivated = true
	}
	if hasPayoutsEverActivated || snap.PayoutsEnabled {
		newState.HasPayoutsEverActivated = true
	}
	n.savePrevState(ctx, org.ID, newState)
	return nil
}

// loadPrevState reads the org's last snapshot from OrgStore and
// unmarshals it. Returns nil on any error (first webhook, corrupt
// JSON, etc.) — the diff() helper treats nil as "no prior state,
// emit notifs".
func (n *Notifier) loadPrevState(ctx context.Context, orgID uuid.UUID) *LastAccountState {
	raw, err := n.orgs.GetStripeLastState(ctx, orgID)
	if err != nil || len(raw) == 0 {
		return nil
	}
	var state LastAccountState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil
	}
	return &state
}

// savePrevState marshals and persists the current snapshot. Best-effort:
// logs a warning on failure but never fails the webhook dispatch.
func (n *Notifier) savePrevState(ctx context.Context, orgID uuid.UUID, state *LastAccountState) {
	raw, err := json.Marshal(state)
	if err != nil {
		slog.Warn("embedded: marshal last_state", "org_id", orgID, "error", err)
		return
	}
	if err := n.orgs.SaveStripeLastState(ctx, orgID, raw); err != nil {
		slog.Warn("embedded: persist last_state", "org_id", orgID, "error", err)
	}
}

/* ----------------------------- Diffing ----------------------------- */

type notifEvent struct {
	Key   string // used for cooldown bucketing
	Type  notifdomain.NotificationType
	Title string
	Body  string
	Meta  map[string]any
}

// diff compares previous vs current state and returns the notifications
// that should be emitted (deduped).
//
// hasEverActivated gates the requirement-class notifications: when false
// (i.e. the user is in their initial KYC onboarding pass and hasn't yet
// reached an active state), Stripe's currently_due / eventually_due /
// past_due lists are noise — the page already surfaces them, the bell
// shouldn't compete. Bad-state events (account suspended, document
// rejected) still fire regardless because they can only happen after
// the user is past onboarding anyway.
//
// hasPayoutsEverActivated gates the positive "Virements sortants
// activés" notification so it fires exactly once on the very first
// PayoutsEnabled=true transition, never on later same-value webhooks.
func (n *Notifier) diff(prev *LastAccountState, cur *portservice.StripeAccountSnapshot, hasEverActivated, hasPayoutsEverActivated bool) []notifEvent {
	var out []notifEvent

	curState := snapshotToState(cur)

	// 1. Charges/payouts activation transitions
	if prev == nil || prev.ChargesEnabled != cur.ChargesEnabled || prev.PayoutsEnabled != cur.PayoutsEnabled {
		if cur.ChargesEnabled && cur.PayoutsEnabled {
			out = append(out, notifEvent{
				Key:   "account_active",
				Type:  notifdomain.TypeStripeAccountStatus,
				Title: "Compte de paiement activé",
				Body:  "Votre compte est pleinement opérationnel : vous pouvez recevoir des paiements et être rémunéré.",
				Meta:  map[string]any{"account_id": cur.AccountID, "status": "active"},
			})
		} else if prev != nil && prev.ChargesEnabled && !cur.ChargesEnabled {
			out = append(out, notifEvent{
				Key:   "charges_disabled",
				Type:  notifdomain.TypeStripeAccountStatus,
				Title: "Paiements entrants suspendus",
				Body:  "Stripe a temporairement suspendu vos paiements. Consultez les actions requises pour rétablir votre compte.",
				Meta:  map[string]any{"account_id": cur.AccountID, "status": "charges_disabled"},
			})
		} else if prev != nil && prev.PayoutsEnabled && !cur.PayoutsEnabled {
			out = append(out, notifEvent{
				Key:   "payouts_disabled",
				Type:  notifdomain.TypeStripeAccountStatus,
				Title: "Virements sortants suspendus",
				Body:  "Les virements vers votre banque sont temporairement suspendus. Vérifiez les informations requises.",
				Meta:  map[string]any{"account_id": cur.AccountID, "status": "payouts_disabled"},
			})
		}
	}

	// 1b. First-ever payouts activation — positive notif. Fires once,
	// the first time we see PayoutsEnabled=true. The hasPayouts*
	// latch is updated by the caller AFTER this diff returns, so
	// repeated webhooks with PayoutsEnabled=true don't re-fire.
	if cur.PayoutsEnabled && !hasPayoutsEverActivated {
		out = append(out, notifEvent{
			Key:   "payouts_first_activated",
			Type:  notifdomain.TypeStripeAccountStatus,
			Title: "Virements sortants activés — tu peux maintenant retirer ton solde",
			Body:  "Votre compte est désormais habilité à recevoir des virements vers votre banque. Vous pouvez retirer votre solde quand vous le souhaitez.",
			Meta:  map[string]any{"account_id": cur.AccountID, "status": "payouts_activated"},
		})
	}

	// 2-5. Requirement-class notifications — only fire when the user
	// has previously reached an active state. During the very first
	// onboarding pass, Stripe's requirements list is not actionable
	// noise: the user is filling the KYC form, the page itself
	// surfaces the requirements, and the notification bell would
	// just compete. The truly bad-state events (disabled_reason,
	// section 6 below) keep firing regardless.
	if hasEverActivated {
		// 2. Requirements added (new currently_due that wasn't there before)
		if curState.CurrentlyDueHash != "" &&
			(prev == nil || prev.CurrentlyDueHash != curState.CurrentlyDueHash) {
			count := len(cur.CurrentlyDue)
			if count > 0 {
				out = append(out, notifEvent{
					Key:   "requirements_currently_due",
					Type:  notifdomain.TypeStripeRequirements,
					Title: fmt.Sprintf("%d information%s requise%s", count, pluralS(count), pluralS(count)),
					Body:  "Stripe a besoin d'informations complémentaires pour maintenir votre compte actif.",
					Meta: map[string]any{
						"account_id":    cur.AccountID,
						"currently_due": cur.CurrentlyDue,
						"count":         count,
					},
				})
			}
		}

		// 3. Eventually due (non-urgent, anticipated requirements)
		if curState.EventuallyDueHash != "" &&
			(prev == nil || prev.EventuallyDueHash != curState.EventuallyDueHash) {
			count := len(cur.EventuallyDue)
			if count > 0 {
				out = append(out, notifEvent{
					Key:   "requirements_eventually_due",
					Type:  notifdomain.TypeStripeRequirements,
					Title: fmt.Sprintf("%d information%s à prévoir", count, pluralS(count)),
					Body:  "Stripe anticipe des informations qui seront nécessaires pour maintenir votre compte à jour. Vous pouvez les fournir dès maintenant.",
					Meta: map[string]any{
						"account_id":     cur.AccountID,
						"eventually_due": cur.EventuallyDue,
						"count":          count,
						"urgent":         false,
					},
				})
			}
		}

		// 4. Past due (urgent)
		if curState.PastDueHash != "" &&
			(prev == nil || prev.PastDueHash != curState.PastDueHash) {
			count := len(cur.PastDue)
			if count > 0 {
				out = append(out, notifEvent{
					Key:   "requirements_past_due",
					Type:  notifdomain.TypeStripeRequirements,
					Title: "Action urgente — délai dépassé",
					Body:  fmt.Sprintf("%d information%s n'a%s pas été fournie%s à temps. Votre compte risque d'être suspendu.", count, pluralS(count), pluralS(count), pluralS(count)),
					Meta: map[string]any{
						"account_id": cur.AccountID,
						"past_due":   cur.PastDue,
						"urgent":     true,
					},
				})
			}
		}

		// 5. Document rejected (new error appeared)
		newErrors := diffErrors(prev, curState)
		for _, ec := range newErrors {
			title, body := errorMessageFor(ec.Code)
			out = append(out, notifEvent{
				Key:   "doc_error_" + ec.Code,
				Type:  notifdomain.TypeStripeRequirements,
				Title: title,
				Body:  body,
				Meta: map[string]any{
					"account_id":  cur.AccountID,
					"requirement": ec.Requirement,
					"code":        ec.Code,
					"reason":      ec.Reason,
				},
			})
		}
	}

	// 6. Account disabled
	if cur.DisabledReason != "" &&
		(prev == nil || prev.DisabledReason != cur.DisabledReason) {
		out = append(out, notifEvent{
			Key:   "account_disabled_" + cur.DisabledReason,
			Type:  notifdomain.TypeStripeAccountStatus,
			Title: "Compte restreint par Stripe",
			Body:  fmt.Sprintf("Motif : %s. Consultez la section paiements pour résoudre la situation.", humanizeDisabledReason(cur.DisabledReason)),
			Meta: map[string]any{
				"account_id":      cur.AccountID,
				"disabled_reason": cur.DisabledReason,
			},
		})
	}

	return out
}

/* --------------------------- Cooldown map --------------------------- */

func (n *Notifier) shouldSend(orgID uuid.UUID, key string) bool {
	k := orgID.String() + "|" + key
	v, ok := n.cooldown.Load(k)
	if !ok {
		return true
	}
	last, _ := v.(time.Time)
	return time.Since(last) >= n.ttl
}

func (n *Notifier) markSent(orgID uuid.UUID, key string) {
	k := orgID.String() + "|" + key
	n.cooldown.Store(k, time.Now())
}

/* --------------------------- Helpers --------------------------- */

func snapshotToState(snap *portservice.StripeAccountSnapshot) *LastAccountState {
	errCodes := make([]string, 0, len(snap.RequirementErrors))
	for _, e := range snap.RequirementErrors {
		errCodes = append(errCodes, e.Requirement+":"+e.Code)
	}
	return &LastAccountState{
		ChargesEnabled:    snap.ChargesEnabled,
		PayoutsEnabled:    snap.PayoutsEnabled,
		DetailsSubmitted:  snap.DetailsSubmitted,
		CurrentlyDueHash:  hashFields(snap.CurrentlyDue),
		PastDueHash:       hashFields(snap.PastDue),
		EventuallyDueHash: hashFields(snap.EventuallyDue),
		ErrorCodes:        errCodes,
		DisabledReason:    snap.DisabledReason,
	}
}

func hashFields(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	// Simple stable-order hash: sort + join
	sorted := make([]string, len(fields))
	copy(sorted, fields)
	// Manual insertion sort (small arrays)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	return strings.Join(sorted, "|")
}

func diffErrors(prev *LastAccountState, cur *LastAccountState) []struct {
	Requirement string
	Code        string
	Reason      string
} {
	prevSet := map[string]struct{}{}
	if prev != nil {
		for _, c := range prev.ErrorCodes {
			prevSet[c] = struct{}{}
		}
	}
	var out []struct {
		Requirement string
		Code        string
		Reason      string
	}
	for _, c := range cur.ErrorCodes {
		if _, seen := prevSet[c]; seen {
			continue
		}
		parts := strings.SplitN(c, ":", 2)
		req, code := "", c
		if len(parts) == 2 {
			req, code = parts[0], parts[1]
		}
		out = append(out, struct {
			Requirement string
			Code        string
			Reason      string
		}{Requirement: req, Code: code})
	}
	return out
}

// errorMessageFor maps a Stripe requirement error code to a user-facing
// title+body in French. Covers the most common codes; unknown codes fall
// back to a generic message.
func errorMessageFor(code string) (string, string) {
	switch code {
	case "verification_document_expired":
		return "Document expiré",
			"Le document fourni a expiré. Veuillez en fournir un valide."
	case "verification_document_too_blurry",
		"verification_document_not_readable":
		return "Document illisible",
			"La photo du document est floue ou illisible. Veuillez la reprendre."
	case "verification_document_name_mismatch",
		"verification_document_nationality_mismatch":
		return "Informations du document non conformes",
			"Les informations du document ne correspondent pas à celles saisies. Vérifiez et réessayez."
	case "verification_document_fraudulent",
		"verification_document_manipulated":
		return "Document refusé",
			"Le document fourni n'a pas pu être validé. Merci d'en fournir un autre."
	case "verification_failed_address_match":
		return "Adresse non vérifiée",
			"Votre adresse n'a pas pu être vérifiée. Vérifiez la saisie."
	case "verification_failed_id_number_match":
		return "Numéro d'identification invalide",
			"Le numéro d'identification ne correspond pas. Vérifiez la saisie."
	case "invalid_value_other":
		return "Valeur invalide",
			"Une information fournie est invalide. Consultez les détails dans votre tableau de bord."
	default:
		return "Action requise sur votre compte",
			"Une information complémentaire est nécessaire pour maintenir votre compte actif."
	}
}

func humanizeDisabledReason(code string) string {
	switch code {
	case "requirements.past_due":
		return "informations requises non fournies à temps"
	case "requirements.pending_verification":
		return "vérification en cours"
	case "listed":
		return "compte sur une liste de vérification"
	case "platform_paused":
		return "pause volontaire de la plateforme"
	case "rejected.fraud":
		return "suspicion de fraude"
	case "rejected.terms_of_service":
		return "violation des conditions d'utilisation"
	case "rejected.listed":
		return "compte listé"
	case "rejected.other":
		return "rejeté par Stripe"
	case "under_review":
		return "compte en cours de revue"
	case "other":
		return "raison non précisée"
	default:
		return code
	}
}

func pluralS(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
