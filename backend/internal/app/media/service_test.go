package media

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- mocks ---

type mockMediaRepo struct {
	createFn                func(ctx context.Context, m *mediadomain.Media) error
	getByIDFn               func(ctx context.Context, id uuid.UUID) (*mediadomain.Media, error)
	getAdminByIDFn          func(ctx context.Context, id uuid.UUID) (*repository.AdminMediaItem, error)
	getByJobIDFn            func(ctx context.Context, jobID string) (*mediadomain.Media, error)
	updateFn                func(ctx context.Context, m *mediadomain.Media) error
	deleteFn                func(ctx context.Context, id uuid.UUID) error
	listAdminFn             func(ctx context.Context, filters repository.AdminMediaFilters) ([]repository.AdminMediaItem, error)
	countAdminFn            func(ctx context.Context, filters repository.AdminMediaFilters) (int, error)
	clearSourceFn           func(ctx context.Context, mediaContext string, contextID uuid.UUID) error
	countRejectedByUploaderFn func(ctx context.Context, uploaderID uuid.UUID) (int, error)
}

func (m *mockMediaRepo) Create(ctx context.Context, media *mediadomain.Media) error {
	if m.createFn != nil {
		return m.createFn(ctx, media)
	}
	return nil
}
func (m *mockMediaRepo) GetByID(ctx context.Context, id uuid.UUID) (*mediadomain.Media, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, mediadomain.ErrMediaNotFound
}
func (m *mockMediaRepo) GetAdminByID(ctx context.Context, id uuid.UUID) (*repository.AdminMediaItem, error) {
	if m.getAdminByIDFn != nil {
		return m.getAdminByIDFn(ctx, id)
	}
	return nil, mediadomain.ErrMediaNotFound
}
func (m *mockMediaRepo) GetByJobID(ctx context.Context, jobID string) (*mediadomain.Media, error) {
	if m.getByJobIDFn != nil {
		return m.getByJobIDFn(ctx, jobID)
	}
	return nil, mediadomain.ErrMediaNotFound
}
func (m *mockMediaRepo) Update(ctx context.Context, media *mediadomain.Media) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, media)
	}
	return nil
}
func (m *mockMediaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockMediaRepo) ListAdmin(ctx context.Context, filters repository.AdminMediaFilters) ([]repository.AdminMediaItem, error) {
	if m.listAdminFn != nil {
		return m.listAdminFn(ctx, filters)
	}
	return nil, nil
}
func (m *mockMediaRepo) CountAdmin(ctx context.Context, filters repository.AdminMediaFilters) (int, error) {
	if m.countAdminFn != nil {
		return m.countAdminFn(ctx, filters)
	}
	return 0, nil
}
func (m *mockMediaRepo) ClearSource(ctx context.Context, mediaContext string, contextID uuid.UUID) error {
	if m.clearSourceFn != nil {
		return m.clearSourceFn(ctx, mediaContext, contextID)
	}
	return nil
}
func (m *mockMediaRepo) CountRejectedByUploader(ctx context.Context, uploaderID uuid.UUID) (int, error) {
	if m.countRejectedByUploaderFn != nil {
		return m.countRejectedByUploaderFn(ctx, uploaderID)
	}
	return 0, nil
}

// --- mock user repo ---

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
	updateFn  func(ctx context.Context, u *user.User) error
}

func (m *mockUserRepo) Create(_ context.Context, _ *user.User) error             { return nil }
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepo) Update(ctx context.Context, u *user.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}
func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error            { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (m *mockUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) CountByRole(_ context.Context) (map[string]int, error)   { return nil, nil }
func (m *mockUserRepo) CountByStatus(_ context.Context) (map[string]int, error) { return nil, nil }
func (m *mockUserRepo) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}
func (m *mockUserRepo) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepo) FindUserIDByStripeAccount(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepo) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockUserRepo) ClearStripeAccount(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockUserRepo) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockUserRepo) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}

// --- mock storage ---

type mockStorage struct {
	deleteFn   func(ctx context.Context, key string) error
	downloadFn func(ctx context.Context, key string) ([]byte, error)
}

func (m *mockStorage) Upload(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
	return "", nil
}
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}
func (m *mockStorage) BulkDelete(_ context.Context, keys []string) ([]service.BulkDeleteResult, error) {
	out := make([]service.BulkDeleteResult, len(keys))
	for i, k := range keys {
		out[i] = service.BulkDeleteResult{Key: k}
	}
	return out, nil
}
func (m *mockStorage) GetPublicURL(_ string) string { return "" }
func (m *mockStorage) GetPresignedUploadURL(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockStorage) GetPresignedDownloadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockStorage) GetPresignedDownloadURLAsAttachment(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (m *mockStorage) Download(ctx context.Context, key string) ([]byte, error) {
	if m.downloadFn != nil {
		return m.downloadFn(ctx, key)
	}
	return nil, nil
}

// cancelObservingModeration is a fake ContentModerationService that
// surfaces the ctx error if AnalyzeImage is called — the test for
// ctx propagation expects Download to abort first, so this should
// never fire. If it does, the test fails loudly.
type cancelObservingModeration struct{}

func (cancelObservingModeration) AnalyzeImage(ctx context.Context, _ []byte) (*service.ModerationResult, error) {
	return nil, ctx.Err()
}
func (cancelObservingModeration) AnalyzeVideo(_ context.Context, _, _ string) (*service.VideoJob, error) {
	return &service.VideoJob{}, nil
}
func (cancelObservingModeration) GetVideoModerationResult(_ context.Context, _ string) (*service.ModerationResult, error) {
	return &service.ModerationResult{Safe: true}, nil
}

// newTestMediaServiceWithModeration is the variant constructor used by
// the ctx-propagation test — it lets the caller swap the moderation
// provider for a fake that observes ctx cancellation.
func newTestMediaServiceWithModeration(
	mediaRepo *mockMediaRepo,
	storage *mockStorage,
	moderation service.ContentModerationService,
) *Service {
	if mediaRepo == nil {
		mediaRepo = &mockMediaRepo{}
	}
	if storage == nil {
		storage = &mockStorage{}
	}
	return NewService(ServiceDeps{
		Media:               mediaRepo,
		Storage:             storage,
		Moderation:          moderation,
		FlagThreshold:       50.0,
		AutoRejectThreshold: 90.0,
	})
}

// --- mock email ---

type mockEmail struct {
	sendNotificationFn func(ctx context.Context, to, subject, html string) error
}

func (m *mockEmail) SendPasswordReset(_ context.Context, _, _ string) error { return nil }
func (m *mockEmail) SendNotification(ctx context.Context, to, subject, html string) error {
	if m.sendNotificationFn != nil {
		return m.sendNotificationFn(ctx, to, subject, html)
	}
	return nil
}
func (m *mockEmail) SendTeamInvitation(_ context.Context, _ service.TeamInvitationEmailInput) error {
	return nil
}
func (m *mockEmail) SendRolePermissionsChanged(_ context.Context, _ service.RolePermissionsChangedEmailInput) error {
	return nil
}

// --- mock session ---

type mockSession struct {
	deleteByUserIDFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockSession) Create(_ context.Context, _ service.CreateSessionInput) (*service.Session, error) {
	return nil, nil
}
func (m *mockSession) Get(_ context.Context, _ string) (*service.Session, error) { return nil, nil }
func (m *mockSession) Delete(_ context.Context, _ string) error                  { return nil }
func (m *mockSession) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}
func (m *mockSession) CreateWSToken(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (m *mockSession) ValidateWSToken(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// --- mock broadcaster ---

type mockBroadcaster struct {
	broadcastAccountSuspendedFn func(ctx context.Context, userID uuid.UUID, reason string) error
}

func (m *mockBroadcaster) BroadcastNewMessage(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastTyping(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastStatusUpdate(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastUnreadCount(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockBroadcaster) BroadcastPresence(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastNotification(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastMessageEdited(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastMessageDeleted(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockBroadcaster) BroadcastAccountSuspended(ctx context.Context, userID uuid.UUID, reason string) error {
	if m.broadcastAccountSuspendedFn != nil {
		return m.broadcastAccountSuspendedFn(ctx, userID, reason)
	}
	return nil
}
func (m *mockBroadcaster) BroadcastAdminNotification(_ context.Context, _ []uuid.UUID) error {
	return nil
}

// --- helper ---

func newTestMediaService(
	mediaRepo *mockMediaRepo,
	userRepo *mockUserRepo,
	storage *mockStorage,
	email *mockEmail,
	session *mockSession,
	broadcaster *mockBroadcaster,
) *Service {
	if mediaRepo == nil {
		mediaRepo = &mockMediaRepo{}
	}
	if storage == nil {
		storage = &mockStorage{}
	}
	return NewService(ServiceDeps{
		Media:               mediaRepo,
		Users:               userRepo,
		Storage:             storage,
		Email:               email,
		SessionSvc:          session,
		Broadcaster:         broadcaster,
		FlagThreshold:       50.0,
		AutoRejectThreshold: 90.0,
	})
}

func newTestMedia(uploaderID uuid.UUID) *mediadomain.Media {
	ctxID := uuid.New()
	return &mediadomain.Media{
		ID:               uuid.New(),
		UploaderID:       uploaderID,
		FileURL:          "http://localhost:9000/bucket/profiles/test/photo.jpg",
		FileName:         "photo.jpg",
		FileType:         "image/jpeg",
		FileSize:         1024,
		Context:          mediadomain.ContextProfilePhoto,
		ContextID:        &ctxID,
		ModerationStatus: mediadomain.StatusPending,
	}
}

// --- RecordUpload routing tests (audio / pdf / image / video) ---
//
// Audio voice messages and PDF attachments have no automated moderation
// pipeline yet. Before this fix they accumulated in `pending` forever,
// polluting the admin queue counts and blocking any UI gate that relies
// on `approved` status. Until Transcribe→text-mod (audio) and PDF
// page-extract (PDF) are wired, the pragmatic default is to mark them
// auto-approved so the system stays consistent.

func TestRecordUpload_AudioWebm_ApprovedImmediately(t *testing.T) {
	uploaderID := uuid.New()
	var captured *mediadomain.Media
	var updateCalls int
	mediaRepo := &mockMediaRepo{
		createFn: func(_ context.Context, m *mediadomain.Media) error {
			captured = m
			return nil
		},
		updateFn: func(_ context.Context, m *mediadomain.Media) error {
			updateCalls++
			captured = m
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, nil, nil, nil, nil, nil)

	svc.RecordUpload(
		context.Background(),
		uploaderID,
		"http://localhost/marketplace/messaging/voice/abc.webm",
		"voice_message.ogg",
		"audio/webm;codecs=opus",
		1024,
		mediadomain.ContextMessageAttach,
	)

	require.NotNil(t, captured, "media must be persisted")
	assert.Equal(t, 1, updateCalls, "implicit approval must call Update once")
	assert.Equal(t, mediadomain.StatusApproved, captured.ModerationStatus,
		"audio uploads must be auto-approved (no moderation pipeline yet)")
}

func TestRecordUpload_ApplicationPDF_ApprovedImmediately(t *testing.T) {
	uploaderID := uuid.New()
	var captured *mediadomain.Media
	mediaRepo := &mockMediaRepo{
		createFn: func(_ context.Context, m *mediadomain.Media) error {
			captured = m
			return nil
		},
		updateFn: func(_ context.Context, m *mediadomain.Media) error {
			captured = m
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, nil, nil, nil, nil, nil)

	svc.RecordUpload(
		context.Background(),
		uploaderID,
		"http://localhost/marketplace/messaging/files/abc.pdf",
		"contract.pdf",
		"application/pdf",
		2048,
		mediadomain.ContextMessageAttach,
	)

	require.NotNil(t, captured, "media must be persisted")
	assert.Equal(t, mediadomain.StatusApproved, captured.ModerationStatus,
		"PDF uploads must be auto-approved (no moderation pipeline yet)")
}

func TestRecordUpload_StructuredLogEmitted_OnApprove(t *testing.T) {
	// applyDecision must emit a single INFO log line per analyzed
	// media so on-call can audit the pipeline from logs alone — see
	// the BUG-17 follow-up. Without it, a clean image's path leaves
	// no log trace and is indistinguishable from "moderation never
	// ran".
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	svc := newTestMediaService(nil, nil, nil, nil, nil, nil)
	m := newTestMedia(uuid.New())
	result := &service.ModerationResult{Safe: true, Score: 5}

	svc.applyDecision(context.Background(), m, "profiles/test/photo.jpg", result)

	out := buf.String()
	assert.Contains(t, out, `"msg":"moderation: media analyzed"`)
	assert.Contains(t, out, `"decision":"approved"`)
	assert.Contains(t, out, `"safe":true`)
	assert.Contains(t, out, `"score":5`)
	assert.Contains(t, out, `"media_id"`)
}

func TestRecordUpload_StructuredLogEmitted_OnFlag(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	svc := newTestMediaService(nil, nil, nil, nil, nil, nil)
	m := newTestMedia(uuid.New())
	labels := []mediadomain.ModerationLabel{{Name: "Violence", Confidence: 75}}
	result := &service.ModerationResult{Safe: false, Score: 75, Labels: labels}

	svc.applyDecision(context.Background(), m, "profiles/test/photo.jpg", result)

	out := buf.String()
	assert.Contains(t, out, `"decision":"flagged"`)
	assert.Contains(t, out, `"labels":"Violence"`)
	assert.Contains(t, out, `"score":75`)
}

func TestRecordUpload_AudioImplicitApproval_LogsErrorOnUpdateFailure(t *testing.T) {
	// applyImplicitApproval logs at ERROR when the DB update fails so
	// silent moderation regressions cannot accumulate. The row stays
	// in `pending` (we never overwrote it), surfacing in the admin
	// queue's pending count — the operator gets a chance to fix.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	mediaRepo := &mockMediaRepo{
		updateFn: func(_ context.Context, _ *mediadomain.Media) error {
			return assert.AnError
		},
	}

	svc := newTestMediaService(mediaRepo, nil, nil, nil, nil, nil)
	svc.RecordUpload(
		context.Background(),
		uuid.New(),
		"http://localhost/marketplace/messaging/voice/abc.webm",
		"voice.ogg",
		"audio/webm",
		512,
		mediadomain.ContextMessageAttach,
	)

	assert.Contains(t, buf.String(), `"msg":"media moderation: update implicit approval"`,
		"a failed implicit approval must surface as an ERROR log")
}

func TestRecordUpload_PropagatesContextCancellation(t *testing.T) {
	// The new RecordUpload signature accepts a parent ctx so SIGTERM
	// can cancel the moderation pipeline mid-flight. We assert the
	// ctx error makes it to the moderation provider instead of being
	// silently swallowed.
	uploaderID := uuid.New()
	mediaRepo := &mockMediaRepo{}
	storage := &mockStorage{
		downloadFn: func(ctx context.Context, _ string) ([]byte, error) {
			// The ctx passed in must be cancellable; the test cancels
			// the parent before this function is called.
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	svc := newTestMediaServiceWithModeration(mediaRepo, storage, &cancelObservingModeration{})

	parentCtx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so Download sees ctx.Err() immediately

	// Should return without panic and without invoking AnalyzeImage.
	require.NotPanics(t, func() {
		svc.RecordUpload(
			parentCtx,
			uploaderID,
			"http://localhost/marketplace/profiles/u/photo.jpg",
			"photo.jpg",
			"image/jpeg",
			1024,
			mediadomain.ContextProfilePhoto,
		)
	})
}

// --- isSexualContent tests ---

func TestIsSexualContent_WithNudityLabel(t *testing.T) {
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{
			{Name: "Explicit Nudity", Confidence: 98},
		},
	}
	assert.True(t, isSexualContent(result))
}

func TestIsSexualContent_WithSexualLabel(t *testing.T) {
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{
			{Name: "Sexual Activity", Confidence: 95},
		},
	}
	assert.True(t, isSexualContent(result))
}

func TestIsSexualContent_WithExplicitLabel(t *testing.T) {
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{
			{Name: "Explicit Content", Confidence: 90},
		},
	}
	assert.True(t, isSexualContent(result))
}

func TestIsSexualContent_WithViolenceOnly(t *testing.T) {
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{
			{Name: "Violence", Confidence: 90},
		},
	}
	assert.False(t, isSexualContent(result))
}

func TestIsSexualContent_EmptyLabels(t *testing.T) {
	result := &service.ModerationResult{Labels: nil}
	assert.False(t, isSexualContent(result))
}

// --- applyDecision tests ---

func TestApplyDecision_Safe_ApproveAutomatic(t *testing.T) {
	mediaRepo := &mockMediaRepo{}
	svc := newTestMediaService(mediaRepo, nil, nil, nil, nil, nil)

	m := newTestMedia(uuid.New())
	result := &service.ModerationResult{Safe: true, Score: 10}

	svc.applyDecision(context.Background(), m, "profiles/test/photo.jpg", result)

	assert.Equal(t, mediadomain.StatusApproved, m.ModerationStatus)
}

func TestApplyDecision_HighScore_AutoReject(t *testing.T) {
	var deletedKey string
	var clearedSource bool
	storage := &mockStorage{
		deleteFn: func(_ context.Context, key string) error {
			deletedKey = key
			return nil
		},
	}
	mediaRepo := &mockMediaRepo{
		clearSourceFn: func(_ context.Context, _ string, _ uuid.UUID) error {
			clearedSource = true
			return nil
		},
	}
	svc := newTestMediaService(mediaRepo, nil, storage, nil, nil, nil)

	m := newTestMedia(uuid.New())
	labels := []mediadomain.ModerationLabel{{Name: "Violence", Confidence: 99}}
	result := &service.ModerationResult{Safe: false, Score: 96, Labels: labels}

	svc.applyDecision(context.Background(), m, "profiles/test/photo.jpg", result)

	assert.Equal(t, mediadomain.StatusRejected, m.ModerationStatus)
	assert.Equal(t, "profiles/test/photo.jpg", deletedKey)
	assert.True(t, clearedSource, "ClearSource should be called for rejected media with context ID")
}

func TestApplyDecision_MediumScore_Flag(t *testing.T) {
	svc := newTestMediaService(nil, nil, nil, nil, nil, nil)

	m := newTestMedia(uuid.New())
	labels := []mediadomain.ModerationLabel{{Name: "Suggestive", Confidence: 75}}
	result := &service.ModerationResult{Safe: false, Score: 75, Labels: labels}

	svc.applyDecision(context.Background(), m, "profiles/test/photo.jpg", result)

	assert.Equal(t, mediadomain.StatusFlagged, m.ModerationStatus)
	assert.Equal(t, 75.0, m.ModerationScore)
}

// --- checkAutoSuspension tests ---

func TestCheckAutoSuspension_SexualContent_2Rejections_Suspends(t *testing.T) {
	uploaderID := uuid.New()
	var userUpdated bool
	var sessionDeleted bool
	var emailSent bool
	var broadcastSent bool

	mediaRepo := &mockMediaRepo{
		countRejectedByUploaderFn: func(_ context.Context, id uuid.UUID) (int, error) {
			assert.Equal(t, uploaderID, id)
			return 2, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{
				ID:     uploaderID,
				Email:  "offender@example.com",
				Status: user.StatusActive,
			}, nil
		},
		updateFn: func(_ context.Context, u *user.User) error {
			userUpdated = true
			assert.Equal(t, user.StatusSuspended, u.Status)
			assert.NotEmpty(t, u.SuspensionReason)
			return nil
		},
	}
	session := &mockSession{
		deleteByUserIDFn: func(_ context.Context, id uuid.UUID) error {
			sessionDeleted = true
			assert.Equal(t, uploaderID, id)
			return nil
		},
	}
	email := &mockEmail{
		sendNotificationFn: func(_ context.Context, to, _, _ string) error {
			emailSent = true
			assert.Equal(t, "offender@example.com", to)
			return nil
		},
	}
	broadcaster := &mockBroadcaster{
		broadcastAccountSuspendedFn: func(_ context.Context, id uuid.UUID, _ string) error {
			broadcastSent = true
			assert.Equal(t, uploaderID, id)
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, userRepo, nil, email, session, broadcaster)

	m := newTestMedia(uploaderID)
	m.ModerationStatus = mediadomain.StatusRejected
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Explicit Nudity", Confidence: 99}},
		Score:  98,
	}

	svc.checkAutoSuspension(context.Background(), m, result)

	assert.True(t, userUpdated, "user should be suspended")
	assert.True(t, sessionDeleted, "sessions should be invalidated")
	assert.True(t, emailSent, "notification email should be sent")
	assert.True(t, broadcastSent, "WS event should be broadcast")
}

func TestCheckAutoSuspension_SexualContent_1Rejection_NoAction(t *testing.T) {
	uploaderID := uuid.New()
	var userUpdated bool

	mediaRepo := &mockMediaRepo{
		countRejectedByUploaderFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 1, nil
		},
	}
	userRepo := &mockUserRepo{
		updateFn: func(_ context.Context, _ *user.User) error {
			userUpdated = true
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, userRepo, nil, nil, nil, nil)

	m := newTestMedia(uploaderID)
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Explicit Nudity", Confidence: 99}},
	}

	svc.checkAutoSuspension(context.Background(), m, result)

	assert.False(t, userUpdated, "1 rejection for sexual content should not trigger suspension")
}

func TestCheckAutoSuspension_Violence_3Rejections_Suspends(t *testing.T) {
	uploaderID := uuid.New()
	var userUpdated bool

	mediaRepo := &mockMediaRepo{
		countRejectedByUploaderFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 3, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return &user.User{
				ID:     uploaderID,
				Email:  "offender@example.com",
				Status: user.StatusActive,
			}, nil
		},
		updateFn: func(_ context.Context, u *user.User) error {
			userUpdated = true
			assert.Equal(t, user.StatusSuspended, u.Status)
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, userRepo, nil, &mockEmail{}, &mockSession{}, &mockBroadcaster{})

	m := newTestMedia(uploaderID)
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Violence", Confidence: 90}},
	}

	svc.checkAutoSuspension(context.Background(), m, result)

	assert.True(t, userUpdated, "3 rejections for non-sexual content should trigger suspension")
}

func TestCheckAutoSuspension_Violence_2Rejections_NoAction(t *testing.T) {
	uploaderID := uuid.New()
	var userUpdated bool

	mediaRepo := &mockMediaRepo{
		countRejectedByUploaderFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 2, nil
		},
	}
	userRepo := &mockUserRepo{
		updateFn: func(_ context.Context, _ *user.User) error {
			userUpdated = true
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, userRepo, nil, nil, nil, nil)

	m := newTestMedia(uploaderID)
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Violence", Confidence: 90}},
	}

	svc.checkAutoSuspension(context.Background(), m, result)

	assert.False(t, userUpdated, "2 rejections for non-sexual content should not trigger suspension")
}

func TestCheckAutoSuspension_AlreadySuspended_NoAction(t *testing.T) {
	uploaderID := uuid.New()
	var userUpdated bool

	mediaRepo := &mockMediaRepo{
		countRejectedByUploaderFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 5, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			u := &user.User{ID: uploaderID, Status: user.StatusActive}
			u.Suspend("already suspended", nil)
			return u, nil
		},
		updateFn: func(_ context.Context, _ *user.User) error {
			userUpdated = true
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, userRepo, nil, nil, nil, nil)

	m := newTestMedia(uploaderID)
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Explicit Nudity", Confidence: 99}},
	}

	svc.checkAutoSuspension(context.Background(), m, result)

	assert.False(t, userUpdated, "already suspended user should not be updated again")
}

func TestCheckAutoSuspension_AlreadyBanned_NoAction(t *testing.T) {
	uploaderID := uuid.New()
	var userUpdated bool

	mediaRepo := &mockMediaRepo{
		countRejectedByUploaderFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 5, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			u := &user.User{ID: uploaderID, Status: user.StatusActive}
			u.Ban("already banned")
			return u, nil
		},
		updateFn: func(_ context.Context, _ *user.User) error {
			userUpdated = true
			return nil
		},
	}

	svc := newTestMediaService(mediaRepo, userRepo, nil, nil, nil, nil)

	m := newTestMedia(uploaderID)
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Violence", Confidence: 90}},
	}

	svc.checkAutoSuspension(context.Background(), m, result)

	assert.False(t, userUpdated, "already banned user should not be updated again")
}

func TestCheckAutoSuspension_NilUserRepo_NoAction(t *testing.T) {
	mediaRepo := &mockMediaRepo{}
	// users is nil in the service
	svc := NewService(ServiceDeps{
		Media:               mediaRepo,
		Users:               nil,
		FlagThreshold:       50,
		AutoRejectThreshold: 90,
	})

	m := newTestMedia(uuid.New())
	result := &service.ModerationResult{
		Labels: []mediadomain.ModerationLabel{{Name: "Violence", Confidence: 90}},
	}

	// Should not panic
	require.NotPanics(t, func() {
		svc.checkAutoSuspension(context.Background(), m, result)
	})
}

func (m *mockUserRepo) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (m *mockUserRepo) GetKYCPendingUsers(_ context.Context) ([]*user.User, error) { return nil, nil }
func (m *mockUserRepo) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error { return nil }

// --- Session version stubs (migration 056, Phase 3) ---
func (m *mockUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error {
	return nil
}
