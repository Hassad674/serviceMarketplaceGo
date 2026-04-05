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
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	notifdomain "marketplace-backend/internal/domain/notification"
	portservice "marketplace-backend/internal/port/service"
)

// NotificationSink is the minimal contract the Notifier needs from the
// notification service. Tests inject a mock.
type NotificationSink interface {
	Send(ctx context.Context, userID uuid.UUID, t notifdomain.NotificationType, title, body string, metadata map[string]any) error
}

// AccountLookup resolves a Stripe account_id back to the platform user_id.
// Returns uuid.Nil + error if no mapping found.
type AccountLookup interface {
	FindUserByStripeAccount(ctx context.Context, accountID string) (uuid.UUID, error)
}

// StateStore persists the last-seen state of a Stripe account so the
// Notifier can detect transitions between webhooks (activated ⇄ suspended,
// requirements list changed, etc.).
type StateStore interface {
	GetLast(ctx context.Context, accountID string) (*LastAccountState, error)
	SaveLast(ctx context.Context, accountID string, state *LastAccountState) error
}

// LastAccountState mirrors the fields of StripeAccountSnapshot we compare on.
type LastAccountState struct {
	ChargesEnabled     bool
	PayoutsEnabled     bool
	DetailsSubmitted   bool
	CurrentlyDueHash   string
	PastDueHash        string
	EventuallyDueHash  string
	ErrorCodes         []string
	DisabledReason     string
}

// Notifier dispatches user-facing notifications for Stripe account lifecycle
// events. Thread-safe: holds an in-memory cooldown map shared across calls.
type Notifier struct {
	notifications NotificationSink
	accounts      AccountLookup
	state         StateStore

	cooldown sync.Map // key: userID|type, val: time.Time (last sent)
	ttl      time.Duration
}

// NewNotifier wires the notifier. ttl sets the minimum interval between
// two identical notifications for the same user (to avoid spam when Stripe
// fires multiple account.updated webhooks back-to-back).
func NewNotifier(sink NotificationSink, accts AccountLookup, state StateStore, ttl time.Duration) *Notifier {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Notifier{
		notifications: sink,
		accounts:      accts,
		state:         state,
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

	userID, err := n.accounts.FindUserByStripeAccount(ctx, snap.AccountID)
	if err != nil {
		return fmt.Errorf("resolve user for account %s: %w", snap.AccountID, err)
	}

	prev, _ := n.state.GetLast(ctx, snap.AccountID)

	events := n.diff(prev, snap)
	for _, ev := range events {
		if !n.shouldSend(userID, ev.Key) {
			slog.Debug("embedded: notification skipped (cooldown)",
				"user_id", userID, "key", ev.Key)
			continue
		}
		if err := n.notifications.Send(ctx, userID, ev.Type, ev.Title, ev.Body, ev.Meta); err != nil {
			slog.Warn("embedded: send notification failed",
				"user_id", userID, "type", ev.Type, "error", err)
			continue
		}
		n.markSent(userID, ev.Key)
	}

	// Persist new state
	_ = n.state.SaveLast(ctx, snap.AccountID, snapshotToState(snap))
	return nil
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
func (n *Notifier) diff(prev *LastAccountState, cur *portservice.StripeAccountSnapshot) []notifEvent {
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

func (n *Notifier) shouldSend(userID uuid.UUID, key string) bool {
	k := userID.String() + "|" + key
	v, ok := n.cooldown.Load(k)
	if !ok {
		return true
	}
	last, _ := v.(time.Time)
	return time.Since(last) >= n.ttl
}

func (n *Notifier) markSent(userID uuid.UUID, key string) {
	k := userID.String() + "|" + key
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
