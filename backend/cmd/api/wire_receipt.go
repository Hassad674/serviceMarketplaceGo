package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/google/uuid"

	pdfadapter "marketplace-backend/internal/adapter/pdf"
	"marketplace-backend/internal/adapter/postgres"
	paymentapp "marketplace-backend/internal/app/payment"
	receiptapp "marketplace-backend/internal/app/receipt"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// receiptDeps captures the upstream resources the receipt feature
// reaches into. The feature is a thin presentation projection over
// payment_records + billing_profile + referral_attribution, so it
// needs the SQL pool (for the receipt repo + a tiny user-org reader)
// + the cross-feature read ports.
type receiptDeps struct {
	DB             *sql.DB
	PaymentSvc     *paymentapp.Service
	BillingProfile repository.BillingProfileRepository // optional — nil disables snapshot.party_for billing fields
	ReferralRepo   repository.ReferralRepository       // optional — nil disables snapshot.referrer
	UsersRepo      repository.UserRepository
	AuditRepo      repository.AuditRepository // optional — nil disables view/download audit events
}

// receiptWiring carries the products of the receipt feature
// initialisation. Nil-valued fields keep the rest of the backend
// booting on minimal builds (e.g. PDF renderer init failure → the
// list/detail endpoints stay live, the PDF endpoint returns 503).
type receiptWiring struct {
	Handler *handler.ReceiptHandler
}

// wireReceipt brings up the receipt feature. The handler is always
// returned (even when the snapshot resolver fails to wire) so the
// list / detail endpoints stay available — they read existing
// payment_records rows and degrade to "données indisponibles" for
// rows without a snapshot. The snapshot resolver is plugged into
// the payment service AS A SIDE-EFFECT of this wire so future
// CreatePaymentIntent calls populate the column.
func wireReceipt(deps receiptDeps) receiptWiring {
	repo := postgres.NewReceiptRepository(deps.DB)

	// PDF renderer — best-effort. If chromedp init fails (no
	// Chrome on the host), the rest of the feature still works
	// and the PDF endpoint returns 503.
	var renderer receiptapp.PDFRenderer
	if r, err := pdfadapter.New(); err == nil {
		renderer = r
	} else {
		slog.Warn("receipt feature: pdf renderer init failed; PDF endpoint disabled", "error", err)
	}

	svc := receiptapp.NewService(receiptapp.ServiceDeps{
		Repo:     repo,
		Renderer: renderer,
	})
	h := handler.NewReceiptHandler(svc)
	if deps.AuditRepo != nil {
		h = h.WithAuditLogger(&receiptAuditAdapter{audits: deps.AuditRepo})
	}

	// Snapshot resolver — wires into ChargeService.CreatePaymentIntent
	// so every new payment_record row carries a billing_snapshot.
	// All sub-deps are optional: a missing dependency simply yields
	// an empty party in the snapshot. The whole resolver is skipped
	// when payment is not configured.
	if deps.PaymentSvc != nil {
		resolver := receiptapp.NewSnapshotResolver(receiptapp.SnapshotResolverDeps{
			Users:     userOrgReaderAdapter(deps.UsersRepo),
			Billing:   deps.BillingProfile,
			Referrals: deps.ReferralRepo, // nil-tolerant inside the resolver
		})
		deps.PaymentSvc.SetReceiptSnapshotResolver(resolver)
		slog.Info("receipt feature: billing snapshot resolver wired into payment.CreatePaymentIntent")
	}

	return receiptWiring{Handler: h}
}

// receiptAuditAdapter satisfies handler.AuditLogger by writing to
// the canonical audit_logs table via the standard repository port.
// Failures are swallowed at WARN — audit logging must never block a
// user-facing read endpoint.
type receiptAuditAdapter struct {
	audits repository.AuditRepository
}

func (a *receiptAuditAdapter) LogReceiptView(ctx interface{}, userID, receiptID uuid.UUID, ip string) {
	a.log(ctx, audit.ActionReceiptView, userID, receiptID, ip)
}

func (a *receiptAuditAdapter) LogReceiptPDFDownload(ctx interface{}, userID, receiptID uuid.UUID, ip string) {
	a.log(ctx, audit.ActionReceiptPDFDownload, userID, receiptID, ip)
}

func (a *receiptAuditAdapter) log(ctxAny interface{}, action audit.Action, userID, receiptID uuid.UUID, ip string) {
	if a == nil || a.audits == nil {
		return
	}
	c, _ := ctxAny.(context.Context)
	if c == nil {
		c = context.Background()
	}
	uidPtr := &userID
	if userID == uuid.Nil {
		uidPtr = nil
	}
	ridPtr := &receiptID
	if receiptID == uuid.Nil {
		ridPtr = nil
	}
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       uidPtr,
		Action:       action,
		ResourceType: audit.ResourceTypeReceipt,
		ResourceID:   ridPtr,
		IPAddress:    ip,
	})
	if err != nil {
		slog.Warn("receipt audit: NewEntry failed", "action", action, "error", err)
		return
	}
	if err := a.audits.Log(c, entry); err != nil {
		slog.Warn("receipt audit: Log failed", "action", action, "error", err)
	}
}

// userOrgReaderAdapter wraps a UserRepository into the closure-shaped
// receiptapp.UserOrgFunc port. Returns uuid.Nil + nil when the user
// has no organization membership — the resolver treats that as an
// empty party (the snapshot field stays unset).
func userOrgReaderAdapter(users repository.UserRepository) receiptapp.UserOrgFunc {
	if users == nil {
		return func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
			return uuid.Nil, nil
		}
	}
	return func(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
		u, err := users.GetByID(ctx, userID)
		if err != nil || u == nil || u.OrganizationID == nil {
			return uuid.Nil, nil
		}
		return *u.OrganizationID, nil
	}
}
