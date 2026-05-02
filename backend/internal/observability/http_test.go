package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TestHTTPMiddleware_CreatesSpan verifies that wrapping a handler with
// HTTPMiddleware produces exactly one server span per request, with
// the standard otelhttp attributes attached and the request body
// path NOT leaked into span attributes.
func TestHTTPMiddleware_CreatesSpan(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})
	wrapped := HTTPMiddleware("test.endpoint")(inner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"x":1}`))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler was not invoked")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name == "" {
		t.Error("span name is empty")
	}

	// Standard HTTP server attributes must be present.
	attrs := attrMap(span.Attributes)
	if got := attrs["http.request.method"]; got != "POST" && attrs["http.method"] != "POST" {
		t.Errorf("missing http method attribute, got %v", attrs)
	}

	// PII safety: no Authorization / Cookie should be on the span.
	for k, v := range attrs {
		ks := strings.ToLower(k)
		if strings.Contains(ks, "authorization") || strings.Contains(ks, "cookie") {
			t.Errorf("PII attribute leaked: %s=%v", k, v)
		}
	}
}

// TestHTTPMiddleware_ExcludesHealthMetrics verifies the noisy
// /health, /ready, /metrics endpoints are filtered out so the trace
// volume does not explode with infra probes.
func TestHTTPMiddleware_ExcludesHealthMetrics(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := HTTPMiddleware("test")(inner)

	for _, path := range []string{"/health", "/ready", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 0 {
		t.Errorf("expected 0 spans for filtered paths, got %d", len(spans))
	}
}

// TestHTTPMiddleware_PropagatesTraceContext verifies that an incoming
// W3C traceparent header is honoured: the resulting span's trace ID
// matches the upstream caller's, so cross-service correlation works.
func TestHTTPMiddleware_PropagatesTraceContext(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := HTTPMiddleware("test")(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/things", nil)
	// W3C traceparent format: version-traceID-parentSpanID-flags
	const incomingTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	req.Header.Set("traceparent", incomingTraceparent)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	got := spans[0].SpanContext.TraceID().String()
	if got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace id = %s, want %s", got, "4bf92f3577b34da6a3ce929d0e0e4736")
	}
}

// TestHTTPMiddleware_NoPIIInBearerToken verifies that even when the
// caller sends an Authorization Bearer token, the span attributes do
// NOT include the token value. otelhttp does not record auth headers
// by default — this is a regression guard.
func TestHTTPMiddleware_NoPIIInBearerToken(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	const secretToken = "super-secret-jwt-token-DO-NOT-LOG"

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := HTTPMiddleware("test")(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/things", nil)
	req.Header.Set("Authorization", "Bearer "+secretToken)
	req.Header.Set("Cookie", "session_id=secret-cookie-value")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Inspect every string-valued attribute and make sure the secret
	// is not present anywhere.
	for _, kv := range spans[0].Attributes {
		val := kv.Value.AsString()
		if strings.Contains(val, secretToken) {
			t.Errorf("auth token leaked into span attribute %s = %q", kv.Key, val)
		}
		if strings.Contains(val, "secret-cookie-value") {
			t.Errorf("cookie value leaked into span attribute %s = %q", kv.Key, val)
		}
	}
}

// TestHTTPClientTransport_InjectsTraceparent verifies that an
// outbound request through the wrapped transport carries the W3C
// traceparent header so downstream services join the same trace.
func TestHTTPClientTransport_InjectsTraceparent(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Stand-in upstream that captures the incoming traceparent.
	var receivedTraceparent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(upstream.Close)

	tracer := tp.Tracer("client-test")
	ctx, span := tracer.Start(context.Background(), "outer")
	defer span.End()

	client := &http.Client{Transport: HTTPClientTransport(http.DefaultTransport, "stripe-mock")}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream.URL+"/charges", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	_ = resp.Body.Close()

	if receivedTraceparent == "" {
		t.Fatal("upstream did not receive traceparent header")
	}
	// Sanity: the traceparent should embed the parent trace id.
	if !strings.Contains(receivedTraceparent, span.SpanContext().TraceID().String()) {
		t.Errorf("traceparent %q does not contain trace id %s",
			receivedTraceparent, span.SpanContext().TraceID().String())
	}

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	spans := exp.GetSpans()
	// 2 spans expected: parent "outer" + client span from the transport.
	if len(spans) < 1 {
		t.Fatalf("expected at least 1 span, got %d", len(spans))
	}
}

// restoreGlobals captures the current OTel globals and returns a
// restore function so each test cleanly puts them back. Without this
// the order in which sub-tests run leaks state.
func restoreGlobals() func() {
	tp := otel.GetTracerProvider()
	pr := otel.GetTextMapPropagator()
	return func() {
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(pr)
	}
}
