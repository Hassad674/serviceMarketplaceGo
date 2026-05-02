package observability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// TestInit_NoEndpoint_InstallsNoop verifies that when
// OTEL_EXPORTER_OTLP_ENDPOINT is empty Init wires the SDK's no-op
// tracer provider — zero exporter dial, zero goroutines, zero
// allocations on the hot path.
func TestInit_NoEndpoint_InstallsNoop(t *testing.T) {
	// Cannot t.Parallel — Init mutates the global TracerProvider.

	prevTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	shutdown, err := Init(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Init returned nil shutdown closure")
	}

	// The installed provider must be a tracenoop.TracerProvider.
	tp := otel.GetTracerProvider()
	if _, ok := tp.(tracenoop.TracerProvider); !ok {
		t.Fatalf("expected noop tracer provider, got %T", tp)
	}

	// Shutdown must succeed and be idempotent.
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("first shutdown returned error: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("second shutdown returned error: %v", err)
	}
}

// TestInit_NoEndpoint_TracerStartsNoopSpan verifies that calls to the
// global tracer with the no-op provider produce non-recording spans
// and return the same context — confirms zero side effects.
func TestInit_NoEndpoint_TracerStartsNoopSpan(t *testing.T) {
	// Cannot t.Parallel — Init mutates the global TracerProvider.

	prevTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	if _, err := Init(context.Background(), Config{}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "noop-span")
	defer span.End()

	if span.IsRecording() {
		t.Error("noop span unexpectedly recording")
	}
	if span.SpanContext().HasTraceID() {
		t.Error("noop span unexpectedly has a trace id")
	}
}

// TestInit_WithEndpoint_InstallsRealProvider verifies that a non-empty
// endpoint installs a real SDK provider that produces recording spans.
// The exporter is unreachable in the test (we just point at localhost
// with no listener) but Init must not block — the OTLP exporter dials
// lazily on the first export attempt.
func TestInit_WithEndpoint_InstallsRealProvider(t *testing.T) {
	// Cannot t.Parallel — Init mutates the global TracerProvider.

	prevTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	cfg := Config{
		Endpoint:       "localhost:14317", // unreachable in test
		Protocol:       "grpc",
		Insecure:       true,
		ServiceName:    "test-service",
		ServiceVersion: "0.0.0",
		SamplingRatio:  1.0,
	}

	shutdown, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	})

	// Real SDK provider produces recording spans.
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "real-span")
	defer span.End()

	if !span.IsRecording() {
		t.Error("expected real span to be recording")
	}
	if !span.SpanContext().HasTraceID() {
		t.Error("expected real span to have a trace id")
	}
	if !span.SpanContext().HasSpanID() {
		t.Error("expected real span to have a span id")
	}
}

// TestInit_UnknownProtocol_ReturnsError verifies that an unknown
// protocol value returns a clear error rather than crashing later.
func TestInit_UnknownProtocol_ReturnsError(t *testing.T) {
	// Cannot t.Parallel — Init mutates the global TracerProvider on success.

	prevTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	cfg := Config{
		Endpoint: "localhost:14317",
		Protocol: "carrier-pigeon",
	}

	_, err := Init(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for unknown protocol, got nil")
	}
}

// TestInit_HTTPProtocol_BootsRealProvider exercises the HTTP exporter
// path so the switch in buildExporter is covered.
func TestInit_HTTPProtocol_BootsRealProvider(t *testing.T) {
	// Cannot t.Parallel — Init mutates the global TracerProvider.

	prevTP := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	cfg := Config{
		Endpoint:    "localhost:14318",
		Protocol:    "http/protobuf",
		Insecure:    true,
		ServiceName: "test-http",
	}

	shutdown, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	})

	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "http-span")
	span.End()
	if !span.SpanContext().IsValid() {
		t.Error("expected valid span context with http exporter")
	}
}

// TestInMemoryExporter_RecordsSpans uses the SDK's tracetest exporter
// to confirm a full record-export round trip works end to end. This
// is the foundation other tests use for asserting span attributes.
func TestInMemoryExporter_RecordsSpans(t *testing.T) {
	t.Parallel()

	tp, exp := newTestProvider()
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "captured")
	span.End()

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "captured" {
		t.Errorf("name = %q, want %q", spans[0].Name, "captured")
	}
}

// TestParseKVList covers the OTEL_EXPORTER_OTLP_HEADERS parser.
func TestParseKVList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{"empty", "", nil},
		{"whitespace", "   ", nil},
		{"single", "foo=bar", map[string]string{"foo": "bar"}},
		{"multi", "a=1,b=2", map[string]string{"a": "1", "b": "2"}},
		{"trim", " a = 1 , b = 2 ", map[string]string{"a": "1", "b": "2"}},
		{"missing-eq-dropped", "a=1,oops,b=2", map[string]string{"a": "1", "b": "2"}},
		{"empty-key-dropped", "=oops,b=2", map[string]string{"b": "2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseKVList(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseKVList(%q) length = %d, want %d (got %v)", tt.input, len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// TestClampRatio bounds hostile sampling values into [0,1].
func TestClampRatio(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, out float64
	}{
		{0, 0},
		{1, 1},
		{0.5, 0.5},
		{-1, 0},
		{2, 1},
		{1.5, 1},
	}
	for _, tc := range cases {
		if got := clampRatio(tc.in); got != tc.out {
			t.Errorf("clampRatio(%v) = %v, want %v", tc.in, got, tc.out)
		}
	}
}

// TestParseBool exercises the env-bool parser.
func TestParseBool(t *testing.T) {
	t.Parallel()
	cases := []struct {
		raw      string
		fallback bool
		want     bool
	}{
		{"", true, true},
		{"", false, false},
		{"true", false, true},
		{"TRUE", false, true},
		{"1", false, true},
		{"false", true, false},
		{"0", true, false},
		{"garbage", true, true},
		{"garbage", false, false},
	}
	for _, tc := range cases {
		if got := parseBool(tc.raw, tc.fallback); got != tc.want {
			t.Errorf("parseBool(%q,%v) = %v, want %v", tc.raw, tc.fallback, got, tc.want)
		}
	}
}

// TestLoadFromEnv covers the env reader and default-fallback behaviour.
func TestLoadFromEnv(t *testing.T) {
	// Cannot t.Parallel because we mutate process env vars.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_SERVICE_VERSION", "")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "")

	cfg := LoadFromEnv()
	if cfg.ServiceName != "marketplace-backend" {
		t.Errorf("default service name = %q, want %q", cfg.ServiceName, "marketplace-backend")
	}
	if cfg.ServiceVersion != "dev" {
		t.Errorf("default service version = %q, want %q", cfg.ServiceVersion, "dev")
	}
	if cfg.Protocol != "grpc" {
		t.Errorf("default protocol = %q, want %q", cfg.Protocol, "grpc")
	}
	if cfg.SamplingRatio != 1.0 {
		t.Errorf("default sampling ratio = %v, want 1.0", cfg.SamplingRatio)
	}

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
	t.Setenv("OTEL_SERVICE_NAME", "custom")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "x-tenant=acme,x-key=val")
	cfg = LoadFromEnv()
	if cfg.ServiceName != "custom" {
		t.Errorf("override service name = %q, want %q", cfg.ServiceName, "custom")
	}
	if cfg.Endpoint != "collector:4317" {
		t.Errorf("endpoint = %q, want %q", cfg.Endpoint, "collector:4317")
	}
	if cfg.Headers["x-tenant"] != "acme" {
		t.Errorf("header tenant = %q, want %q", cfg.Headers["x-tenant"], "acme")
	}
}

// Benchmark_NoOpOtelOverhead measures the cost of the global tracer
// when OTel is disabled. The expectation is "essentially free":
//
//   - tracer creation goes through the global no-op
//   - Span.Start returns a non-recording span
//   - End is a no-op
//
// This benchmark guards the "zero overhead when disabled" promise of
// the package. Reported allocs/op should stay at 0 — any regression
// here is a sign the no-op fast path was broken.
func Benchmark_NoOpOtelOverhead(b *testing.B) {
	prevTP := otel.GetTracerProvider()
	b.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	if _, err := Init(context.Background(), Config{}); err != nil {
		b.Fatalf("Init returned error: %v", err)
	}

	tracer := otel.Tracer("bench")
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "noop-bench")
		span.End()
	}
}

// newTestProvider builds an in-memory tracer provider for tests that
// need to assert exported spans without dialling an exporter.
func newTestProvider() (sdkProvider, *tracetest.InMemoryExporter) {
	exp := tracetest.NewInMemoryExporter()
	tp := newProvider(exp)
	return tp, exp
}

// sdkProvider is the small set of TracerProvider methods our tests
// reach for. Avoids importing the sdk package directly in test files
// so they keep working if we ever swap providers.
type sdkProvider interface {
	trace.TracerProvider
	ForceFlush(context.Context) error
	Shutdown(context.Context) error
}
