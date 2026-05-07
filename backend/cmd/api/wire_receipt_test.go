package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
)

// stubAuditRepo records every Log call so we can assert the receipt
// audit adapter wired into the right action+resource type.
type stubAuditRepo struct {
	calls []*audit.Entry
	err   error
}

func (s *stubAuditRepo) Log(_ context.Context, entry *audit.Entry) error {
	s.calls = append(s.calls, entry)
	return s.err
}

func (s *stubAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func (s *stubAuditRepo) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func TestReceiptAuditAdapter_LogReceiptView_WritesEntryWithCanonicalAction(t *testing.T) {
	repo := &stubAuditRepo{}
	a := &receiptAuditAdapter{audits: repo}

	uid := uuid.New()
	rid := uuid.New()
	a.LogReceiptView(context.Background(), uid, rid, "127.0.0.1")

	require.Len(t, repo.calls, 1)
	entry := repo.calls[0]
	assert.Equal(t, audit.ActionReceiptView, entry.Action)
	assert.Equal(t, audit.ResourceTypeReceipt, entry.ResourceType)
	require.NotNil(t, entry.UserID)
	assert.Equal(t, uid, *entry.UserID)
	require.NotNil(t, entry.ResourceID)
	assert.Equal(t, rid, *entry.ResourceID)
	require.NotNil(t, entry.IPAddress)
}

func TestReceiptAuditAdapter_LogReceiptPDFDownload_WritesPDFAction(t *testing.T) {
	repo := &stubAuditRepo{}
	a := &receiptAuditAdapter{audits: repo}

	a.LogReceiptPDFDownload(context.Background(), uuid.New(), uuid.New(), "")
	require.Len(t, repo.calls, 1)
	assert.Equal(t, audit.ActionReceiptPDFDownload, repo.calls[0].Action)
}

func TestReceiptAuditAdapter_NilUserAndReceiptID_StoresNullPointers(t *testing.T) {
	repo := &stubAuditRepo{}
	a := &receiptAuditAdapter{audits: repo}

	a.LogReceiptView(context.Background(), uuid.Nil, uuid.Nil, "")
	require.Len(t, repo.calls, 1)
	assert.Nil(t, repo.calls[0].UserID)
	assert.Nil(t, repo.calls[0].ResourceID)
}

func TestReceiptAuditAdapter_NilAdapter_NoOp(t *testing.T) {
	var a *receiptAuditAdapter
	// Calling on a nil receiver must not panic — the audit feature is
	// optional and the receipt feature must keep working without it.
	assert.NotPanics(t, func() {
		a.LogReceiptView(context.Background(), uuid.New(), uuid.New(), "")
		a.LogReceiptPDFDownload(context.Background(), uuid.New(), uuid.New(), "")
	})
}

func TestReceiptAuditAdapter_NilRepo_NoOp(t *testing.T) {
	a := &receiptAuditAdapter{audits: nil}
	assert.NotPanics(t, func() {
		a.LogReceiptView(context.Background(), uuid.New(), uuid.New(), "")
	})
}
