package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects the search-engine counters / histograms / gauges
// needed to expose a Prometheus text-format endpoint. Implementation
// avoids pulling the Prometheus client library into go.sum — the text
// format is stable and trivial to emit by hand.
//
// All methods are safe for concurrent use.
//
// Exposed metrics (scoped to phase 5C observability):
//
//	search_requests_total{persona,status}      counter
//	search_duration_seconds{persona,hybrid}    histogram
//	search_results_count{persona}              histogram
//	search_embedding_retries_total             counter
//	search_reindex_duration_seconds            histogram
//	typesense_drift_ratio                      gauge
//
// Buckets follow the Prometheus-recommended exponential progression
// for sub-second HTTP latency.
type Metrics struct {
	mu sync.RWMutex

	// counters: map of label-string -> count.
	// label-string shape: "persona=\"freelance\",status=\"success\"".
	searchRequestsTotal   map[string]*atomic.Uint64
	embeddingRetriesTotal atomic.Uint64

	// histograms: map of label-string -> *histogram.
	searchDurationSeconds map[string]*histogram
	searchResultsCount    map[string]*histogram
	searchReindexSeconds  *histogram

	// gauges
	typesenseDriftRatio atomic.Int64 // stored as fixed-point (x1e6)
}

// NewMetrics returns a ready-to-use Metrics registry.
func NewMetrics() *Metrics {
	return &Metrics{
		searchRequestsTotal:   make(map[string]*atomic.Uint64),
		searchDurationSeconds: make(map[string]*histogram),
		searchResultsCount:    make(map[string]*histogram),
		searchReindexSeconds:  newHistogram([]float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120}),
	}
}

// -----------------------------------------------------------------
// Observation methods (called from instrumented code paths).
// -----------------------------------------------------------------

func (m *Metrics) ObserveSearch(persona, status string, duration time.Duration, resultCount int, hybrid bool) {
	if m == nil {
		return
	}
	m.counterIncr(m.searchRequestsTotal, fmt.Sprintf(`persona="%s",status="%s"`, persona, status))
	m.histObserve(m.searchDurationSeconds, fmt.Sprintf(`persona="%s",hybrid="%v"`, persona, hybrid), duration.Seconds(),
		[]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5})
	m.histObserve(m.searchResultsCount, fmt.Sprintf(`persona="%s"`, persona), float64(resultCount),
		[]float64{0, 1, 5, 10, 20, 50, 100, 200})
}

func (m *Metrics) ObserveEmbeddingRetry() {
	if m == nil {
		return
	}
	m.embeddingRetriesTotal.Add(1)
}

func (m *Metrics) ObserveReindex(duration time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.searchReindexSeconds.observe(duration.Seconds())
	m.mu.Unlock()
}

// SetDriftRatio records the last-observed Postgres-vs-Typesense drift
// ratio (0.0 to 1.0). Called by the drift-check CLI or worker.
func (m *Metrics) SetDriftRatio(ratio float64) {
	if m == nil {
		return
	}
	if ratio < 0 {
		ratio = 0
	}
	m.typesenseDriftRatio.Store(int64(ratio * 1_000_000))
}

// -----------------------------------------------------------------
// Internal helpers.
// -----------------------------------------------------------------

func (m *Metrics) counterIncr(bucket map[string]*atomic.Uint64, label string) {
	m.mu.RLock()
	c, ok := bucket[label]
	m.mu.RUnlock()
	if !ok {
		m.mu.Lock()
		c, ok = bucket[label]
		if !ok {
			c = new(atomic.Uint64)
			bucket[label] = c
		}
		m.mu.Unlock()
	}
	c.Add(1)
}

func (m *Metrics) histObserve(bucket map[string]*histogram, label string, value float64, buckets []float64) {
	m.mu.RLock()
	h, ok := bucket[label]
	m.mu.RUnlock()
	if !ok {
		m.mu.Lock()
		h, ok = bucket[label]
		if !ok {
			h = newHistogram(buckets)
			bucket[label] = h
		}
		m.mu.Unlock()
	}
	h.observe(value)
}

// histogram is a simple fixed-bucket histogram. Concurrency-safe via
// internal mutex — we prioritise correctness over raw speed.
type histogram struct {
	mu      sync.Mutex
	buckets []float64
	counts  []uint64
	sum     float64
	total   uint64
}

func newHistogram(buckets []float64) *histogram {
	b := make([]float64, len(buckets))
	copy(b, buckets)
	sort.Float64s(b)
	return &histogram{
		buckets: b,
		counts:  make([]uint64, len(b)+1), // +1 for +Inf
	}
}

func (h *histogram) observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sum += v
	h.total++
	for i, b := range h.buckets {
		if v <= b {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.counts)-1]++
}

// -----------------------------------------------------------------
// Prometheus text-format rendering.
// -----------------------------------------------------------------

// Handler returns an http.HandlerFunc that emits the registry in the
// Prometheus exposition format.
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		var sb strings.Builder

		// -------- counters --------
		sb.WriteString("# HELP search_requests_total Count of /api/v1/search calls labelled by persona + outcome.\n")
		sb.WriteString("# TYPE search_requests_total counter\n")
		m.mu.RLock()
		keys := make([]string, 0, len(m.searchRequestsTotal))
		for k := range m.searchRequestsTotal {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("search_requests_total{%s} %d\n", k, m.searchRequestsTotal[k].Load()))
		}
		m.mu.RUnlock()

		sb.WriteString("# HELP search_embedding_retries_total Number of OpenAI embedding retries performed.\n")
		sb.WriteString("# TYPE search_embedding_retries_total counter\n")
		sb.WriteString(fmt.Sprintf("search_embedding_retries_total %d\n", m.embeddingRetriesTotal.Load()))

		// -------- histograms --------
		m.mu.RLock()
		m.renderHistograms(&sb, "search_duration_seconds", "Latency of /api/v1/search requests.", m.searchDurationSeconds)
		m.renderHistograms(&sb, "search_results_count", "Number of results returned per /api/v1/search call.", m.searchResultsCount)
		m.mu.RUnlock()

		sb.WriteString("# HELP search_reindex_duration_seconds Wall-clock duration of bulk reindex runs.\n")
		sb.WriteString("# TYPE search_reindex_duration_seconds histogram\n")
		m.renderHistogram(&sb, "search_reindex_duration_seconds", "", m.searchReindexSeconds)

		// -------- gauge --------
		sb.WriteString("# HELP typesense_drift_ratio Postgres-vs-Typesense doc count drift ratio [0,1].\n")
		sb.WriteString("# TYPE typesense_drift_ratio gauge\n")
		drift := float64(m.typesenseDriftRatio.Load()) / 1_000_000.0
		sb.WriteString(fmt.Sprintf("typesense_drift_ratio %g\n", drift))

		_, _ = w.Write([]byte(sb.String()))
	}
}

func (m *Metrics) renderHistograms(sb *strings.Builder, name, help string, hs map[string]*histogram) {
	if help != "" {
		sb.WriteString("# HELP " + name + " " + help + "\n")
		sb.WriteString("# TYPE " + name + " histogram\n")
	}
	labels := make([]string, 0, len(hs))
	for k := range hs {
		labels = append(labels, k)
	}
	sort.Strings(labels)
	for _, lbl := range labels {
		m.renderHistogram(sb, name, lbl, hs[lbl])
	}
}

func (m *Metrics) renderHistogram(sb *strings.Builder, name, labels string, h *histogram) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	cum := uint64(0)
	labelPrefix := ""
	if labels != "" {
		labelPrefix = labels + ","
	}
	for i, b := range h.buckets {
		cum += h.counts[i]
		sb.WriteString(fmt.Sprintf("%s_bucket{%sle=\"%g\"} %d\n", name, labelPrefix, b, cum))
	}
	cum += h.counts[len(h.counts)-1]
	sb.WriteString(fmt.Sprintf("%s_bucket{%sle=\"+Inf\"} %d\n", name, labelPrefix, cum))
	if labels != "" {
		sb.WriteString(fmt.Sprintf("%s_sum{%s} %g\n", name, labels, h.sum))
		sb.WriteString(fmt.Sprintf("%s_count{%s} %d\n", name, labels, h.total))
	} else {
		sb.WriteString(fmt.Sprintf("%s_sum %g\n", name, h.sum))
		sb.WriteString(fmt.Sprintf("%s_count %d\n", name, h.total))
	}
}
