// Package observability wires OpenTelemetry tracing for the API.
//
// The package follows the OTLP standard so the deployment chooses the
// backend (Jaeger / Honeycomb / Datadog / Grafana Tempo). No vendor
// SDK is imported — only the OTLP exporter and the OpenTelemetry SDK.
//
// Zero-overhead default
//
//	When OTEL_EXPORTER_OTLP_ENDPOINT is empty (the default in dev / CI)
//	Init returns a no-op shutdown closure and installs a no-op tracer
//	provider. No goroutines are started, no exporter is dialled, no
//	span buffer is allocated. Application code can call
//	otel.Tracer(...).Start(ctx, ...) safely — the call resolves to the
//	no-op SDK and incurs only the fixed cost of the interface dispatch.
//
// Standard environment variables
//
//	OTEL_EXPORTER_OTLP_ENDPOINT  — gRPC endpoint (e.g. otel-collector:4317)
//	OTEL_EXPORTER_OTLP_PROTOCOL  — "grpc" (default) or "http/protobuf"
//	OTEL_EXPORTER_OTLP_HEADERS   — comma-separated key=value pairs
//	OTEL_EXPORTER_OTLP_INSECURE  — "true" disables TLS (gRPC only)
//	OTEL_SERVICE_NAME            — defaults to "marketplace-backend"
//	OTEL_SERVICE_VERSION         — defaults to "dev"
//	OTEL_RESOURCE_ATTRIBUTES     — extra resource attrs (key=value,...)
//	OTEL_TRACES_SAMPLER_ARG      — sampling ratio 0..1 (default 1)
//
// PII safety
//
//	No span attributes derived from request bodies or auth tokens are
//	emitted by this package. Downstream instrumentation (HTTP, DB,
//	Redis, outbound HTTP) is configured through helpers exported from
//	this package — see HTTPMiddleware, WrapDB, InstrumentRedis,
//	WrapHTTPClient — every helper applies the same minimum-attribute
//	policy so callers cannot leak PII by accident.
package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// ShutdownFunc flushes pending spans and tears down the exporter.
// Safe to call multiple times (subsequent calls return nil). Always
// non-nil — callers may defer it unconditionally.
type ShutdownFunc func(ctx context.Context) error

// Config captures the boot-time inputs for the OTel pipeline. The
// zero value (empty Endpoint) selects the no-op path, which is the
// expected default in development and CI.
type Config struct {
	// Endpoint is the OTLP endpoint (host:port for gRPC, full URL for HTTP).
	// Reads from OTEL_EXPORTER_OTLP_ENDPOINT when not set explicitly.
	// Empty value selects the no-op exporter.
	Endpoint string

	// Protocol is "grpc" (default) or "http/protobuf". Reads from
	// OTEL_EXPORTER_OTLP_PROTOCOL when not set.
	Protocol string

	// Headers contains the OTLP exporter headers (e.g. tenant id /
	// API key for Honeycomb). Reads from OTEL_EXPORTER_OTLP_HEADERS.
	Headers map[string]string

	// Insecure disables TLS for the gRPC exporter. Reads from
	// OTEL_EXPORTER_OTLP_INSECURE.
	Insecure bool

	// ServiceName defaults to "marketplace-backend".
	ServiceName string

	// ServiceVersion defaults to "dev".
	ServiceVersion string

	// Environment is stamped into the resource (e.g. "production").
	Environment string

	// SamplingRatio caps the trace sample rate (0..1). 1 (the default)
	// records every trace; 0 records none. Sub-sampling is useful for
	// hot paths in production where 1:1 traces cost too much.
	SamplingRatio float64

	// ResourceAttributes are extra resource attributes appended to the
	// SDK-detected set. Reads from OTEL_RESOURCE_ATTRIBUTES.
	ResourceAttributes map[string]string
}

// LoadFromEnv reads the standard OTEL_* environment variables. Unset
// values fall back to documented defaults. The result is safe to pass
// directly to Init.
func LoadFromEnv() Config {
	cfg := Config{
		Endpoint:           strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		Protocol:           strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")),
		Headers:            parseKVList(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS")),
		Insecure:           parseBool(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), false),
		ServiceName:        strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")),
		ServiceVersion:     strings.TrimSpace(os.Getenv("OTEL_SERVICE_VERSION")),
		Environment:        strings.TrimSpace(os.Getenv("ENV")),
		SamplingRatio:      parseFloat(os.Getenv("OTEL_TRACES_SAMPLER_ARG"), 1.0),
		ResourceAttributes: parseKVList(os.Getenv("OTEL_RESOURCE_ATTRIBUTES")),
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "marketplace-backend"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "dev"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "grpc"
	}
	return cfg
}

// noopShutdown is returned when OTel is disabled. Always nil so callers
// can defer the closure unconditionally.
func noopShutdown(context.Context) error { return nil }

// Init wires the global TracerProvider + propagator. When cfg.Endpoint
// is empty the function installs the OTel SDK no-op tracer and returns
// a no-op shutdown closure — zero goroutines, zero allocations on the
// hot path.
//
// When cfg.Endpoint is set the function dials the OTLP exporter,
// builds a batching SpanProcessor, and registers the resulting
// TracerProvider as the global one. The returned shutdown closure
// drains the SpanProcessor (Shutdown) and the exporter — callers must
// invoke it within the graceful-shutdown budget so spans are flushed
// before the process exits.
func Init(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		// Wire the no-op tracer + the standard propagators so callers
		// that depend on context propagation (e.g. otelhttp) keep
		// behaving as expected — the W3C trace headers still round-
		// trip; only span recording is suppressed.
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		otel.SetTextMapPropagator(defaultPropagator())
		slog.Info("otel: disabled (OTEL_EXPORTER_OTLP_ENDPOINT empty)")
		return noopShutdown, nil
	}

	exp, err := buildExporter(ctx, cfg)
	if err != nil {
		return noopShutdown, fmt.Errorf("otel: build exporter: %w", err)
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		// Never fatal — fall back to the SDK default resource
		// (process + host attributes only) and continue.
		slog.Warn("otel: build resource failed, using default", "error", err)
		res = resource.Default()
	}

	bsp := sdktrace.NewBatchSpanProcessor(exp,
		sdktrace.WithBatchTimeout(5*time.Second),
		sdktrace.WithMaxQueueSize(2048),
		sdktrace.WithMaxExportBatchSize(512),
	)

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(clampRatio(cfg.SamplingRatio)))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(defaultPropagator())
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Warn("otel: internal error", "error", err)
	}))

	slog.Info("otel: enabled",
		"service_name", cfg.ServiceName,
		"service_version", cfg.ServiceVersion,
		"protocol", cfg.Protocol,
		"sampling_ratio", clampRatio(cfg.SamplingRatio),
	)

	return func(shutdownCtx context.Context) error {
		// First flush + stop the span processor (drains the in-memory
		// queue), then shut down the exporter cleanly. The two errors
		// are joined so callers see both if both fail.
		var errs []error
		if err := tp.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		}
		// tp.Shutdown already closes the exporter, but if the user
		// shadowed the exporter into a custom processor we still want
		// a best-effort close.
		if err := exp.Shutdown(shutdownCtx); err != nil &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) {
			errs = append(errs, fmt.Errorf("exporter shutdown: %w", err))
		}
		return errors.Join(errs...)
	}, nil
}

// buildExporter dials the OTLP exporter for the requested protocol.
// gRPC is the default; HTTP is supported for environments where gRPC
// is firewalled or proxied.
func buildExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	protocol := strings.ToLower(strings.TrimSpace(cfg.Protocol))
	switch protocol {
	case "", "grpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithTimeout(10 * time.Second),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
		}
		return otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	case "http", "http/protobuf":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
			otlptracehttp.WithTimeout(10 * time.Second),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
		}
		return otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
	default:
		return nil, fmt.Errorf("unknown OTEL protocol %q (expected grpc or http/protobuf)", protocol)
	}
}

// buildResource composes the SDK-detected resource (process, host,
// container) with the service identity and any extra attributes. The
// service name + version are always set so traces in the backend can
// be filtered without a downstream reprocessor.
func buildResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
	}
	if cfg.Environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironment(cfg.Environment))
	}
	for k, v := range cfg.ResourceAttributes {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		attrs = append(attrs, attribute.String(k, v))
	}
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(attrs...),
	)
}

// defaultPropagator wires the W3C TraceContext + Baggage propagators
// so spans round-trip across HTTP boundaries without any per-call
// configuration.
func defaultPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// parseKVList parses "k1=v1,k2=v2" into a map. Empty input returns
// nil. Malformed pairs are silently dropped — the OTel spec calls for
// a best-effort parse.
func parseKVList(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := make(map[string]string)
	for _, kv := range strings.Split(raw, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		eq := strings.IndexRune(kv, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(kv[:eq])
		v := strings.TrimSpace(kv[eq+1:])
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// parseBool returns true for "1", "t", "true" (case-insensitive).
func parseBool(raw string, fallback bool) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "t", "true", "yes", "y":
		return true
	case "0", "f", "false", "no", "n":
		return false
	}
	return fallback
}

// parseFloat returns the parsed value or the fallback.
func parseFloat(raw string, fallback float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

// clampRatio bounds the ratio into [0,1] so a hostile env var cannot
// crash the SDK.
func clampRatio(r float64) float64 {
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}
