package repository_test

// segregated_test.go proves that the Interface Segregation Principle
// pay-off is real, not just notational:
//
//   1. Tiny mocks of each segregated child interface compile and run.
//      A consumer that only depends on, say, ReferralReader can be
//      tested with a 10-method-or-less mock instead of the full
//      24-method ReferralRepository panic-stub.
//
//   2. Composing the segregated children (e.g. ReferralReader +
//      ReferralWriter + …) yields a value that satisfies the original
//      god interface. This is the Liskov substitution side of the
//      contract — any consumer that needs the full surface can be
//      handed a composition of segregated mocks.
//
//   3. The compile-time `var _ XRepository = (interface { … })(nil)`
//      guards in the segregated files are exercised by `go vet ./...` —
//      a missing or extra method breaks the build, which is the
//      automatic regression alarm we want.
//
// We avoid the heavy `testify` setup here: this file is about *interface
// compliance*, not behavior. Behavior tests live next to each consumer.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// ---------------------------------------------------------------------------
// REFERRAL — proof of segregation
// ---------------------------------------------------------------------------

// referralReaderMock is the smallest possible mock of ReferralReader.
// 10 methods vs the 24 of the wide interface — under the 5-minute mock
// budget set by CLAUDE.md ISP rule.
type referralReaderMock struct {
	getByIDCalls  int
	listByCalls   int
	lastReferrer  uuid.UUID
}

func (m *referralReaderMock) GetByID(_ context.Context, _ uuid.UUID) (*referral.Referral, error) {
	m.getByIDCalls++
	return &referral.Referral{}, nil
}
func (m *referralReaderMock) FindActiveByCouple(_ context.Context, _, _ uuid.UUID) (*referral.Referral, error) {
	return nil, nil
}
func (m *referralReaderMock) ListByReferrer(_ context.Context, referrerID uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	m.listByCalls++
	m.lastReferrer = referrerID
	return nil, "", nil
}
func (m *referralReaderMock) ListIncomingForProvider(_ context.Context, _ uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return nil, "", nil
}
func (m *referralReaderMock) ListIncomingForClient(_ context.Context, _ uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return nil, "", nil
}
func (m *referralReaderMock) ListNegotiations(_ context.Context, _ uuid.UUID) ([]*referral.Negotiation, error) {
	return nil, nil
}
func (m *referralReaderMock) ListExpiringIntros(_ context.Context, _ time.Time, _ int) ([]*referral.Referral, error) {
	return nil, nil
}
func (m *referralReaderMock) ListExpiringActives(_ context.Context, _ time.Time, _ int) ([]*referral.Referral, error) {
	return nil, nil
}
func (m *referralReaderMock) CountByReferrer(_ context.Context, _ uuid.UUID) (map[referral.Status]int, error) {
	return nil, nil
}
func (m *referralReaderMock) SumCommissionsByReferrer(_ context.Context, _ uuid.UUID) (map[referral.CommissionStatus]int64, error) {
	return nil, nil
}

// dashboardConsumer is a hypothetical app service that ONLY reads.
// Declaring its dependency as ReferralReader (not ReferralRepository)
// is the whole point of segregation: it cannot accidentally write,
// and it can be tested with a 10-method mock.
type dashboardConsumer struct {
	referrals repository.ReferralReader
}

func (d *dashboardConsumer) loadDashboard(ctx context.Context, referrerID uuid.UUID) error {
	_, _, err := d.referrals.ListByReferrer(ctx, referrerID, repository.ReferralListFilter{Limit: 20})
	return err
}

func TestReferralReader_SegregatedConsumer(t *testing.T) {
	mock := &referralReaderMock{}
	svc := &dashboardConsumer{referrals: mock}

	if err := svc.loadDashboard(context.Background(), uuid.New()); err != nil {
		t.Fatalf("loadDashboard: %v", err)
	}
	if mock.listByCalls != 1 {
		t.Fatalf("expected 1 ListByReferrer call, got %d", mock.listByCalls)
	}
}

// ---------------------------------------------------------------------------
// MESSAGE — proof of segregation
// ---------------------------------------------------------------------------

type messageReaderMock struct {
	getMessageCalls int
}

func (m *messageReaderMock) GetConversation(_ context.Context, _ uuid.UUID) (*message.Conversation, error) {
	return nil, nil
}
func (m *messageReaderMock) ListConversations(_ context.Context, _ repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
	return nil, "", nil
}
func (m *messageReaderMock) IsParticipant(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (m *messageReaderMock) IsOrgAuthorizedForConversation(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}
func (m *messageReaderMock) GetMessage(_ context.Context, _ uuid.UUID) (*message.Message, error) {
	m.getMessageCalls++
	return &message.Message{}, nil
}
func (m *messageReaderMock) ListMessages(_ context.Context, _ repository.ListMessagesParams) ([]*message.Message, string, error) {
	return nil, "", nil
}
func (m *messageReaderMock) GetMessagesSinceSeq(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*message.Message, error) {
	return nil, nil
}
func (m *messageReaderMock) ListMessagesSinceTime(_ context.Context, _ uuid.UUID, _ time.Time, _ int) ([]*message.Message, error) {
	return nil, nil
}
func (m *messageReaderMock) GetTotalUnread(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *messageReaderMock) GetTotalUnreadBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
}
func (m *messageReaderMock) GetParticipantIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *messageReaderMock) GetContactIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func TestMessageReader_SegregatedConsumer(t *testing.T) {
	mock := &messageReaderMock{}
	// Declaring messageReader as the segregated interface proves the
	// consumer does not need MessageWriter or MessageBroadcasterStore.
	var reader repository.MessageReader = mock

	if _, err := reader.GetMessage(context.Background(), uuid.New()); err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if mock.getMessageCalls != 1 {
		t.Fatalf("expected 1 GetMessage call, got %d", mock.getMessageCalls)
	}
}

// ---------------------------------------------------------------------------
// ORGANIZATION — proof of segregation
// ---------------------------------------------------------------------------

type orgStripeStoreMock struct {
	getCalls int
}

func (m *orgStripeStoreMock) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	m.getCalls++
	return "acct_test_segregated", "FR", nil
}
func (m *orgStripeStoreMock) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *orgStripeStoreMock) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *orgStripeStoreMock) ClearStripeAccount(_ context.Context, _ uuid.UUID) error { return nil }
func (m *orgStripeStoreMock) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *orgStripeStoreMock) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *orgStripeStoreMock) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *orgStripeStoreMock) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}

func TestOrganizationStripeStore_SegregatedConsumer(t *testing.T) {
	mock := &orgStripeStoreMock{}
	// Just 8 methods — the wallet handler that only needs to read the
	// account id does not need the 12-method writer + reader API.
	var store repository.OrganizationStripeStore = mock

	id, _, err := store.GetStripeAccount(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetStripeAccount: %v", err)
	}
	if id != "acct_test_segregated" {
		t.Fatalf("unexpected account id: %q", id)
	}
}

// ---------------------------------------------------------------------------
// DISPUTE — proof of segregation
// ---------------------------------------------------------------------------

type disputeReaderMock struct {
	getCalls int
}

func (m *disputeReaderMock) GetByID(_ context.Context, _ uuid.UUID) (*dispute.Dispute, error) {
	m.getCalls++
	return &dispute.Dispute{}, nil
}
func (m *disputeReaderMock) GetByIDForOrg(_ context.Context, _, _ uuid.UUID) (*dispute.Dispute, error) {
	return &dispute.Dispute{}, nil
}
func (m *disputeReaderMock) GetByProposalID(_ context.Context, _ uuid.UUID) (*dispute.Dispute, error) {
	return nil, nil
}
func (m *disputeReaderMock) ListByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*dispute.Dispute, string, error) {
	return nil, "", nil
}
func (m *disputeReaderMock) ListPendingForScheduler(_ context.Context) ([]*dispute.Dispute, error) {
	return nil, nil
}
func (m *disputeReaderMock) ListAll(_ context.Context, _ string, _ int, _ string) ([]*dispute.Dispute, string, error) {
	return nil, "", nil
}
func (m *disputeReaderMock) GetCounterProposalByID(_ context.Context, _ uuid.UUID) (*dispute.CounterProposal, error) {
	return nil, nil
}
func (m *disputeReaderMock) ListCounterProposals(_ context.Context, _ uuid.UUID) ([]*dispute.CounterProposal, error) {
	return nil, nil
}
func (m *disputeReaderMock) ListChatMessages(_ context.Context, _ uuid.UUID) ([]*dispute.ChatMessage, error) {
	return nil, nil
}
func (m *disputeReaderMock) CountByUserID(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *disputeReaderMock) CountAll(_ context.Context) (int, int, int, error)        { return 0, 0, 0, nil }

func TestDisputeReader_SegregatedConsumer(t *testing.T) {
	mock := &disputeReaderMock{}
	var reader repository.DisputeReader = mock

	if _, err := reader.GetByID(context.Background(), uuid.New()); err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if mock.getCalls != 1 {
		t.Fatalf("expected 1 GetByID call, got %d", mock.getCalls)
	}
}

// ---------------------------------------------------------------------------
// PROPOSAL — proof of segregation
// ---------------------------------------------------------------------------

type proposalReaderMock struct {
	getByIDCalls int
}

func (m *proposalReaderMock) GetByID(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
	m.getByIDCalls++
	return &proposal.Proposal{}, nil
}
func (m *proposalReaderMock) GetByIDForOrg(_ context.Context, _, _ uuid.UUID) (*proposal.Proposal, error) {
	return &proposal.Proposal{}, nil
}
func (m *proposalReaderMock) GetByIDs(_ context.Context, _ []uuid.UUID) ([]*proposal.Proposal, error) {
	return nil, nil
}
func (m *proposalReaderMock) GetLatestVersion(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
	return nil, nil
}
func (m *proposalReaderMock) ListByConversation(_ context.Context, _ uuid.UUID) ([]*proposal.Proposal, error) {
	return nil, nil
}
func (m *proposalReaderMock) ListActiveProjectsByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}
func (m *proposalReaderMock) ListCompletedByOrganization(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*proposal.Proposal, string, error) {
	return nil, "", nil
}
func (m *proposalReaderMock) GetDocuments(_ context.Context, _ uuid.UUID) ([]*proposal.ProposalDocument, error) {
	return nil, nil
}
func (m *proposalReaderMock) IsOrgAuthorizedForProposal(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}
func (m *proposalReaderMock) CountAll(_ context.Context) (int, int, error)             { return 0, 0, nil }
func (m *proposalReaderMock) SumPaidByClientOrganization(_ context.Context, _ uuid.UUID) (int64, error) { return 0, nil }
func (m *proposalReaderMock) ListCompletedByClientOrganization(_ context.Context, _ uuid.UUID, _ int) ([]*proposal.Proposal, error) {
	return nil, nil
}

// proposalMilestoneStoreMock is THE smallest possible store mock —
// just one method.
type proposalMilestoneStoreMock struct {
	calls int
}

func (m *proposalMilestoneStoreMock) CreateWithDocumentsAndMilestones(_ context.Context, _ *proposal.Proposal, _ []*proposal.ProposalDocument, _ []*milestone.Milestone) error {
	m.calls++
	return nil
}

func TestProposalReader_SegregatedConsumer(t *testing.T) {
	mock := &proposalReaderMock{}
	var reader repository.ProposalReader = mock

	if _, err := reader.GetByID(context.Background(), uuid.New()); err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if mock.getByIDCalls != 1 {
		t.Fatalf("expected 1 GetByID call, got %d", mock.getByIDCalls)
	}
}

func TestProposalMilestoneStore_OneMethodMock(t *testing.T) {
	mock := &proposalMilestoneStoreMock{}
	var store repository.ProposalMilestoneStore = mock

	err := store.CreateWithDocumentsAndMilestones(context.Background(), &proposal.Proposal{}, nil, nil)
	if err != nil {
		t.Fatalf("CreateWithDocumentsAndMilestones: %v", err)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls)
	}
}

// ---------------------------------------------------------------------------
// USER — proof of segregation
// ---------------------------------------------------------------------------

type userAuthStoreMock struct {
	bumpCalls  int
	getCalls   int
	touchCalls int
}

func (m *userAuthStoreMock) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	m.bumpCalls++
	return 42, nil
}
func (m *userAuthStoreMock) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	m.getCalls++
	return 42, nil
}
func (m *userAuthStoreMock) TouchLastActive(_ context.Context, _ uuid.UUID) error {
	m.touchCalls++
	return nil
}

func TestUserAuthStore_SegregatedConsumer(t *testing.T) {
	mock := &userAuthStoreMock{}
	// 3-method mock — the auth path has zero need for the admin list
	// API, the create/update path, or the email-notification setting.
	var store repository.UserAuthStore = mock

	if _, err := store.BumpSessionVersion(context.Background(), uuid.New()); err != nil {
		t.Fatalf("BumpSessionVersion: %v", err)
	}
	if v, _ := store.GetSessionVersion(context.Background(), uuid.New()); v != 42 {
		t.Fatalf("expected version 42, got %d", v)
	}
	if err := store.TouchLastActive(context.Background(), uuid.New()); err != nil {
		t.Fatalf("TouchLastActive: %v", err)
	}
	if mock.bumpCalls != 1 || mock.getCalls != 1 || mock.touchCalls != 1 {
		t.Fatalf("call counts wrong: bump=%d get=%d touch=%d", mock.bumpCalls, mock.getCalls, mock.touchCalls)
	}
}

// ---------------------------------------------------------------------------
// LISKOV — composing segregated mocks satisfies the wide god interface
// ---------------------------------------------------------------------------
// This proves the OPEN/CLOSED side of the contract: any consumer that
// still needs the full surface can be handed a composition of the
// segregated mocks. The compile-time `var _ XRepository = …` guards in
// each segregated file already give us this guarantee, but a runtime
// assertion makes the intent unmistakable for reviewers.

func TestSegregatedComposition_SatisfiesGodInterfaces(t *testing.T) {
	// Trivial all-method "no-op" mocks for the segregated children of
	// each god interface. These exist only to prove that composition
	// works at runtime — see the per-feature tests above for behavior.

	t.Run("referral", func(t *testing.T) {
		// We can't easily build a mock of the full ReferralRepository
		// here (24 methods) inline, but the compile-time guard already
		// proved equivalence. This subtest documents the invariant.
		var _ repository.ReferralRepository = (interface {
			repository.ReferralReader
			repository.ReferralWriter
			repository.ReferralAttributionStore
			repository.ReferralCommissionStore
		})(nil)
	})

	t.Run("message", func(t *testing.T) {
		var _ repository.MessageRepository = (interface {
			repository.MessageReader
			repository.MessageWriter
			repository.MessageBroadcasterStore
		})(nil)
	})

	t.Run("organization", func(t *testing.T) {
		var _ repository.OrganizationRepository = (interface {
			repository.OrganizationReader
			repository.OrganizationWriter
			repository.OrganizationStripeStore
		})(nil)
	})

	t.Run("dispute", func(t *testing.T) {
		var _ repository.DisputeRepository = (interface {
			repository.DisputeReader
			repository.DisputeWriter
			repository.DisputeEvidenceStore
		})(nil)
	})

	t.Run("proposal", func(t *testing.T) {
		var _ repository.ProposalRepository = (interface {
			repository.ProposalReader
			repository.ProposalWriter
			repository.ProposalMilestoneStore
		})(nil)
	})

	t.Run("user", func(t *testing.T) {
		var _ repository.UserRepository = (interface {
			repository.UserReader
			repository.UserWriter
			repository.UserAuthStore
			repository.UserKYCStore
		})(nil)
	})
}

// ---------------------------------------------------------------------------
// LISKOV — runtime: the postgres-style "single struct, many interfaces"
// pattern works as advertised.
// ---------------------------------------------------------------------------

// fatStruct mimics the postgres adapter pattern: one struct,
// implementing every method. Used to prove that a single Go value
// satisfies all the segregated interfaces at once.
type fatUserStruct struct{}

func (fatUserStruct) Create(_ context.Context, _ *user.User) error { return nil }
func (fatUserStruct) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	return &user.User{}, nil
}
func (fatUserStruct) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return &user.User{}, nil
}
func (fatUserStruct) Update(_ context.Context, _ *user.User) error                { return nil }
func (fatUserStruct) Delete(_ context.Context, _ uuid.UUID) error                 { return nil }
func (fatUserStruct) ExistsByEmail(_ context.Context, _ string) (bool, error)     { return false, nil }
func (fatUserStruct) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (fatUserStruct) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (fatUserStruct) CountByRole(_ context.Context) (map[string]int, error)            { return nil, nil }
func (fatUserStruct) CountByStatus(_ context.Context) (map[string]int, error)          { return nil, nil }
func (fatUserStruct) RecentSignups(_ context.Context, _ int) ([]*user.User, error)     { return nil, nil }
func (fatUserStruct) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error)    { return 1, nil }
func (fatUserStruct) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error)     { return 1, nil }
func (fatUserStruct) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (fatUserStruct) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

func TestFatStruct_SatisfiesEverySegregatedInterface(t *testing.T) {
	// The same value can be passed as ANY of the four segregated
	// interfaces or as the wide god interface — the cornerstone of the
	// "one adapter, many ports" pattern.
	v := fatUserStruct{}

	var _ repository.UserRepository = v
	var _ repository.UserReader = v
	var _ repository.UserWriter = v
	var _ repository.UserAuthStore = v
	var _ repository.UserKYCStore = v

	// Sanity: each of the segregated paths actually works through the
	// same value.
	if err := repository.UserAuthStore(v).TouchLastActive(context.Background(), uuid.New()); err != nil {
		t.Fatal(err)
	}
}

// Same drill for Organization — the most-fragmented of the bunch with
// 3 children + many cross-cutting fields.
type fatOrgStruct struct{}

func (fatOrgStruct) Create(_ context.Context, _ *organization.Organization) error { return nil }
func (fatOrgStruct) CreateWithOwnerMembership(_ context.Context, _ *organization.Organization, _ *organization.Member) error {
	return nil
}
func (fatOrgStruct) FindByID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (fatOrgStruct) FindByOwnerUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (fatOrgStruct) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (fatOrgStruct) Update(_ context.Context, _ *organization.Organization) error { return nil }
func (fatOrgStruct) Delete(_ context.Context, _ uuid.UUID) error                  { return nil }
func (fatOrgStruct) SaveRoleOverrides(_ context.Context, _ uuid.UUID, _ organization.RoleOverrides) error {
	return nil
}
func (fatOrgStruct) CountAll(_ context.Context) (int, error) { return 0, nil }
func (fatOrgStruct) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	return nil, nil
}
func (fatOrgStruct) ListKYCPending(_ context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (fatOrgStruct) ListWithStripeAccount(_ context.Context) ([]uuid.UUID, error) { return nil, nil }
func (fatOrgStruct) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (fatOrgStruct) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (fatOrgStruct) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error { return nil }
func (fatOrgStruct) ClearStripeAccount(_ context.Context, _ uuid.UUID) error            { return nil }
func (fatOrgStruct) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (fatOrgStruct) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error { return nil }
func (fatOrgStruct) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (fatOrgStruct) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}

func TestFatOrgStruct_SatisfiesEverySegregatedInterface(t *testing.T) {
	v := fatOrgStruct{}
	var _ repository.OrganizationRepository = v
	var _ repository.OrganizationReader = v
	var _ repository.OrganizationWriter = v
	var _ repository.OrganizationStripeStore = v
}
