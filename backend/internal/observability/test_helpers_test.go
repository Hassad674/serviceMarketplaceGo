package observability

import (
	"go.opentelemetry.io/otel/attribute"
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

// attrMap flattens a slice of attribute.KeyValue into a map keyed by
// the attribute name with the stringified value. Tests use this for
// looking up specific attribute keys without iterating the slice.
func attrMap(in []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(in))
	for _, kv := range in {
		out[string(kv.Key)] = kv.Value.Emit()
	}
	return out
}
