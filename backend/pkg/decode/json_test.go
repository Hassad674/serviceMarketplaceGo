package decode

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type sample struct {
	Name string `json:"name"`
	Age  int    `json:"age,omitempty"`
}

func newRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestDecodeBody_Valid(t *testing.T) {
	var s sample
	r := newRequest(`{"name":"Alice","age":30}`)
	w := httptest.NewRecorder()
	if err := DecodeBody(w, r, &s, DefaultMaxBodyBytes); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.Name != "Alice" || s.Age != 30 {
		t.Fatalf("unexpected decoded value: %+v", s)
	}
}

func TestDecodeBody_UnknownField(t *testing.T) {
	var s sample
	r := newRequest(`{"name":"Bob","extra":"oops"}`)
	w := httptest.NewRecorder()
	err := DecodeBody(w, r, &s, DefaultMaxBodyBytes)
	if err == nil {
		t.Fatal("expected ErrUnknownField, got nil")
	}
	if !errors.Is(err, ErrUnknownField) {
		t.Fatalf("expected ErrUnknownField, got %v", err)
	}
	if !strings.Contains(err.Error(), "extra") {
		t.Fatalf("expected error to mention the unknown field name, got %q", err.Error())
	}
}

func TestDecodeBody_BodyTooLarge(t *testing.T) {
	var s sample
	// Build a body with valid JSON shape but well past our cap.
	long := strings.Repeat("x", 1024)
	body := `{"name":"` + long + `"}`
	r := newRequest(body)
	w := httptest.NewRecorder()
	// Cap below the body size on purpose so MaxBytesReader trips.
	err := DecodeBody(w, r, &s, 32)
	if err == nil {
		t.Fatal("expected ErrBodyTooLarge, got nil")
	}
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge, got %v", err)
	}
}

func TestDecodeBody_EmptyBody(t *testing.T) {
	var s sample
	r := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	w := httptest.NewRecorder()
	err := DecodeBody(w, r, &s, DefaultMaxBodyBytes)
	if err == nil {
		t.Fatal("expected ErrEmptyBody, got nil")
	}
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("expected ErrEmptyBody, got %v", err)
	}
}

func TestDecodeBody_Malformed(t *testing.T) {
	var s sample
	r := newRequest(`{"name":`)
	w := httptest.NewRecorder()
	err := DecodeBody(w, r, &s, DefaultMaxBodyBytes)
	if err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
	// Must NOT classify as unknown-field or empty-body.
	if errors.Is(err, ErrUnknownField) || errors.Is(err, ErrEmptyBody) || errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("malformed JSON should be a generic decode error, got %v", err)
	}
	if !strings.Contains(err.Error(), "decode: invalid JSON") {
		t.Fatalf("expected wrapped decode error, got %q", err.Error())
	}
}

func TestDecodeBody_TrailingContent(t *testing.T) {
	var s sample
	// Two concatenated JSON objects — decoder accepts the first then
	// our More() check rejects the rest. Closes a JSON-smuggling vector.
	r := newRequest(`{"name":"a"}{"name":"b"}`)
	w := httptest.NewRecorder()
	err := DecodeBody(w, r, &s, DefaultMaxBodyBytes)
	if err == nil {
		t.Fatal("expected error on trailing content, got nil")
	}
	if !strings.Contains(err.Error(), "trailing content") {
		t.Fatalf("expected trailing-content error, got %q", err.Error())
	}
}

func TestDecodeBody_NilRequest(t *testing.T) {
	var s sample
	w := httptest.NewRecorder()
	err := DecodeBody(w, nil, &s, DefaultMaxBodyBytes)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("expected ErrEmptyBody on nil request, got %v", err)
	}
}

func TestDecodeBody_DefaultsMaxBytesWhenZero(t *testing.T) {
	var s sample
	r := newRequest(`{"name":"ok"}`)
	w := httptest.NewRecorder()
	// Pass 0 to verify the default 1 MiB cap kicks in.
	if err := DecodeBody(w, r, &s, 0); err != nil {
		t.Fatalf("expected default cap to allow small body, got %v", err)
	}
	if s.Name != "ok" {
		t.Fatalf("unexpected decode result: %+v", s)
	}
}
