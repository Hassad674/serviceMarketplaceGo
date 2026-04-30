package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateStorageRedirect_Accepts confirms every storage backend
// the application uses today maps to a "safe to redirect" verdict.
// If a new backend is wired in cmd/api/main.go and the test fails,
// add the new host suffix to allowedRedirectHostSuffixes.
func TestValidateStorageRedirect_Accepts(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"S3 production presigned", "https://my-bucket.s3.eu-west-3.amazonaws.com/profiles/abc.jpg?X-Amz-Algorithm=…"},
		{"S3 path-style", "https://s3.eu-west-3.amazonaws.com/my-bucket/profiles/abc.jpg"},
		{"Cloudflare R2 v2", "https://abc.r2.cloudflarestorage.com/abc.jpg?X-Amz-Algorithm=…"},
		{"Cloudflare R2 public alias", "https://pub-abcdef0123.r2.dev/abc.jpg"},
		{"local minio docker hostname", "http://minio:9000/bucket/abc.jpg"},
		{"local minio loopback", "http://127.0.0.1:9000/bucket/abc.jpg"},
		{"local minio localhost", "http://localhost:9000/bucket/abc.jpg"},
		{"R2 v1 with deep query string", "https://x.r2.cloudflarestorage.com/path/file.pdf?X-Amz-Signature=deadbeef&Expires=999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := validateStorageRedirect(tt.raw)
			assert.NoError(t, err, "expected accepted: %s", tt.raw)
			assert.NotNil(t, u)
		})
	}
}

// TestValidateStorageRedirect_Rejects exhaustively covers the attack
// surface gosec G710 was warning about. Each row is a real-world
// open-redirect payload tried against production-grade web apps.
func TestValidateStorageRedirect_Rejects(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty string", ""},
		{"javascript URI", "javascript:alert(1)"},
		{"data URI HTML", "data:text/html,<script>alert(1)</script>"},
		{"vbscript URI", "vbscript:msgbox(1)"},
		{"file URI", "file:///etc/passwd"},
		{"protocol-relative double-slash", "//evil.com/redirect"},
		{"path-only relative", "/path/to/resource"},
		{"http with attacker host", "http://evil.com/file.pdf"},
		{"https with attacker host", "https://attacker.example/file.pdf"},
		{"suffix-confusion: amazonaws.com.attacker.com", "https://amazonaws.com.attacker.com/x"},
		{"suffix-confusion: r2.dev.attacker.com", "https://r2.dev.attacker.com/x"},
		{"suffix-confusion: minio.attacker.com", "https://minio.attacker.com/x"},
		{"empty host with auth-style trick", "https:///path"},
		{"CRLF header smuggling", "https://my-bucket.s3.amazonaws.com/x\r\nLocation: evil.com"},
		{"trailing CRLF", "https://my-bucket.s3.amazonaws.com/x\n"},
		{"valid scheme but no host", "http://"},
		{"unparseable URL", "::::not a url::::"},
		{"IPv6 attacker", "https://[2001:db8::1]/path"},
		{"private RFC1918", "https://10.0.0.1/foo"},
		{"link-local", "https://169.254.169.254/latest/meta-data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := validateStorageRedirect(tt.raw)
			assert.ErrorIs(t, err, errRedirectNotAllowed,
				"expected reject: %s, got u=%v err=%v", tt.raw, u, err)
		})
	}
}

// TestHostMatchesSuffix locks down the suffix-matching rules so a
// future refactor cannot accidentally permit "amazonaws.com.evil.com".
func TestHostMatchesSuffix(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		suffix string
		want   bool
	}{
		{"exact match", "minio", "minio", true},
		{"exact match localhost", "localhost", "localhost", true},
		{"subdomain matches dot suffix", "x.amazonaws.com", ".amazonaws.com", true},
		{"deep subdomain matches dot suffix", "a.b.c.amazonaws.com", ".amazonaws.com", true},
		{"different host: rejected", "evil.com", ".amazonaws.com", false},
		{"suffix-confusion attack: rejected", "amazonaws.com.attacker.com", ".amazonaws.com", false},
		{"prefix collision: rejected", "amazonawscomx.com", ".amazonaws.com", false},
		{"no leading dot, partial: rejected", "minioattacker", "minio", false},
		{"empty host: rejected", "", ".amazonaws.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostMatchesSuffix(tt.host, tt.suffix)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidateStorageRedirect_PreservesQuery confirms presigned URL
// query params (X-Amz-Signature, Expires, …) survive validation —
// without them the download URL is useless.
func TestValidateStorageRedirect_PreservesQuery(t *testing.T) {
	raw := "https://my-bucket.s3.amazonaws.com/file.pdf?X-Amz-Signature=abc&X-Amz-Expires=300"
	u, err := validateStorageRedirect(raw)
	assert.NoError(t, err)
	assert.Equal(t, "abc", u.Query().Get("X-Amz-Signature"))
	assert.Equal(t, "300", u.Query().Get("X-Amz-Expires"))
}
