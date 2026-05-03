package main

import (
	"context"

	"github.com/go-chi/chi/v5"

	"marketplace-backend/internal/adapter/ws"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
)

// App is the assembled, ready-to-serve product of bootstrap. Holds
// every long-lived handle the HTTP server + its 3-step graceful
// shutdown reach into:
//
//   - Router: the chi root that runServer hands to net/http.
//   - WSHub: drained at shutdown phase 1 so live WS clients receive a
//     proper close frame.
//   - UploadHandler / UploadCancel: drained at phase 2 so in-flight
//     RecordUpload goroutines wind down their downstream work cleanly.
//   - WorkerCancels: every long-running goroutine's context cancel
//     (notifications, pending events, media moderation, KYC scheduler,
//     dispute scheduler, GDPR purger, infra). Drained at phase 3.
//   - OtelShutdown: flushes pending spans before the process exits.
//   - Close: idempotent helper that drives the same 3-step sequence
//     when bootstrap is invoked from a test harness rather than runServer.
//
// Keeping the App boundary explicit lets bootstrap_test.go assert
// "every wired service is non-nil and ready to serve" without having
// to mock os.Signal / os.Exit.
type App struct {
	Cfg           *config.Config
	Router        chi.Router
	WSHub         *ws.Hub
	UploadCancel  context.CancelFunc
	UploadHandler *handler.UploadHandler
	WorkerCancels []context.CancelFunc
	OtelShutdown  func(context.Context) error

	// closeFns are the deferred cancellations bootstrap wired during
	// assembly (infra cancel, otel deferred flush). When the process
	// follows runServer's normal shutdown, the WorkerCancels above
	// drive the same handles. closeFns is the safety net for test
	// harnesses that exit bootstrap without invoking runServer.
	closeFns []func()
}

// Close drives a best-effort shutdown of the resources bootstrap
// created. It is idempotent and safe to call from tests; the normal
// runtime path goes through runServer which exercises the same
// cancels via WorkerCancels + OtelShutdown.
func (a *App) Close() {
	if a == nil {
		return
	}
	for _, fn := range a.closeFns {
		fn()
	}
	a.closeFns = nil
}
