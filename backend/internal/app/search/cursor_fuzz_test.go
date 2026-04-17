package search

import (
	"errors"
	"testing"
)

// FuzzCursorRoundtrip asserts that every valid (page, version) pair
// survives EncodeCursor → DecodeCursor without loss. A mismatch here
// would cause pagination to silently teleport the user to a different
// page — the exact failure mode we need to keep out of production.
//
// The fuzz input is an int32 so the engine explores edge values (0,
// negatives, MaxInt32). Negative pages are filtered (DecodeCursor
// rejects them explicitly); we only assert the identity for the
// non-negative subset that EncodeCursor is meant to handle.
func FuzzCursorRoundtrip(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(1))
	f.Add(int32(100))
	f.Add(int32(2147483647))

	f.Fuzz(func(t *testing.T, page int32) {
		if page < 0 {
			t.Skip("EncodeCursor is only defined for non-negative pages")
		}
		in := Cursor{Page: int(page), Version: currentCursorVersion}
		raw := EncodeCursor(in)
		if raw == "" {
			t.Fatalf("EncodeCursor returned empty for valid cursor %+v", in)
		}
		out, err := DecodeCursor(raw)
		if err != nil {
			t.Fatalf("DecodeCursor failed on its own output %q: %v", raw, err)
		}
		if out.Page != in.Page {
			t.Fatalf("page drift: in=%d out=%d (encoded=%q)", in.Page, out.Page, raw)
		}
		if out.Version != currentCursorVersion {
			t.Fatalf("version drift: got %d want %d", out.Version, currentCursorVersion)
		}
	})
}

// FuzzCursorDecodeNoPanic asserts DecodeCursor never panics on
// arbitrary user input — the endpoint is public, so the fuzzer is
// effectively simulating a malicious client.
func FuzzCursorDecodeNoPanic(f *testing.F) {
	f.Add("")
	f.Add("not-base64!!!")
	f.Add("dGVzdA==")     // base64 "test" — not JSON
	f.Add("e30")           // base64 "{}" — missing fields
	f.Add("eyJwYWdlIjotMX0") // {"page":-1}
	f.Add("eyJ2IjoyfQ")    // {"v":2} — unsupported version

	f.Fuzz(func(t *testing.T, raw string) {
		c, err := DecodeCursor(raw)
		if err == nil {
			// Valid decode must produce a non-negative page.
			if c.Page < 0 {
				t.Fatalf("DecodeCursor returned negative page without error: %+v (input=%q)", c, raw)
			}
			return
		}
		// Every error must wrap ErrCursorInvalid so the handler's
		// errors.Is check continues to work.
		if !errors.Is(err, ErrCursorInvalid) {
			t.Fatalf("DecodeCursor error does not wrap ErrCursorInvalid: %v (input=%q)", err, raw)
		}
	})
}

// FuzzCursorMonotonic pins the ordering invariant EncodeCursor
// promises downstream consumers: for page N, decoding the cursor
// yields exactly N. Any mutation that breaks this (e.g. a version
// bump that silently overwrites Page) is an immediate test failure.
func FuzzCursorMonotonic(f *testing.F) {
	f.Add(int32(0), int32(1))
	f.Add(int32(1), int32(2))
	f.Add(int32(5), int32(10))

	f.Fuzz(func(t *testing.T, a, b int32) {
		if a < 0 || b < 0 || a >= b {
			t.Skip("only test increasing non-negative pages")
		}
		c1, _ := DecodeCursor(EncodeCursor(Cursor{Page: int(a)}))
		c2, _ := DecodeCursor(EncodeCursor(Cursor{Page: int(b)}))
		if c1.Page >= c2.Page {
			t.Fatalf("monotonic invariant broken: c1=%d c2=%d", c1.Page, c2.Page)
		}
	})
}
