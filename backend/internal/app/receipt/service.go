// Package receipt is the use-case layer of the "Reçus" feature.
//
// It is a thin orchestration over the read repository and the PDF
// renderer. The receipt is a presentation projection — the domain
// itself owns no behaviour beyond "is this org a party on me", so
// the service stays small.
//
// The feature is fully removable: deleting this directory + its
// wiring lines in cmd/api/main.go takes the routes off the router
// and the rest of the backend keeps compiling.
package receipt

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/receipt"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// PDFRenderer is the narrow port the receipt service uses to turn a
// hydrated Receipt into a PDF byte buffer. The chromedp renderer
// already wraps a chromedp instance — we depend on a tiny method
// instead of importing the full PDFRenderer interface so the service
// stays under the project's dependency limits.
type PDFRenderer interface {
	RenderReceipt(ctx context.Context, rec *domain.Receipt, language string) ([]byte, error)
}

// Service is the use-case orchestrator. Construct via NewService.
type Service struct {
	repo     repository.ReceiptRepository
	renderer PDFRenderer
}

// ServiceDeps groups the constructor arguments under the project's
// 4-arg limit.
type ServiceDeps struct {
	Repo     repository.ReceiptRepository
	Renderer PDFRenderer // optional — nil disables PDF endpoint
}

// NewService wires the receipt service. Repo is mandatory; Renderer
// is optional (the PDF endpoint short-circuits with
// ErrPDFRendererUnavailable when nil so the rest of the receipt
// surface stays available even on minimal builds).
func NewService(deps ServiceDeps) *Service {
	return &Service{
		repo:     deps.Repo,
		renderer: deps.Renderer,
	}
}

// ErrPDFRendererUnavailable is returned by RenderPDF when the
// renderer dependency is not wired.
var ErrPDFRendererUnavailable = errors.New("receipt pdf renderer not configured")

// ListPage is the cursor-based slice the handler returns. Receipts
// is non-nil even when empty so the JSON renders as `[]` rather
// than `null`.
type ListPage struct {
	Receipts   []*domain.Receipt
	NextCursor string
}

// List returns receipts for the caller's organization. The
// repository SQL filter pins the rows to those where the org is a
// party — we re-validate at the handler layer (defense in depth).
func (s *Service) List(ctx context.Context, orgID uuid.UUID, cursor string, limit int) (*ListPage, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("receipt service not configured")
	}
	if orgID == uuid.Nil {
		return nil, errors.New("organization id is required")
	}
	rows, next, err := s.repo.ListForOrganization(ctx, orgID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}
	if rows == nil {
		rows = []*domain.Receipt{}
	}
	return &ListPage{Receipts: rows, NextCursor: next}, nil
}

// Get returns one receipt by id. Surfaces:
//   - domain.ErrNotFound — the row does not exist.
//   - domain.ErrForbidden — exists but caller is not a party.
//
// Both errors keep their identity through fmt.Errorf wrapping in
// the call sites that need extra context, so the handler's
// errors.Is gates still work.
func (s *Service) Get(ctx context.Context, receiptID, orgID uuid.UUID) (*domain.Receipt, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("receipt service not configured")
	}
	if receiptID == uuid.Nil || orgID == uuid.Nil {
		return nil, errors.New("receipt id and org id are required")
	}
	rec, err := s.repo.GetForOrganization(ctx, receiptID, orgID)
	if err != nil {
		return nil, err
	}
	// Final domain-level ownership check. The repository SQL filter
	// is the primary defense; this is the safety net that catches
	// any future query path bug.
	if !rec.IsParty(orgID) {
		return nil, domain.ErrForbidden
	}
	return rec, nil
}

// RenderPDF produces the PDF bytes for one receipt. The caller is
// responsible for the HTTP transport (Content-Type, filename, etc.)
// — this method only orchestrates the read + render.
//
// language is "fr" or "en". Anything else falls back to "fr" inside
// the renderer (FR is our primary market).
func (s *Service) RenderPDF(ctx context.Context, receiptID, orgID uuid.UUID, language string) ([]byte, *domain.Receipt, error) {
	if s == nil || s.repo == nil {
		return nil, nil, errors.New("receipt service not configured")
	}
	if s.renderer == nil {
		return nil, nil, ErrPDFRendererUnavailable
	}
	rec, err := s.Get(ctx, receiptID, orgID)
	if err != nil {
		return nil, nil, err
	}
	pdf, err := s.renderer.RenderReceipt(ctx, rec, language)
	if err != nil {
		return nil, nil, fmt.Errorf("render receipt pdf: %w", err)
	}
	return pdf, rec, nil
}

// Re-exports so callers depending on this package only import
// app/receipt — the handler errors.Is checks stay readable.
var (
	ErrNotFound  = domain.ErrNotFound
	ErrForbidden = domain.ErrForbidden
)

// Compile-time assertion: Service satisfies the contract callers
// expect. Kept here as a sanity check in case the public method
// signatures drift accidentally.
var _ interface {
	List(ctx context.Context, orgID uuid.UUID, cursor string, limit int) (*ListPage, error)
	Get(ctx context.Context, receiptID, orgID uuid.UUID) (*domain.Receipt, error)
	RenderPDF(ctx context.Context, receiptID, orgID uuid.UUID, language string) ([]byte, *domain.Receipt, error)
} = (*Service)(nil)

// Re-export the port service package types receivers use, so the
// payment snapshot resolver wiring stays in sync. (This block is a
// stub — the actual snapshot-build flow lives in the snapshot.go
// adapter wired to the payment app at boot.)
var _ = portservice.ReceiptSnapshotInput{}
