package moderation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmoderation "marketplace-backend/internal/app/moderation"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- Mocks ---------------------------------------------------------------

type mockTextModeration struct {
	result *service.TextModerationResult
	err    error
	calls  int
}

func (m *mockTextModeration) AnalyzeText(_ context.Context, _ string) (*service.TextModerationResult, error) {
	m.calls++
	return m.result, m.err
}

type mockResultsRepo struct {
	upsertCalls  int
	upsertErr    error
	lastUpserted *moderation.Result
}

func (m *mockResultsRepo) Upsert(_ context.Context, r *moderation.Result) error {
	m.upsertCalls++
	m.lastUpserted = r
	return m.upsertErr
}
func (m *mockResultsRepo) GetByContent(_ context.Context, _ moderation.ContentType, _ uuid.UUID) (*moderation.Result, error) {
	return nil, moderation.ErrResultNotFound
}
func (m *mockResultsRepo) List(_ context.Context, _ repository.ModerationResultsFilters) ([]*moderation.Result, int, error) {
	return nil, 0, nil
}
func (m *mockResultsRepo) MarkReviewed(_ context.Context, _ moderation.ContentType, _ uuid.UUID, _ uuid.UUID, _ moderation.Status) error {
	return nil
}

type mockAuditRepo struct {
	logCalls    int
	logErr      error
	lastEntry   *audit.Entry
	lastAction  audit.Action
}

func (m *mockAuditRepo) Log(_ context.Context, e *audit.Entry) error {
	m.logCalls++
	m.lastEntry = e
	m.lastAction = e.Action
	return m.logErr
}
func (m *mockAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (m *mockAuditRepo) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

type mockAdminNotifier struct {
	incCalls       int
	lastCategory   string
}

func (m *mockAdminNotifier) IncrementAll(_ context.Context, category string) error {
	m.incCalls++
	m.lastCategory = category
	return nil
}
func (m *mockAdminNotifier) GetAll(_ context.Context, _ uuid.UUID) (map[string]int64, error) {
	return nil, nil
}
func (m *mockAdminNotifier) Reset(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// --- Helpers -------------------------------------------------------------

func newServiceWithMocks(t *testing.T) (
	*appmoderation.Service,
	*mockTextModeration,
	*mockResultsRepo,
	*mockAuditRepo,
	*mockAdminNotifier,
) {
	t.Helper()
	tm := &mockTextModeration{}
	repo := &mockResultsRepo{}
	audr := &mockAuditRepo{}
	notif := &mockAdminNotifier{}
	svc := appmoderation.NewService(appmoderation.Deps{
		TextModeration: tm,
		Results:        repo,
		Audit:          audr,
		AdminNotifier:  notif,
	})
	return svc, tm, repo, audr, notif
}

func makeAnalysis(score float64, labels ...service.TextModerationLabel) *service.TextModerationResult {
	return &service.TextModerationResult{
		MaxScore: score,
		IsSafe:   score < 0.5,
		Labels:   labels,
	}
}

// --- Tests ---------------------------------------------------------------

func TestService_Moderate_CleanText_NoSideEffects(t *testing.T) {
	// When the engine returns a low score, nothing should be persisted
	// to moderation_results, the audit log, or the admin notifier.
	svc, tm, repo, audr, notif := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.10)

	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "bonjour ça va ?",
	})
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusClean, res.Status)
	assert.Equal(t, 0, repo.upsertCalls, "clean text must not persist a row")
	assert.Equal(t, 0, audr.logCalls, "clean text must not audit")
	assert.Equal(t, 0, notif.incCalls, "clean text must not bump admin notifier")
}

func TestService_Moderate_FlaggedText_PersistsAuditNotifies(t *testing.T) {
	svc, tm, repo, audr, notif := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.65, service.TextModerationLabel{Name: "harassment", Score: 0.65})
	authorID := uuid.New()

	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType:  moderation.ContentTypeMessage,
		ContentID:    uuid.New(),
		AuthorUserID: &authorID,
		Text:         "tu es nul",
	})
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusFlagged, res.Status)

	require.Equal(t, 1, repo.upsertCalls)
	require.NotNil(t, repo.lastUpserted)
	assert.Equal(t, moderation.StatusFlagged, repo.lastUpserted.Status)
	assert.InDelta(t, 0.65, repo.lastUpserted.Score, 0.001)
	assert.NotNil(t, repo.lastUpserted.AuthorUserID)

	require.Equal(t, 1, audr.logCalls)
	assert.Contains(t, string(audr.lastAction), "auto_flag_message")

	assert.Equal(t, 1, notif.incCalls, "flagged content must bump the admin badge")
}

func TestService_Moderate_HiddenStatus_UsesHiddenCategory(t *testing.T) {
	// When status escalates to hidden, the admin notifier must use the
	// "messages_hidden" category so the existing badge classification
	// keeps working.
	svc, tm, _, _, notif := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.92, service.TextModerationLabel{Name: "harassment", Score: 0.92})

	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "très toxique",
	})
	require.NoError(t, err)
	assert.Equal(t, service.AdminNotifMessagesHidden, notif.lastCategory)
}

func TestService_Moderate_DeletedStatus_AuditsAutoDelete(t *testing.T) {
	svc, tm, repo, audr, _ := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.97, service.TextModerationLabel{Name: "harassment/threatening", Score: 0.85})

	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "je vais te tuer",
	})
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusDeleted, res.Status)
	assert.Equal(t, 1, repo.upsertCalls)
	require.Equal(t, 1, audr.logCalls)
	assert.Contains(t, string(audr.lastAction), "auto_delete_message")
}

func TestService_Moderate_BlockingMode_BelowThreshold_DoesNotBlock(t *testing.T) {
	// The blocking-mode caller should still get a flagged decision when
	// the score is high enough to flag but below their blocking bar.
	svc, tm, repo, _, _ := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.55)

	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType:       moderation.ContentTypeJobTitle,
		ContentID:         uuid.New(),
		Text:              "borderline title",
		BlockingMode:      true,
		BlockingThreshold: 0.85,
	})
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusFlagged, res.Status)
	assert.Equal(t, 1, repo.upsertCalls, "flagged still persists")
}

func TestService_Moderate_BlockingMode_AboveThreshold_ReturnsErrContentBlocked(t *testing.T) {
	svc, tm, repo, audr, notif := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.92, service.TextModerationLabel{Name: "harassment", Score: 0.92})

	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType:       moderation.ContentTypeUserDisplayName,
		ContentID:         uuid.New(),
		Text:              "fils de pute",
		BlockingMode:      true,
		BlockingThreshold: 0.50,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, moderation.ErrContentBlocked))
	assert.Equal(t, moderation.StatusBlocked, res.Status)

	assert.Equal(t, 1, repo.upsertCalls, "blocked attempts MUST be persisted for admin visibility")
	require.NotNil(t, repo.lastUpserted)
	assert.Equal(t, moderation.StatusBlocked, repo.lastUpserted.Status)
	assert.Equal(t, moderation.ReasonBlockedCreate, repo.lastUpserted.Reason)

	require.Equal(t, 1, audr.logCalls)
	assert.Contains(t, string(audr.lastAction), "block_create_user_display_name")

	assert.Equal(t, 0, notif.incCalls, "blocked attempts must not bump admin badge (no work for admin)")
}

func TestService_Moderate_BlockingMode_ZeroThreshold_NeverBlocks(t *testing.T) {
	svc, tm, _, _, _ := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.99)

	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType:       moderation.ContentTypeJobTitle,
		ContentID:         uuid.New(),
		Text:              "extreme",
		BlockingMode:      true,
		BlockingThreshold: 0,
	})
	require.NoError(t, err, "threshold 0 means no blocking")
	assert.NotEqual(t, moderation.StatusBlocked, res.Status)
}

func TestService_Moderate_AnalyzerError_PropagatesAndDoesNotPersist(t *testing.T) {
	svc, tm, repo, audr, notif := newServiceWithMocks(t)
	tm.err = errors.New("openai unavailable")

	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "anything",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openai unavailable")
	assert.Equal(t, 0, repo.upsertCalls)
	assert.Equal(t, 0, audr.logCalls)
	assert.Equal(t, 0, notif.incCalls)
}

func TestService_Moderate_EmptyText_ShortCircuitsClean(t *testing.T) {
	svc, tm, _, _, _ := newServiceWithMocks(t)
	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "",
	})
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusClean, res.Status)
	assert.Equal(t, 0, tm.calls, "empty text must skip the engine call")
}

func TestService_Moderate_RepoError_AsyncMode_ReturnsNilButLogs(t *testing.T) {
	// Async caller must not bubble up persistence failures — we already
	// did the analysis, the verdict still applies to the user-facing
	// flow, and an admin queue hiccup is a degraded-mode concern.
	svc, tm, repo, _, _ := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.65)
	repo.upsertErr = errors.New("db down")

	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "tu es nul",
	})
	assert.NoError(t, err, "async flagged decisions tolerate repo failures")
}

func TestService_Moderate_RepoError_BlockingMode_StillReturnsErrBlocked(t *testing.T) {
	// Even if the persistence side fails in blocking mode, the user-
	// facing answer is "we refused" — the user must NOT be allowed to
	// create a public profile with toxic content just because our
	// admin queue is down.
	svc, tm, repo, _, _ := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.92)
	repo.upsertErr = errors.New("db down")

	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType:       moderation.ContentTypeUserDisplayName,
		ContentID:         uuid.New(),
		Text:              "very toxic",
		BlockingMode:      true,
		BlockingThreshold: 0.50,
	})
	assert.True(t, errors.Is(err, moderation.ErrContentBlocked))
}

func TestService_Moderate_RejectsEmptyContentType(t *testing.T) {
	svc, _, _, _, _ := newServiceWithMocks(t)
	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: "",
		ContentID:   uuid.New(),
		Text:        "anything",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content_type required")
}

func TestService_Moderate_RejectsNilContentID(t *testing.T) {
	svc, _, _, _, _ := newServiceWithMocks(t)
	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.Nil,
		Text:        "anything",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content_id required")
}

func TestService_Moderate_NilOptionalSinks_DoesNotPanic(t *testing.T) {
	// Run without audit + adminNotifier wired (CI scenario).
	tm := &mockTextModeration{result: makeAnalysis(0.65)}
	repo := &mockResultsRepo{}
	svc := appmoderation.NewService(appmoderation.Deps{
		TextModeration: tm,
		Results:        repo,
	})
	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "tu es nul",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.upsertCalls, "main persistence must still happen without optional sinks")
}

func TestService_Moderate_NilResultsRepo_StillReturnsClean(t *testing.T) {
	// Even more degraded: no repo wired (extreme test scenario).
	tm := &mockTextModeration{result: makeAnalysis(0.10)}
	svc := appmoderation.NewService(appmoderation.Deps{TextModeration: tm})
	res, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeMessage,
		ContentID:   uuid.New(),
		Text:        "hello",
	})
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusClean, res.Status)
}

func TestService_Moderate_ReviewContentType_UsesReviewsCategory(t *testing.T) {
	svc, tm, _, _, notif := newServiceWithMocks(t)
	tm.result = makeAnalysis(0.65, service.TextModerationLabel{Name: "harassment", Score: 0.65})

	_, err := svc.Moderate(context.Background(), appmoderation.ModerateInput{
		ContentType: moderation.ContentTypeReview,
		ContentID:   uuid.New(),
		Text:        "bad review",
	})
	require.NoError(t, err)
	assert.Equal(t, service.AdminNotifReviewsFlagged, notif.lastCategory,
		"review content must keep its dedicated AdminNotifReviewsFlagged badge")
}
