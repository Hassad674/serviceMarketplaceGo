package main

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/app/auth"
	twofactorapp "marketplace-backend/internal/app/twofactor"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// twoFactorWiring carries the products of the B.6 email-2FA feature
// initialisation: the app service (used both to gate Login and to
// power the /me/two-factor/* endpoints) and the gate adapter the
// auth service consumes through the auth.TwoFactorGate interface.
type twoFactorWiring struct {
	Service *twofactorapp.Service
	Gate    auth.TwoFactorGate
}

// twoFactorDeps captures the upstream dependencies the 2FA feature
// needs. Kept narrow so the feature is removable in one delete.
type twoFactorDeps struct {
	DB        *sql.DB
	Hasher    service.HasherService
	Email     service.EmailService
	Audits    repository.AuditRepository
	UserFlags postgres.TwoFactorFlagSetter // satisfied by *postgres.UserRepository
}

// wireTwoFactor brings up the email-2FA feature.
//
//   - Construct the postgres TwoFactorChallengeRepository.
//   - Construct the twofactor app service from its narrow ServiceDeps.
//   - Wrap the service in a tiny adapter that satisfies
//     auth.TwoFactorGate (the auth package can't import the twofactor
//     app package directly without a cycle, so a structurally-typed
//     adapter bridges them).
func wireTwoFactor(deps twoFactorDeps) twoFactorWiring {
	challengeRepo := postgres.NewTwoFactorChallengeRepository(deps.DB)
	svc := twofactorapp.NewService(twofactorapp.ServiceDeps{
		Challenges: challengeRepo,
		Hasher:     deps.Hasher,
		Email:      deps.Email,
		Audits:     deps.Audits,
	})
	gate := &twoFactorGateAdapter{
		flagReader: deps.UserFlags,
		service:    svc,
	}
	return twoFactorWiring{
		Service: svc,
		Gate:    gate,
	}
}

// twoFactorGateAdapter bridges the twofactor app service into the
// auth.TwoFactorGate interface. Done via a 3-method shim because the
// twofactor service's input/output structs differ slightly from the
// auth side's.
type twoFactorGateAdapter struct {
	flagReader postgres.TwoFactorFlagSetter
	service    *twofactorapp.Service
}

func (a *twoFactorGateAdapter) IsEnabledForUser(ctx context.Context, userID uuid.UUID) (bool, error) {
	return a.flagReader.IsEmailTwoFactorEnabled(ctx, userID)
}

func (a *twoFactorGateAdapter) RequestChallenge(ctx context.Context, in auth.TwoFactorChallengeRequest) (uuid.UUID, error) {
	c, err := a.service.RequestChallenge(ctx, twofactorapp.RequestChallengeInput{
		UserID:        in.UserID,
		EmailTo:       in.EmailTo,
		ClientIP:      in.ClientIP,
		UserAgentHash: in.UserAgentHash,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return c.ID, nil
}

func (a *twoFactorGateAdapter) VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error {
	_, err := a.service.VerifyChallenge(ctx, twofactorapp.VerifyChallengeInput{
		UserID: userID,
		Code:   code,
	})
	return err
}

// twoFactorChallengerShim bridges auth.TwoFactorGate (which has 3
// methods, including IsEnabledForUser) into handler.TwoFactorChallenger
// (which only needs RequestChallenge + VerifyChallenge). Without this
// shim Go wouldn't accept the gate variable at the AttachTwoFactor
// call site because handler.TwoFactorChallenger and auth.TwoFactorGate
// are distinct interfaces — even though the gate happens to have a
// superset of the methods.
type twoFactorChallengerShim struct {
	gate auth.TwoFactorGate
}

func (s *twoFactorChallengerShim) RequestChallenge(ctx context.Context, in auth.TwoFactorChallengeRequest) (uuid.UUID, error) {
	return s.gate.RequestChallenge(ctx, in)
}

func (s *twoFactorChallengerShim) VerifyChallenge(ctx context.Context, userID uuid.UUID, code string) error {
	return s.gate.VerifyChallenge(ctx, userID, code)
}

// twoFactorEnablerShim bridges postgres.TwoFactorFlagSetter to
// handler.TwoFactorEnabler. Same trick as above — Go interfaces are
// nominal so we wrap to satisfy both contracts at the call site.
type twoFactorEnablerShim struct {
	flag postgres.TwoFactorFlagSetter
}

func (s *twoFactorEnablerShim) IsEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.flag.IsEmailTwoFactorEnabled(ctx, userID)
}

func (s *twoFactorEnablerShim) SetEmailTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return s.flag.SetEmailTwoFactorEnabled(ctx, userID, enabled)
}

// attachTwoFactorToHandler wires the 2FA-related deps onto the auth
// handler.
func attachTwoFactorToHandler(
	authHandler *handler.AuthHandler,
	authSvc *auth.Service,
	flagRepo postgres.TwoFactorFlagSetter,
	gate auth.TwoFactorGate,
) {
	authHandler.AttachTwoFactor(
		&twoFactorEnablerShim{flag: flagRepo},
		&twoFactorChallengerShim{gate: gate},
		authSvc,
	)
}
