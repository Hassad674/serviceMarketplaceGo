package main

import (
	"log/slog"

	"marketplace-backend/internal/adapter/livekit"
	redisadapter "marketplace-backend/internal/adapter/redis"
	callapp "marketplace-backend/internal/app/call"
	"marketplace-backend/internal/app/messaging"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"

	goredis "github.com/redis/go-redis/v9"
)

// callDeps captures the upstream dependencies the call feature
// reaches into. The LiveKit URL / API key / API secret come from
// *config.Config; if any of them are missing the helper returns a
// nil handler and the router skips the corresponding routes.
type callDeps struct {
	Cfg          *config.Config
	Redis        *goredis.Client
	Presence     service.PresenceService
	Broadcaster  service.CallBroadcaster
	MessagingSvc *messaging.Service
	UserRepo     repository.UserRepository
}

// wireCall brings up the call feature when LiveKit is configured.
// Returns nil when LiveKitConfigured() is false; the router has a
// `if x != nil` short-circuit so a nil handler simply omits the
// /call/* routes.
//
// LiveKit + call code internals are not modified by this helper —
// only the wiring block previously inlined in main.go has moved.
func wireCall(deps callDeps) *handler.CallHandler {
	// Call feature (optional — only when LiveKit is configured)
	if !deps.Cfg.LiveKitConfigured() {
		slog.Info("call feature disabled (LiveKit not configured)")
		return nil
	}
	lkClient := livekit.NewClient(deps.Cfg.LiveKitURL, deps.Cfg.LiveKitAPIKey, deps.Cfg.LiveKitAPISecret)
	callStateSvc := redisadapter.NewCallStateService(deps.Redis)
	callSvc := callapp.NewService(callapp.ServiceDeps{
		LiveKit:     lkClient,
		CallState:   callStateSvc,
		Presence:    deps.Presence,
		Broadcaster: deps.Broadcaster,
		Messages:    deps.MessagingSvc,
		Users:       deps.UserRepo,
	})
	callHandler := handler.NewCallHandler(callSvc)
	slog.Info("call feature enabled (LiveKit configured)")
	return callHandler
}
