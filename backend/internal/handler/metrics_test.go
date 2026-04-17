package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMetrics_NilSafe(t *testing.T) {
	var m *Metrics
	// None of these must panic on a nil receiver.
	m.ObserveSearch("freelance", "success", time.Millisecond, 10, false)
	m.ObserveEmbeddingRetry()
	m.ObserveReindex(time.Second)
	m.SetDriftRatio(0.1)
}

func TestMetrics_RendersCounters(t *testing.T) {
	m := NewMetrics()
	m.ObserveSearch("freelance", "success", 50*time.Millisecond, 12, false)
	m.ObserveSearch("freelance", "success", 40*time.Millisecond, 8, false)
	m.ObserveSearch("agency", "error", 100*time.Millisecond, 0, true)
	m.ObserveEmbeddingRetry()
	m.ObserveEmbeddingRetry()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `search_requests_total{persona="freelance",status="success"} 2`) {
		t.Fatalf("missing freelance counter in:\n%s", body)
	}
	if !strings.Contains(body, `search_requests_total{persona="agency",status="error"} 1`) {
		t.Fatalf("missing agency counter")
	}
	if !strings.Contains(body, "search_embedding_retries_total 2") {
		t.Fatalf("missing retries counter")
	}
	if !strings.Contains(body, "# TYPE search_duration_seconds histogram") {
		t.Fatalf("missing histogram type")
	}
}

func TestMetrics_DriftGauge(t *testing.T) {
	m := NewMetrics()
	m.SetDriftRatio(0.0123)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "typesense_drift_ratio 0.0123") {
		t.Fatalf("gauge not rendered: %s", body)
	}
	// Negative values are clamped to zero.
	m.SetDriftRatio(-0.5)
	rec = httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "typesense_drift_ratio 0") {
		t.Fatalf("negative value not clamped: %s", rec.Body.String())
	}
}

func TestMetrics_HistogramBuckets(t *testing.T) {
	m := NewMetrics()
	// 3 fast, 1 slow request.
	m.ObserveSearch("freelance", "success", 10*time.Millisecond, 5, false)
	m.ObserveSearch("freelance", "success", 20*time.Millisecond, 5, false)
	m.ObserveSearch("freelance", "success", 30*time.Millisecond, 5, false)
	m.ObserveSearch("freelance", "success", 3*time.Second, 5, false)

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	// The +Inf bucket MUST equal total count (4).
	if !strings.Contains(body, `search_duration_seconds_bucket{persona="freelance",hybrid="false",le="+Inf"} 4`) {
		t.Fatalf("+Inf bucket wrong:\n%s", body)
	}
	// The count line matches.
	if !strings.Contains(body, `search_duration_seconds_count{persona="freelance",hybrid="false"} 4`) {
		t.Fatalf("count wrong")
	}
}

func TestMetrics_ConcurrentObserve(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.ObserveSearch("freelance", "success", time.Microsecond, 1, false)
				m.ObserveEmbeddingRetry()
			}
		}()
	}
	wg.Wait()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `search_requests_total{persona="freelance",status="success"} 1000`) {
		t.Fatalf("race-free count failed:\n%s", body)
	}
	if !strings.Contains(body, "search_embedding_retries_total 1000") {
		t.Fatalf("retry count wrong")
	}
}
