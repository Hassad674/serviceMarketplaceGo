package observability

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newProvider builds a TracerProvider wired with a synchronous
// span processor + the supplied in-memory exporter. The processor is
// synchronous so tests do not need to call ForceFlush before assertion.
func newProvider(exp *tracetest.InMemoryExporter) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
}
