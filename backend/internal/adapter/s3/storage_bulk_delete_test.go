package s3_test

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapter "marketplace-backend/internal/adapter/s3"
)

// fakeS3 is a minimal HTTP handler that mimics the S3-compatible
// DeleteObjects endpoint. It records the keys received in the request
// body and replies with a configurable list of deletions / errors.
//
// We cannot reuse a real MinIO testcontainer here because we want
// deterministic per-key error injection. The XML schema mirrors the
// AWS SDK's serialization: one <Object><Key>...</Key></Object> per
// requested key, and the response is a <DeleteResult> with
// <Deleted><Key>...</Key></Deleted> + optional <Error><Key>...</Key>
// <Code>...</Code><Message>...</Message></Error> entries.
type fakeS3 struct {
	mu          *sync.Mutex
	gotKeys     []string
	failKeyCode map[string]string // key -> error code to inject
}

type bulkDeleteReq struct {
	XMLName xml.Name           `xml:"Delete"`
	Objects []bulkDeleteObject `xml:"Object"`
}

type bulkDeleteObject struct {
	Key string `xml:"Key"`
}

type bulkDeleteResp struct {
	XMLName xml.Name              `xml:"DeleteResult"`
	Deleted []bulkDeleteDeleted   `xml:"Deleted"`
	Errors  []bulkDeleteRespError `xml:"Error"`
}

type bulkDeleteDeleted struct {
	Key string `xml:"Key"`
}

type bulkDeleteRespError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("delete") != "" || strings.Contains(r.URL.RawQuery, "delete") {
		body, _ := io.ReadAll(r.Body)
		var req bulkDeleteReq
		if err := xml.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		f.mu.Lock()
		var resp bulkDeleteResp
		for _, o := range req.Objects {
			f.gotKeys = append(f.gotKeys, o.Key)
			if code, bad := f.failKeyCode[o.Key]; bad {
				resp.Errors = append(resp.Errors, bulkDeleteRespError{
					Key: o.Key, Code: code, Message: code + " injected by test",
				})
				continue
			}
			resp.Deleted = append(resp.Deleted, bulkDeleteDeleted{Key: o.Key})
		}
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/xml")
		_ = xml.NewEncoder(w).Encode(resp)
		return
	}
	http.Error(w, "unexpected request", http.StatusInternalServerError)
}

// TestBulkDelete_AllSuccess sanity-checks the happy path: every key
// is reported as deleted with no error. Confirms the adapter wires
// the request and parses the empty <Errors> reply correctly.
func TestBulkDelete_AllSuccess(t *testing.T) {
	fake := &fakeS3{mu: &sync.Mutex{}, failKeyCode: map[string]string{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	svc := adapter.NewStorageService(
		strings.TrimPrefix(srv.URL, "http://"),
		"AKIAEXAMPLE",
		"SECRETEXAMPLE",
		"test-bucket",
		srv.URL+"/test-bucket",
		false,
	)

	keys := []string{"alpha.jpg", "videos/v1.mp4", "kyc/passport.pdf"}
	results, err := svc.BulkDelete(context.Background(), keys)
	require.NoError(t, err)
	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, keys[i], r.Key)
		assert.NoError(t, r.Err, "key %q should not error", keys[i])
	}
	assert.ElementsMatch(t, keys, fake.gotKeys)
}

// TestBulkDelete_PerKeyErrors asserts that S3-reported per-object
// errors are surfaced on the matching BulkDeleteResult.Err while
// successful keys keep nil err — best-effort batch semantics.
func TestBulkDelete_PerKeyErrors(t *testing.T) {
	fake := &fakeS3{mu: &sync.Mutex{}, failKeyCode: map[string]string{
		"locked.jpg": "AccessDenied",
		"missing.mp4": "NoSuchKey",
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	svc := adapter.NewStorageService(
		strings.TrimPrefix(srv.URL, "http://"),
		"AKIAEXAMPLE",
		"SECRETEXAMPLE",
		"test-bucket",
		srv.URL+"/test-bucket",
		false,
	)

	keys := []string{"alpha.jpg", "locked.jpg", "missing.mp4", "ok.pdf"}
	results, err := svc.BulkDelete(context.Background(), keys)
	require.NoError(t, err)
	require.Len(t, results, 4)

	// alpha.jpg + ok.pdf — clean
	assert.NoError(t, results[0].Err)
	assert.NoError(t, results[3].Err)
	// locked.jpg — AccessDenied surfaced
	require.Error(t, results[1].Err)
	assert.Contains(t, results[1].Err.Error(), "AccessDenied")
	// missing.mp4 — NoSuchKey surfaced
	require.Error(t, results[2].Err)
	assert.Contains(t, results[2].Err.Error(), "NoSuchKey")
}

// TestBulkDelete_EmptyInput is a no-op fast path — no HTTP call is
// issued so we don't even need the fake server. Important: empty
// input must NOT produce an empty DeleteObjects request (the API
// rejects that with a 400).
func TestBulkDelete_EmptyInput(t *testing.T) {
	svc := adapter.NewStorageService(
		"localhost:9000",
		"AKIAEXAMPLE",
		"SECRETEXAMPLE",
		"test-bucket",
		"http://localhost:9000/test-bucket",
		false,
	)

	results, err := svc.BulkDelete(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, results)
}
