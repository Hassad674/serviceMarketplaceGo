package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	gdprapp "marketplace-backend/internal/app/gdpr"
	domaingdpr "marketplace-backend/internal/domain/gdpr"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"
	res "marketplace-backend/pkg/response"
)

// GDPRHandler exposes the four right-to-erasure / right-to-export
// endpoints. The handler is purely a thin HTTP layer — every business
// rule lives in app/gdpr.
//
// Routes (mounted under /api/v1):
//
//	GET  /me/export                       (auth)         → ZIP file
//	POST /me/account/request-deletion     (auth)         → 200 / 401 / 409
//	GET  /me/account/confirm-deletion     (token query)  → 200 / 401
//	POST /me/account/cancel-deletion      (auth)         → 200
type GDPRHandler struct {
	svc *gdprapp.Service
}

func NewGDPRHandler(svc *gdprapp.Service) *GDPRHandler {
	return &GDPRHandler{svc: svc}
}

// Export builds the export aggregate, writes a ZIP to the response,
// and streams the bytes back. The ZIP is built fully in memory before
// the first byte is written so an error mid-build can still surface
// as a JSON 500 instead of a half-written archive on the wire.
func (h *GDPRHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	export, err := h.svc.ExportData(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUserNotFound):
			res.Error(w, http.StatusNotFound, "user_not_found", err.Error())
		case errors.Is(err, user.ErrAccountScheduledForDeletion):
			res.Error(w, http.StatusGone, "account_scheduled_for_deletion",
				"Account is scheduled for deletion; cancel the deletion to export your data.")
		default:
			slog.Error("gdpr export build", "error", err.Error())
			res.Error(w, http.StatusInternalServerError, "internal_error", "could not build export")
		}
		return
	}

	buf := &bytes.Buffer{}
	if err := writeZIP(buf, export); err != nil {
		slog.Error("gdpr zip write", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "could not encode export")
		return
	}

	timestamp := export.Timestamp.UTC().Format("20060102-150405")
	filename := fmt.Sprintf("marketplace-export-%s-%s.zip", export.UserID.String(), timestamp)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// writeZIP encodes the export aggregate into the writer per Decision 1.
// The ZIP contains:
//   - manifest.json  (version, timestamp, file list)
//   - README.txt     (FR + EN explanation of every file)
//   - one .json per domain section
func writeZIP(w *bytes.Buffer, exp *domaingdpr.Export) error {
	zw := zip.NewWriter(w)

	manifest := map[string]any{
		"version":   domaingdpr.ExportVersion,
		"user_id":   exp.UserID.String(),
		"email":     exp.Email,
		"locale":    exp.Locale,
		"timestamp": exp.Timestamp.UTC().Format(time.RFC3339),
		"files":     exp.FileNames(),
	}
	if err := writeJSONFile(zw, "manifest.json", manifest); err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	readmeContent := renderReadme(exp.Locale)
	if err := writeRawFile(zw, "README.txt", []byte(readmeContent)); err != nil {
		return fmt.Errorf("readme: %w", err)
	}

	for _, name := range exp.FileNames() {
		section := exp.SectionFor(name)
		if section == nil {
			section = []map[string]any{}
		}
		if err := writeJSONFile(zw, name, section); err != nil {
			return fmt.Errorf("section %s: %w", name, err)
		}
	}
	return zw.Close()
}

func writeJSONFile(zw *zip.Writer, name string, payload any) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func writeRawFile(zw *zip.Writer, name string, data []byte) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

// renderReadme returns the bilingual README text. Decision 1: every
// export carries a human-readable explanation of what each JSON file
// contains so the recipient does not need to read the OpenAPI spec.
func renderReadme(locale string) string {
	en := `MARKETPLACE SERVICE — DATA EXPORT (RGPD / GDPR)
================================================

This archive contains every personal datum the platform holds for
your account, in JSON format. Each file can be opened in Excel
(via "Get Data → JSON") or any text editor.

Files:
  manifest.json     metadata about this export (version, timestamp)
  profile.json      your user profile + organization
  proposals.json    proposals you authored or received
  messages.json     messages you sent
  invoices.json     invoices addressed to your organization
  reviews.json      reviews you wrote or received
  notifications.json  in-app notifications addressed to you
  jobs.json         jobs you published
  portfolios.json   portfolio items of organizations you belong to
  reports.json      moderation reports involving you
  audit_logs.json   security events for your account

Retention: this export was generated on demand. We do NOT store a copy.
If you re-request, a fresh archive is built from current data.

Right to erasure: to delete your account permanently, visit
/account/delete in your dashboard.
`
	fr := `SERVICE MARKETPLACE — EXPORT DE DONNÉES (RGPD)
==================================================

Cette archive contient l'intégralité des données personnelles que la
plateforme détient pour votre compte, au format JSON. Chaque fichier
peut être ouvert dans Excel (via "Données → À partir d'un JSON") ou
tout éditeur de texte.

Fichiers :
  manifest.json     métadonnées de cet export (version, horodatage)
  profile.json      votre profil utilisateur + organisation
  proposals.json    propositions que vous avez écrites ou reçues
  messages.json     messages que vous avez envoyés
  invoices.json     factures adressées à votre organisation
  reviews.json      avis que vous avez écrits ou reçus
  notifications.json  notifications in-app qui vous étaient adressées
  jobs.json         offres que vous avez publiées
  portfolios.json   éléments de portfolio des organisations dont vous êtes membre
  reports.json      signalements de modération vous impliquant
  audit_logs.json   événements de sécurité de votre compte

Conservation : cet export est généré à la demande. Nous ne conservons
PAS de copie. Une nouvelle demande génère une archive fraîche.

Droit à l'effacement : pour supprimer votre compte définitivement,
rendez-vous sur /account/delete depuis votre tableau de bord.
`
	if locale == "en" {
		return en + "\n---\n\n" + fr
	}
	return fr + "\n---\n\n" + en
}

// RequestDeletion is the password-protected entry point the frontend
// calls when the user clicks "Delete account" in the settings panel.
// The handler verifies the password via the service, sends the
// confirmation email, and returns the email address echoed back so
// the UX can show "we sent an email to xx@yy.com — check your inbox".
//
// Status codes:
//
//	200  email sent (or re-sent for an already-scheduled account)
//	401  wrong password
//	404  user not found (auth said yes but DB lost the row)
//	409  user owns one or more orgs with active members
//	500  service error
func (h *GDPRHandler) RequestDeletion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req struct {
		Password string `json:"password"`
		Confirm  bool   `json:"confirm"` // explicit "I understand" checkbox
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !req.Confirm {
		res.Error(w, http.StatusBadRequest, "confirm_required",
			"explicit confirmation is required")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		res.Error(w, http.StatusBadRequest, "password_required", "password is required")
		return
	}

	result, err := h.svc.RequestDeletion(r.Context(), gdprapp.RequestDeletionInput{
		UserID:   userID,
		Password: req.Password,
	})
	if err != nil {
		var blocked *domaingdpr.OwnerBlockedError
		if errors.As(err, &blocked) {
			res.JSON(w, http.StatusConflict, map[string]any{
				"error": map[string]any{
					"code":    "owner_must_transfer_or_dissolve",
					"message": "You own an organization with active members. Transfer ownership or dissolve before deleting your account.",
					"details": map[string]any{
						"blocked_orgs": blocked.Orgs,
					},
				},
			})
			return
		}
		switch {
		case errors.Is(err, user.ErrInvalidCredentials):
			res.Error(w, http.StatusUnauthorized, "invalid_password", "password verification failed")
		case errors.Is(err, user.ErrUserNotFound):
			res.Error(w, http.StatusNotFound, "user_not_found", "user not found")
		default:
			slog.Error("gdpr request_deletion", "error", err.Error())
			res.Error(w, http.StatusInternalServerError, "internal_error", "could not process the request")
		}
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"email_sent_to": result.EmailSentTo,
		"expires_at":    result.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// ConfirmDeletion validates the JWT carried in the email link and
// sets users.deleted_at. The endpoint is intentionally a GET so a
// click in any email client works without JS.
//
// Status codes:
//
//	200  deleted_at successfully set; body carries the schedule
//	401  invalid / expired / wrong-purpose token
//	404  user from token claim no longer exists
func (h *GDPRHandler) ConfirmDeletion(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		res.Error(w, http.StatusBadRequest, "token_required", "token is required")
		return
	}
	result, err := h.svc.ConfirmDeletion(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrUnauthorized):
			res.Error(w, http.StatusUnauthorized, "invalid_token", "token is invalid or expired")
		case errors.Is(err, user.ErrUserNotFound):
			res.Error(w, http.StatusNotFound, "user_not_found", "user not found")
		default:
			slog.Error("gdpr confirm_deletion", "error", err.Error())
			res.Error(w, http.StatusInternalServerError, "internal_error", "could not confirm deletion")
		}
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"user_id":        result.UserID.String(),
		"deleted_at":     result.DeletedAt.UTC().Format(time.RFC3339),
		"hard_delete_at": result.HardDeleteAt.UTC().Format(time.RFC3339),
	})
}

// CancelDeletion clears users.deleted_at while the 30-day cooldown is
// still open. Auth-required because we want to be sure only the
// account owner can cancel — a leaked session cookie is the worst
// case but the standard auth check is enough.
//
// Status codes:
//
//	200  cancelled (or no-op when already not scheduled)
//	401  unauthenticated
func (h *GDPRHandler) CancelDeletion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	result, err := h.svc.CancelDeletion(r.Context(), userID)
	if err != nil {
		slog.Error("gdpr cancel_deletion", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "could not cancel deletion")
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"cancelled": !result.NoOp,
	})
}
