package referral_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

// validInput returns a NewReferralInput with all required fields populated.
// Tests mutate it field-by-field to exercise validation branches.
func validInput() referral.NewReferralInput {
	return referral.NewReferralInput{
		ReferrerID:     uuid.New(),
		ProviderID:     uuid.New(),
		ClientID:       uuid.New(),
		RatePct:        5.0,
		DurationMonths: 6,
		IntroSnapshot: referral.IntroSnapshot{
			Provider: referral.ProviderSnapshot{
				ExpertiseDomains: []string{"branding", "web design"},
				Region:           "Île-de-France",
				Languages:        []string{"fr", "en"},
			},
			Client: referral.ClientSnapshot{
				Industry:    "Mode / Retail",
				SizeBucket:  "pme",
				Region:      "Île-de-France",
				NeedSummary: "Refonte branding et site web",
			},
		},
		IntroMessageProvider: "Bonjour Jane, ce client cherche exactement ton profil",
		IntroMessageClient:   "Voici un provider top niveau pour votre projet",
	}
}

func TestNewReferral_Valid(t *testing.T) {
	in := validInput()
	r, err := referral.NewReferral(in)

	require.NoError(t, err)
	require.NotNil(t, r)
	assert.NotEqual(t, uuid.Nil, r.ID)
	assert.Equal(t, referral.StatusPendingProvider, r.Status)
	assert.Equal(t, 1, r.Version)
	assert.Equal(t, in.ReferrerID, r.ReferrerID)
	assert.Equal(t, in.ProviderID, r.ProviderID)
	assert.Equal(t, in.ClientID, r.ClientID)
	assert.Equal(t, in.RatePct, r.RatePct)
	assert.Equal(t, in.DurationMonths, r.DurationMonths)
	assert.Equal(t, referral.SnapshotVersion, r.IntroSnapshotVersion)
	assert.Nil(t, r.ActivatedAt)
	assert.Nil(t, r.ExpiresAt)
	assert.False(t, r.LastActionAt.IsZero())
	assert.False(t, r.CreatedAt.IsZero())
	assert.False(t, r.UpdatedAt.IsZero())
}

func TestNewReferral_DefaultsDurationToSix(t *testing.T) {
	in := validInput()
	in.DurationMonths = 0

	r, err := referral.NewReferral(in)
	require.NoError(t, err)
	assert.Equal(t, int16(referral.DefaultDurationMonths), r.DurationMonths)
}

func TestNewReferral_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*referral.NewReferralInput)
		want   error
	}{
		{
			name:   "nil referrer id",
			mutate: func(in *referral.NewReferralInput) { in.ReferrerID = uuid.Nil },
			want:   referral.ErrNotAuthorized,
		},
		{
			name:   "nil provider id",
			mutate: func(in *referral.NewReferralInput) { in.ProviderID = uuid.Nil },
			want:   referral.ErrNotAuthorized,
		},
		{
			name:   "nil client id",
			mutate: func(in *referral.NewReferralInput) { in.ClientID = uuid.Nil },
			want:   referral.ErrNotAuthorized,
		},
		{
			name: "self referral — referrer is provider",
			mutate: func(in *referral.NewReferralInput) {
				in.ProviderID = in.ReferrerID
			},
			want: referral.ErrSelfReferral,
		},
		{
			name: "self referral — referrer is client",
			mutate: func(in *referral.NewReferralInput) {
				in.ClientID = in.ReferrerID
			},
			want: referral.ErrSelfReferral,
		},
		{
			name: "self referral — provider is client",
			mutate: func(in *referral.NewReferralInput) {
				in.ClientID = in.ProviderID
			},
			want: referral.ErrSelfReferral,
		},
		{
			name:   "rate below zero",
			mutate: func(in *referral.NewReferralInput) { in.RatePct = -1 },
			want:   referral.ErrRateOutOfRange,
		},
		{
			name:   "rate above cap",
			mutate: func(in *referral.NewReferralInput) { in.RatePct = 51 },
			want:   referral.ErrRateOutOfRange,
		},
		{
			name:   "rate exactly at cap is valid",
			mutate: func(in *referral.NewReferralInput) { in.RatePct = 50 },
			want:   nil,
		},
		{
			name:   "rate exactly zero is valid",
			mutate: func(in *referral.NewReferralInput) { in.RatePct = 0 },
			want:   nil,
		},
		{
			name:   "duration too low",
			mutate: func(in *referral.NewReferralInput) { in.DurationMonths = -1 },
			want:   referral.ErrDurationOutOfRange,
		},
		{
			name:   "duration too high",
			mutate: func(in *referral.NewReferralInput) { in.DurationMonths = 25 },
			want:   referral.ErrDurationOutOfRange,
		},
		{
			name:   "empty provider message",
			mutate: func(in *referral.NewReferralInput) { in.IntroMessageProvider = "   " },
			want:   referral.ErrEmptyMessage,
		},
		{
			name:   "empty client message",
			mutate: func(in *referral.NewReferralInput) { in.IntroMessageClient = "" },
			want:   referral.ErrEmptyMessage,
		},
		{
			name: "provider message too long",
			mutate: func(in *referral.NewReferralInput) {
				in.IntroMessageProvider = strings.Repeat("a", referral.MaxIntroMessageLen+1)
			},
			want: referral.ErrMessageTooLong,
		},
		{
			name: "client message too long",
			mutate: func(in *referral.NewReferralInput) {
				in.IntroMessageClient = strings.Repeat("z", referral.MaxIntroMessageLen+1)
			},
			want: referral.ErrMessageTooLong,
		},
		{
			name: "snapshot invalid years experience",
			mutate: func(in *referral.NewReferralInput) {
				bad := -1
				in.IntroSnapshot.Provider.YearsExperience = &bad
			},
			want: referral.ErrSnapshotInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.mutate(&in)

			r, err := referral.NewReferral(in)
			if tt.want == nil {
				require.NoError(t, err)
				require.NotNil(t, r)
				return
			}
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, r)
		})
	}
}

// freshReferral returns a Referral pinned in the requested status, with stable
// referrer/provider/client IDs that the actor-permission tests can target.
func freshReferral(t *testing.T, status referral.Status) *referral.Referral {
	t.Helper()
	in := validInput()
	r, err := referral.NewReferral(in)
	require.NoError(t, err)
	// Force the desired starting status without going through the public API
	// (the tests below do that for us in the path leading to the target state,
	// but for transitions like "active → terminate" we want a direct setup).
	r.Status = status
	if status == referral.StatusActive {
		now := time.Now().UTC()
		exp := now.AddDate(0, int(r.DurationMonths), 0)
		r.ActivatedAt = &now
		r.ExpiresAt = &exp
	}
	return r
}

func TestStateMachine_AcceptByProvider(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	prevAction := r.LastActionAt
	time.Sleep(time.Millisecond)

	require.NoError(t, r.AcceptByProvider(r.ProviderID))
	assert.Equal(t, referral.StatusPendingClient, r.Status)
	assert.True(t, r.LastActionAt.After(prevAction))
}

func TestStateMachine_AcceptByProvider_NotAuthorized(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	err := r.AcceptByProvider(r.ReferrerID)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)
	assert.Equal(t, referral.StatusPendingProvider, r.Status)
}

func TestStateMachine_AcceptByProvider_WrongStatus(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingClient)
	err := r.AcceptByProvider(r.ProviderID)
	require.ErrorIs(t, err, referral.ErrInvalidTransition)
}

func TestStateMachine_NegotiateByProvider(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	originalVersion := r.Version

	require.NoError(t, r.NegotiateByProvider(r.ProviderID, 3.5))
	assert.Equal(t, referral.StatusPendingReferrer, r.Status)
	assert.Equal(t, originalVersion+1, r.Version)
	assert.Equal(t, 3.5, r.RatePct)
}

func TestStateMachine_NegotiateByProvider_RateOutOfRange(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	err := r.NegotiateByProvider(r.ProviderID, 99)
	require.ErrorIs(t, err, referral.ErrRateOutOfRange)
	assert.Equal(t, referral.StatusPendingProvider, r.Status)
}

func TestStateMachine_RejectByProvider(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	require.NoError(t, r.RejectByProvider(r.ProviderID, "not interested"))
	assert.Equal(t, referral.StatusRejected, r.Status)
	assert.Equal(t, "not interested", r.RejectionReason)
	require.NotNil(t, r.RejectedBy)
	assert.Equal(t, r.ProviderID, *r.RejectedBy)
}

func TestStateMachine_AcceptByReferrer(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingReferrer)
	require.NoError(t, r.AcceptByReferrer(r.ReferrerID))
	assert.Equal(t, referral.StatusPendingClient, r.Status)
}

func TestStateMachine_NegotiateByReferrer(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingReferrer)
	r.RatePct = 3
	require.NoError(t, r.NegotiateByReferrer(r.ReferrerID, 4))
	assert.Equal(t, referral.StatusPendingProvider, r.Status)
	assert.Equal(t, 4.0, r.RatePct)
	assert.Equal(t, 2, r.Version)
}

func TestStateMachine_RejectByReferrer(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingReferrer)
	require.NoError(t, r.RejectByReferrer(r.ReferrerID, "rate trop bas"))
	assert.Equal(t, referral.StatusRejected, r.Status)
}

func TestStateMachine_AcceptByClient_Activates(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingClient)
	require.NoError(t, r.AcceptByClient(r.ClientID))
	assert.Equal(t, referral.StatusActive, r.Status)
	require.NotNil(t, r.ActivatedAt)
	require.NotNil(t, r.ExpiresAt)
	expectedExp := r.ActivatedAt.AddDate(0, int(r.DurationMonths), 0)
	assert.WithinDuration(t, expectedExp, *r.ExpiresAt, time.Second)
}

func TestStateMachine_AcceptByClient_NotAuthorized(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingClient)
	err := r.AcceptByClient(r.ProviderID)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)
}

func TestStateMachine_RejectByClient(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingClient)
	require.NoError(t, r.RejectByClient(r.ClientID, "pas le bon fit"))
	assert.Equal(t, referral.StatusRejected, r.Status)
}

func TestStateMachine_Cancel(t *testing.T) {
	for _, st := range []referral.Status{
		referral.StatusPendingProvider,
		referral.StatusPendingReferrer,
		referral.StatusPendingClient,
	} {
		t.Run(string(st), func(t *testing.T) {
			r := freshReferral(t, st)
			require.NoError(t, r.Cancel(r.ReferrerID))
			assert.Equal(t, referral.StatusCancelled, r.Status)
		})
	}
}

func TestStateMachine_Cancel_NotAuthorized(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	err := r.Cancel(r.ProviderID)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)
}

func TestStateMachine_Cancel_NotPendingFails(t *testing.T) {
	r := freshReferral(t, referral.StatusActive)
	err := r.Cancel(r.ReferrerID)
	require.ErrorIs(t, err, referral.ErrInvalidTransition)
}

func TestStateMachine_Terminate(t *testing.T) {
	r := freshReferral(t, referral.StatusActive)
	require.NoError(t, r.Terminate(r.ReferrerID))
	assert.Equal(t, referral.StatusTerminated, r.Status)
}

func TestStateMachine_Terminate_OnlyReferrer(t *testing.T) {
	r := freshReferral(t, referral.StatusActive)
	err := r.Terminate(r.ProviderID)
	require.ErrorIs(t, err, referral.ErrNotAuthorized)
}

func TestStateMachine_Terminate_NonActiveFails(t *testing.T) {
	r := freshReferral(t, referral.StatusPendingProvider)
	err := r.Terminate(r.ReferrerID)
	require.ErrorIs(t, err, referral.ErrInvalidTransition)
}

func TestStateMachine_Expire_FromPending(t *testing.T) {
	for _, st := range []referral.Status{
		referral.StatusPendingProvider,
		referral.StatusPendingReferrer,
		referral.StatusPendingClient,
		referral.StatusActive,
	} {
		t.Run(string(st), func(t *testing.T) {
			r := freshReferral(t, st)
			require.NoError(t, r.Expire())
			assert.Equal(t, referral.StatusExpired, r.Status)
		})
	}
}

func TestStateMachine_Expire_TerminalFails(t *testing.T) {
	r := freshReferral(t, referral.StatusRejected)
	err := r.Expire()
	require.ErrorIs(t, err, referral.ErrAlreadyTerminal)
}

func TestStateMachine_TerminalGuard_BlocksAllTransitions(t *testing.T) {
	for _, term := range []referral.Status{
		referral.StatusRejected,
		referral.StatusExpired,
		referral.StatusCancelled,
		referral.StatusTerminated,
	} {
		t.Run(string(term), func(t *testing.T) {
			r := freshReferral(t, term)
			actor := r.ProviderID
			require.ErrorIs(t, r.AcceptByProvider(actor), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.NegotiateByProvider(actor, 3), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.RejectByProvider(actor, "x"), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.AcceptByReferrer(r.ReferrerID), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.NegotiateByReferrer(r.ReferrerID, 3), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.RejectByReferrer(r.ReferrerID, "x"), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.AcceptByClient(r.ClientID), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.RejectByClient(r.ClientID, "x"), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.Cancel(r.ReferrerID), referral.ErrAlreadyTerminal)
			require.ErrorIs(t, r.Terminate(r.ReferrerID), referral.ErrAlreadyTerminal)
		})
	}
}

func TestIsExclusivityActive(t *testing.T) {
	r := freshReferral(t, referral.StatusActive)
	now := time.Now().UTC()
	expFuture := now.Add(24 * time.Hour)
	expPast := now.Add(-24 * time.Hour)

	r.ExpiresAt = &expFuture
	assert.True(t, r.IsExclusivityActive(now))

	r.ExpiresAt = &expPast
	assert.False(t, r.IsExclusivityActive(now))

	r.Status = referral.StatusExpired
	r.ExpiresAt = &expFuture
	assert.False(t, r.IsExclusivityActive(now))
}

func TestStatus_Helpers(t *testing.T) {
	assert.True(t, referral.StatusActive.IsValid())
	assert.False(t, referral.Status("garbage").IsValid())

	assert.True(t, referral.StatusRejected.IsTerminal())
	assert.False(t, referral.StatusActive.IsTerminal())

	assert.True(t, referral.StatusPendingProvider.IsPending())
	assert.True(t, referral.StatusPendingReferrer.IsPending())
	assert.True(t, referral.StatusPendingClient.IsPending())
	assert.False(t, referral.StatusActive.IsPending())

	assert.True(t, referral.StatusActive.LocksCouple())
	assert.True(t, referral.StatusPendingProvider.LocksCouple())
	assert.False(t, referral.StatusRejected.LocksCouple())
}

func TestActorRole_IsValid(t *testing.T) {
	assert.True(t, referral.ActorReferrer.IsValid())
	assert.True(t, referral.ActorProvider.IsValid())
	assert.True(t, referral.ActorClient.IsValid())
	assert.False(t, referral.ActorRole("admin").IsValid())
}
