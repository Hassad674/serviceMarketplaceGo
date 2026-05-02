package stripe

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v82"

	"marketplace-backend/internal/observability"
)

// Service implements port/service.StripeService using the Stripe API.
type Service struct {
	webhookSecret string
}

// NewService initializes the Stripe SDK and returns a new Service.
//
// The Stripe SDK uses package-globals (stripe.Key + the global
// backends) so tests rely on the established pattern of stubbing
// backends BEFORE the unit under test calls Stripe. To preserve that
// pattern this constructor only sets the API key — the OTel
// wrapping of the HTTP transport happens via InstallOTelBackends,
// which the caller must invoke once at process boot (see
// cmd/api/wire_payment.go).
func NewService(secretKey, webhookSecret string) *Service {
	stripe.Key = secretKey
	return &Service{webhookSecret: webhookSecret}
}

// InstallOTelBackends reconfigures the Stripe SDK's API + Connect +
// Uploads backends so every outbound HTTP call is captured as an
// OTel client span. Call once at process boot, before the first
// NewService call. Tests do NOT call this — their stubBackends
// helper installs an httptest server as the backend instead.
func InstallOTelBackends() {
	httpClient := &http.Client{
		Transport: observability.HTTPClientTransport(http.DefaultTransport, "stripe"),
	}
	for _, backendType := range []stripe.SupportedBackend{
		stripe.APIBackend,
		stripe.ConnectBackend,
		stripe.UploadsBackend,
	} {
		stripe.SetBackend(backendType, stripe.GetBackendWithConfig(backendType, &stripe.BackendConfig{
			HTTPClient: httpClient,
		}))
	}
}
