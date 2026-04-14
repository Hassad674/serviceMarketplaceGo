package nominatim

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/port/service"
)

// newTestGeocoder wires a Geocoder against an httptest.Server so
// the tests exercise the real HTTP pipeline (headers, query string,
// JSON decode) without hitting the public Nominatim API.
func newTestGeocoder(t *testing.T, handler http.Handler) (*Geocoder, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	g := NewGeocoder("marketplace-test/1.0")
	g.endpoint = srv.URL
	return g, srv
}

func TestGeocoder_HappyPath(t *testing.T) {
	var capturedUA, capturedCity, capturedCountry string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		capturedCity = r.URL.Query().Get("city")
		capturedCountry = r.URL.Query().Get("countrycodes")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"lat":"48.8566","lon":"2.3522"}]`))
	})

	g, _ := newTestGeocoder(t, handler)

	lat, lng, err := g.Geocode(context.Background(), "Paris", "FR")
	require.NoError(t, err)
	assert.InDelta(t, 48.8566, lat, 0.0001)
	assert.InDelta(t, 2.3522, lng, 0.0001)
	assert.Equal(t, "marketplace-test/1.0", capturedUA)
	assert.Equal(t, "Paris", capturedCity)
	assert.Equal(t, "FR", capturedCountry)
}

func TestGeocoder_EmptyCity(t *testing.T) {
	g := NewGeocoder("test-agent")
	_, _, err := g.Geocode(context.Background(), "", "FR")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
}

func TestGeocoder_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	g, _ := newTestGeocoder(t, handler)

	_, _, err := g.Geocode(context.Background(), "Paris", "FR")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
	assert.Contains(t, err.Error(), "500")
}

func TestGeocoder_EmptyResultArray(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	g, _ := newTestGeocoder(t, handler)

	_, _, err := g.Geocode(context.Background(), "Atlantis", "")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
}

func TestGeocoder_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not a json`))
	})
	g, _ := newTestGeocoder(t, handler)

	_, _, err := g.Geocode(context.Background(), "Paris", "FR")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
}

func TestGeocoder_UnparseableCoordinates(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"lat":"not-a-float","lon":"2.3"}]`))
	})
	g, _ := newTestGeocoder(t, handler)

	_, _, err := g.Geocode(context.Background(), "Paris", "FR")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
}

func TestGeocoder_UnparseableLongitude(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"lat":"48.8","lon":"bogus"}]`))
	})
	g, _ := newTestGeocoder(t, handler)

	_, _, err := g.Geocode(context.Background(), "Paris", "FR")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
}

func TestGeocoder_Timeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Sleep longer than the client timeout so the transport
		// aborts with a context.DeadlineExceeded error.
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`[]`))
	})
	g, _ := newTestGeocoder(t, handler)
	// Override the client timeout to a very small value so the
	// test runs fast.
	g.client = &http.Client{Timeout: 50 * time.Millisecond}

	_, _, err := g.Geocode(context.Background(), "Paris", "FR")
	assert.ErrorIs(t, err, service.ErrGeocodingFailed)
}

func TestGeocoder_WrapsOriginalError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})
	g, _ := newTestGeocoder(t, handler)

	_, _, err := g.Geocode(context.Background(), "Paris", "FR")
	require.Error(t, err)
	// Both errors.Is matching and a descriptive message are expected.
	assert.True(t, errors.Is(err, service.ErrGeocodingFailed))
	assert.Contains(t, err.Error(), "decode")
}
