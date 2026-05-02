package searchindex_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/searchindex"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/search"
)

// fakePendingEvents records every Schedule(Tx) call so tests can
// assert on the emitted events. Implements only the subset of
// repository.PendingEventRepository that the publisher uses.
type fakePendingEvents struct {
	scheduled    []*pendingevent.PendingEvent
	scheduledTx  []*pendingevent.PendingEvent
	scheduleErr  error
	scheduleTxErr error
}

func (f *fakePendingEvents) Schedule(_ context.Context, e *pendingevent.PendingEvent) error {
	if f.scheduleErr != nil {
		return f.scheduleErr
	}
	f.scheduled = append(f.scheduled, e)
	return nil
}

func (f *fakePendingEvents) ScheduleTx(_ context.Context, _ *sql.Tx, e *pendingevent.PendingEvent) error {
	if f.scheduleTxErr != nil {
		return f.scheduleTxErr
	}
	f.scheduledTx = append(f.scheduledTx, e)
	return nil
}

// ScheduleStripe is required by the PendingEventRepository interface
// after P8. The publisher does not exercise the Stripe webhook path,
// so this stub returns success without recording the event.
func (f *fakePendingEvents) ScheduleStripe(_ context.Context, _ *pendingevent.PendingEvent) (bool, error) {
	return true, nil
}

func (f *fakePendingEvents) PopDue(_ context.Context, _ int) ([]*pendingevent.PendingEvent, error) {
	return nil, nil
}
func (f *fakePendingEvents) MarkDone(_ context.Context, _ *pendingevent.PendingEvent) error {
	return nil
}
func (f *fakePendingEvents) MarkFailed(_ context.Context, _ *pendingevent.PendingEvent) error {
	return nil
}
func (f *fakePendingEvents) GetByID(_ context.Context, _ uuid.UUID) (*pendingevent.PendingEvent, error) {
	return nil, nil
}

func TestNewPublisher_RequiresRepo(t *testing.T) {
	_, err := searchindex.NewPublisher(searchindex.PublisherConfig{})
	assert.ErrorContains(t, err, "pending events repository")
}

func TestPublisher_Nil_IsSafe(t *testing.T) {
	// A nil publisher must return nil from both methods so the
	// feature services can pass the publisher conditionally.
	var p *searchindex.Publisher
	assert.NoError(t, p.PublishReindex(context.Background(), uuid.New(), search.PersonaFreelance))
	assert.NoError(t, p.PublishDelete(context.Background(), uuid.New()))
}

func TestPublisher_PublishReindex_WritesEvent(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	orgID := uuid.New()
	require.NoError(t, pub.PublishReindex(context.Background(), orgID, search.PersonaFreelance))
	require.Len(t, events.scheduled, 1)

	ev := events.scheduled[0]
	assert.Equal(t, pendingevent.TypeSearchReindex, ev.EventType)

	var payload searchindex.ReindexPayload
	require.NoError(t, json.Unmarshal(ev.Payload, &payload))
	assert.Equal(t, orgID, payload.OrganizationID)
	assert.Equal(t, search.PersonaFreelance, payload.Persona)
}

func TestPublisher_PublishReindex_DebouncedWithinCooldown(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{
		Events:   events,
		Cooldown: time.Hour,
	})
	require.NoError(t, err)

	orgID := uuid.New()
	require.NoError(t, pub.PublishReindex(context.Background(), orgID, search.PersonaFreelance))
	require.NoError(t, pub.PublishReindex(context.Background(), orgID, search.PersonaFreelance))
	require.NoError(t, pub.PublishReindex(context.Background(), orgID, search.PersonaFreelance))

	assert.Len(t, events.scheduled, 1, "three rapid publishes must dedupe to one")
}

func TestPublisher_PublishReindex_DifferentOrgsNotDeduped(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{
		Events:   events,
		Cooldown: time.Hour,
	})
	require.NoError(t, err)

	orgA := uuid.New()
	orgB := uuid.New()
	require.NoError(t, pub.PublishReindex(context.Background(), orgA, search.PersonaFreelance))
	require.NoError(t, pub.PublishReindex(context.Background(), orgB, search.PersonaFreelance))
	assert.Len(t, events.scheduled, 2)
}

func TestPublisher_PublishReindex_RejectsInvalidInputs(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	assert.ErrorContains(t, pub.PublishReindex(context.Background(), uuid.Nil, search.PersonaFreelance), "orgID is required")
	assert.ErrorContains(t, pub.PublishReindex(context.Background(), uuid.New(), "enterprise"), "invalid persona")
	assert.Len(t, events.scheduled, 0)
}

func TestPublisher_PublishDelete_NotDebounced(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{
		Events:   events,
		Cooldown: time.Hour,
	})
	require.NoError(t, err)

	orgID := uuid.New()
	require.NoError(t, pub.PublishDelete(context.Background(), orgID))
	require.NoError(t, pub.PublishDelete(context.Background(), orgID))
	assert.Len(t, events.scheduled, 2, "deletes must never debounce")

	for _, ev := range events.scheduled {
		assert.Equal(t, pendingevent.TypeSearchDelete, ev.EventType)
	}
}

func TestPublisher_PublishDelete_RejectsNilOrg(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	assert.ErrorContains(t, pub.PublishDelete(context.Background(), uuid.Nil), "orgID is required")
}

func TestPublisher_RepoErrorPropagates(t *testing.T) {
	events := &fakePendingEvents{scheduleErr: errors.New("db down")}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	err = pub.PublishReindex(context.Background(), uuid.New(), search.PersonaFreelance)
	assert.ErrorContains(t, err, "db down")
}

// ---------------------------------------------------------------------------
// BUG-05 — outbox path: PublishReindexTx / PublishDeleteTx must
// participate in the caller's transaction.
// ---------------------------------------------------------------------------

func TestPublisher_PublishReindexTx_WritesEventViaTxPath(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	orgID := uuid.New()
	// A nil-but-non-nil tx pointer is not produced here; we use a real
	// *sql.Tx zero value via &sql.Tx{} so the publisher accepts it as
	// a non-nil transaction. The fake repo never dereferences the tx.
	tx := &sql.Tx{}
	require.NoError(t, pub.PublishReindexTx(context.Background(), tx, orgID, search.PersonaFreelance))

	assert.Empty(t, events.scheduled, "tx path must NOT use the pool-bound Schedule")
	require.Len(t, events.scheduledTx, 1, "tx path must call ScheduleTx exactly once")

	ev := events.scheduledTx[0]
	assert.Equal(t, pendingevent.TypeSearchReindex, ev.EventType)

	var payload searchindex.ReindexPayload
	require.NoError(t, json.Unmarshal(ev.Payload, &payload))
	assert.Equal(t, orgID, payload.OrganizationID)
	assert.Equal(t, search.PersonaFreelance, payload.Persona)
}

func TestPublisher_PublishReindexTx_RejectsNilTx(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	err = pub.PublishReindexTx(context.Background(), nil, uuid.New(), search.PersonaFreelance)
	assert.ErrorContains(t, err, "tx is required")
	assert.Empty(t, events.scheduledTx)
}

func TestPublisher_PublishReindexTx_RepoErrorPropagates(t *testing.T) {
	events := &fakePendingEvents{scheduleTxErr: errors.New("tx exec failed")}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	err = pub.PublishReindexTx(context.Background(), &sql.Tx{}, uuid.New(), search.PersonaFreelance)
	assert.ErrorContains(t, err, "tx exec failed")
}

func TestPublisher_PublishReindexTx_DebouncedSameAsHorsTx(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{
		Events:   events,
		Cooldown: time.Hour,
	})
	require.NoError(t, err)

	orgID := uuid.New()
	tx := &sql.Tx{}
	require.NoError(t, pub.PublishReindexTx(context.Background(), tx, orgID, search.PersonaFreelance))
	require.NoError(t, pub.PublishReindexTx(context.Background(), tx, orgID, search.PersonaFreelance))
	require.NoError(t, pub.PublishReindexTx(context.Background(), tx, orgID, search.PersonaFreelance))

	assert.Len(t, events.scheduledTx, 1, "same {org, persona} within cooldown must dedupe to one tx insert")
}

func TestPublisher_PublishDeleteTx_WritesEvent(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	orgID := uuid.New()
	require.NoError(t, pub.PublishDeleteTx(context.Background(), &sql.Tx{}, orgID))

	require.Len(t, events.scheduledTx, 1)
	assert.Equal(t, pendingevent.TypeSearchDelete, events.scheduledTx[0].EventType)
}

func TestPublisher_PublishDeleteTx_RejectsNilTx(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	err = pub.PublishDeleteTx(context.Background(), nil, uuid.New())
	assert.ErrorContains(t, err, "tx is required")
}

func TestPublisher_NilPublisher_TxMethodsAreSafe(t *testing.T) {
	var p *searchindex.Publisher
	assert.NoError(t, p.PublishReindexTx(context.Background(), &sql.Tx{}, uuid.New(), search.PersonaFreelance))
	assert.NoError(t, p.PublishDeleteTx(context.Background(), &sql.Tx{}, uuid.New()))
}

// PublishReindexAllPersonas fans out to all three personas
// (Freelance, Agency, Referrer). One Schedule call per persona.
func TestPublisher_PublishReindexAllPersonas_FansOutToThree(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	orgID := uuid.New()
	require.NoError(t, pub.PublishReindexAllPersonas(context.Background(), orgID))
	require.Len(t, events.scheduled, 3,
		"AllPersonas must schedule exactly 3 events (Freelance + Agency + Referrer)")

	personas := map[search.Persona]int{}
	for _, ev := range events.scheduled {
		var payload searchindex.ReindexPayload
		require.NoError(t, json.Unmarshal(ev.Payload, &payload))
		assert.Equal(t, orgID, payload.OrganizationID)
		personas[payload.Persona]++
	}
	assert.Equal(t, 1, personas[search.PersonaFreelance])
	assert.Equal(t, 1, personas[search.PersonaAgency])
	assert.Equal(t, 1, personas[search.PersonaReferrer])
}

// AllPersonas must surface the underlying error wrapped with the
// "publish reindex all personas" prefix so the caller can correlate.
func TestPublisher_PublishReindexAllPersonas_PropagatesError(t *testing.T) {
	events := &fakePendingEvents{scheduleErr: errors.New("schedule failed")}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	err = pub.PublishReindexAllPersonas(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorContains(t, err, "publish reindex all personas")
	assert.ErrorContains(t, err, "schedule failed",
		"the wrapped underlying error message must propagate so logs are actionable")
}

// AllPersonas on a nil publisher is safe — same contract as the
// underlying PublishReindex.
func TestPublisher_PublishReindexAllPersonas_NilPublisher_IsSafe(t *testing.T) {
	var p *searchindex.Publisher
	assert.NoError(t, p.PublishReindexAllPersonas(context.Background(), uuid.New()))
}

// AllPersonas with an invalid (zero) orgID must surface the error
// without scheduling anything.
func TestPublisher_PublishReindexAllPersonas_RejectsNilOrgID(t *testing.T) {
	events := &fakePendingEvents{}
	pub, err := searchindex.NewPublisher(searchindex.PublisherConfig{Events: events})
	require.NoError(t, err)

	err = pub.PublishReindexAllPersonas(context.Background(), uuid.Nil)
	require.Error(t, err)
	assert.Empty(t, events.scheduled,
		"a rejected orgID must NOT leak any events to the outbox")
}
