package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestUploadHandler(
	storage *mockStorageService,
	profiles *mockProfileRepo,
) *UploadHandler {
	return NewUploadHandler(storage, profiles, nil)
}

// buildMultipartRequest creates a multipart form request with a file field.
func buildMultipartRequest(
	method, url, fieldName, fileName, contentType string,
	fileContent []byte,
) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	header.Set("Content-Type", contentType)

	part, _ := writer.CreatePart(header)
	part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// ---------------------------------------------------------------------------
// Magic-byte fixtures — REAL signatures so http.DetectContentType returns
// the right MIME. The previous fixtures were `bytes.Repeat([]byte{0xFF})`
// which the new validation correctly rejects as "unknown".
// ---------------------------------------------------------------------------

// validJPEG returns a minimal byte buffer that http.DetectContentType
// classifies as "image/jpeg". Padded to a few KB so it survives the
// completeness check (needs at least 16 bytes; we pad to 1KB).
func validJPEG() []byte {
	// SOI marker + APP0 (JFIF) header + filler + EOI marker.
	header := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, // SOI + APP0
		0x00, 0x10, // length
		'J', 'F', 'I', 'F', 0x00, // identifier
		0x01, 0x01, // version
		0x00,       // density units
		0x00, 0x01, // x density
		0x00, 0x01, // y density
		0x00, 0x00, // thumbnail dimensions
	}
	body := bytes.Repeat([]byte{0x55}, 1024)
	tail := []byte{0xFF, 0xD9} // EOI
	return append(append(header, body...), tail...)
}

// validPNG returns a minimal byte buffer recognised as "image/png".
func validPNG() []byte {
	// 8-byte PNG signature + 1KB filler + IEND chunk.
	signature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	body := bytes.Repeat([]byte{0x42}, 1024)
	iend := []byte{
		0x00, 0x00, 0x00, 0x00, // length=0
		0x49, 0x45, 0x4E, 0x44, // "IEND"
		0xAE, 0x42, 0x60, 0x82, // CRC
	}
	return append(append(signature, body...), iend...)
}

// validWebP returns a minimal byte buffer recognised as "image/webp".
func validWebP() []byte {
	// RIFF...WEBP magic. 4 + 4 + 4 + 1KB filler.
	header := []byte{
		'R', 'I', 'F', 'F',
		0x00, 0x10, 0x00, 0x00, // file size little-endian (placeholder)
		'W', 'E', 'B', 'P',
		'V', 'P', '8', ' ',
		0x00, 0x10, 0x00, 0x00, // chunk size
	}
	body := bytes.Repeat([]byte{0x33}, 1024)
	return append(header, body...)
}

// validMP4 returns a minimal byte buffer recognised as "video/mp4" by
// http.DetectContentType. The matcher requires the major brand to be
// one of the canonical MP4 brands recognised by net/http (avc1, dash,
// iso2..6, mmp4, mp41, mp42, mp71, msnv, ndas, ndsc, ndsh, ndsm, ndsp,
// ndss, ndxc, ndxh, ndxm, ndxp, ndxs). We pick mp42.
func validMP4() []byte {
	header := []byte{
		0x00, 0x00, 0x00, 0x20, // box size = 0x20
		'f', 't', 'y', 'p',
		'm', 'p', '4', '2', // major brand
		0x00, 0x00, 0x00, 0x00, // minor version
		'm', 'p', '4', '2', // compat brand 1
		'i', 's', 'o', 'm', // compat brand 2
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	body := bytes.Repeat([]byte{0x77}, 4096)
	return append(header, body...)
}

// validWebM returns a minimal byte buffer recognised as "video/webm".
// EBML header magic + DocType "webm".
func validWebM() []byte {
	// EBML header signature (0x1A45DFA3) + skip bytes + DocType "webm".
	header := []byte{
		0x1A, 0x45, 0xDF, 0xA3, // EBML
		0x9F, 0x42, 0x86, 0x81, 0x01, // EBMLVersion=1
		0x42, 0xF7, 0x81, 0x01, // EBMLReadVersion=1
		0x42, 0xF2, 0x81, 0x04, // EBMLMaxIDLength=4
		0x42, 0xF3, 0x81, 0x08, // EBMLMaxSizeLength=8
		0x42, 0x82, 0x84, 'w', 'e', 'b', 'm', // DocType=webm
	}
	body := bytes.Repeat([]byte{0x88}, 4096)
	return append(header, body...)
}

// validPDF returns a minimal byte buffer recognised as "application/pdf".
func validPDF() []byte {
	header := []byte("%PDF-1.4\n")
	body := bytes.Repeat([]byte{0x77}, 1024)
	footer := []byte("\n%%EOF\n")
	return append(append(header, body...), footer...)
}

// disguisedHTML returns bytes that LOOK like HTML — http.DetectContentType
// returns "text/html". The attacker scenario: client claims
// Content-Type: image/png but the body is HTML — magic-byte detection
// must catch this and return 415.
func disguisedHTML() []byte {
	return []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>")
}

// disguisedSVG returns bytes recognised as "image/svg+xml". SVG must be
// REJECTED on every photo endpoint because it can carry inline scripts.
func disguisedSVG() []byte {
	return []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
}

// disguisedExe returns bytes recognised as "application/octet-stream"
// (a binary blob with no recognisable magic). Used to verify .exe-style
// payloads can't sneak through under photo / video scopes.
func disguisedExe() []byte {
	// MZ DOS header — http.DetectContentType returns
	// "application/octet-stream" for unknown binary content.
	return append([]byte{0x4D, 0x5A}, bytes.Repeat([]byte{0x00}, 512)...)
}

// ---------------------------------------------------------------------------
// detectMimeFromBytes — table-driven unit test of the helper itself.
// ---------------------------------------------------------------------------

func TestDetectMimeFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		scope   UploadScope
		wantOK  bool
		wantExt string
	}{
		{name: "JPEG in photo scope", buf: validJPEG(), scope: ScopePhoto, wantOK: true, wantExt: "jpg"},
		{name: "PNG in photo scope", buf: validPNG(), scope: ScopePhoto, wantOK: true, wantExt: "png"},
		{name: "WebP in photo scope", buf: validWebP(), scope: ScopePhoto, wantOK: true, wantExt: "webp"},
		{name: "SVG in photo scope -> rejected", buf: disguisedSVG(), scope: ScopePhoto, wantOK: false},
		{name: "HTML in photo scope -> rejected", buf: disguisedHTML(), scope: ScopePhoto, wantOK: false},
		{name: "exe in photo scope -> rejected", buf: disguisedExe(), scope: ScopePhoto, wantOK: false},
		{name: "PDF in photo scope -> rejected", buf: validPDF(), scope: ScopePhoto, wantOK: false},
		{name: "MP4 in photo scope -> rejected", buf: validMP4(), scope: ScopePhoto, wantOK: false},

		{name: "MP4 in video scope", buf: validMP4(), scope: ScopeVideo, wantOK: true, wantExt: "mp4"},
		{name: "WebM in video scope", buf: validWebM(), scope: ScopeVideo, wantOK: true, wantExt: "webm"},
		{name: "JPEG in video scope -> rejected", buf: validJPEG(), scope: ScopeVideo, wantOK: false},
		{name: "HTML in video scope -> rejected", buf: disguisedHTML(), scope: ScopeVideo, wantOK: false},
		{name: "exe in video scope -> rejected", buf: disguisedExe(), scope: ScopeVideo, wantOK: false},

		{name: "PDF in document scope", buf: validPDF(), scope: ScopeDocument, wantOK: true, wantExt: "pdf"},
		{name: "JPEG in document scope", buf: validJPEG(), scope: ScopeDocument, wantOK: true, wantExt: "jpg"},
		{name: "PNG in document scope", buf: validPNG(), scope: ScopeDocument, wantOK: true, wantExt: "png"},
		{name: "WebP in document scope -> rejected", buf: validWebP(), scope: ScopeDocument, wantOK: false},
		{name: "HTML in document scope -> rejected", buf: disguisedHTML(), scope: ScopeDocument, wantOK: false},
		{name: "MP4 in document scope -> rejected", buf: validMP4(), scope: ScopeDocument, wantOK: false},

		{name: "empty buffer", buf: []byte{}, scope: ScopePhoto, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ext, ok := detectMimeFromBytes(tt.buf, tt.scope)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantExt, ext)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UploadPhoto tests — table-driven exhaustive coverage of SEC-09 / SEC-21.
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name          string
		userID        *uuid.UUID
		fileName      string
		mime          string
		content       []byte
		setupMocks    func(*mockStorageService, *mockProfileRepo)
		wantStatus    int
		wantCode      string
		assertKeyForm func(t *testing.T, key string, fileName string)
	}{
		{
			name:     "success JPEG",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  validJPEG(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
			assertKeyForm: func(t *testing.T, key, fileName string) {
				assert.True(t, strings.HasSuffix(key, ".jpg"), "key must end with .jpg from magic bytes")
				assert.NotContains(t, key, fileName, "client filename must NOT appear in storage key (SEC-21)")
			},
		},
		{
			name:     "success PNG",
			userID:   &uid,
			fileName: "photo.png",
			mime:     "image/png",
			content:  validPNG(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "success WebP",
			userID:   &uid,
			fileName: "photo.webp",
			mime:     "image/webp",
			content:  validWebP(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "SVG rejected (XSS payload could embed <script>)",
			userID:     &uid,
			fileName:   "logo.svg",
			mime:       "image/svg+xml",
			content:    disguisedSVG(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:       "HTML disguised as PNG rejected by magic bytes (SEC-09)",
			userID:     &uid,
			fileName:   "fake.png",
			mime:       "image/png",
			content:    disguisedHTML(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:       "exe renamed .png rejected",
			userID:     &uid,
			fileName:   "evil.png",
			mime:       "image/png",
			content:    disguisedExe(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:       "PDF in photo endpoint rejected",
			userID:     &uid,
			fileName:   "doc.pdf",
			mime:       "application/pdf",
			content:    validPDF(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:       "MP4 in photo endpoint rejected",
			userID:     &uid,
			fileName:   "video.mp4",
			mime:       "video/mp4",
			content:    validMP4(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:       "empty body rejected",
			userID:     &uid,
			fileName:   "empty.jpg",
			mime:       "image/jpeg",
			content:    []byte{},
			wantStatus: http.StatusBadRequest,
			wantCode:   "read_failed",
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			fileName:   "photo.jpg",
			mime:       "image/jpeg",
			content:    validJPEG(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:     "filename with path traversal — key randomised, no traversal (SEC-21)",
			userID:   &uid,
			fileName: "../../../etc/passwd.jpg",
			mime:     "image/jpeg",
			content:  validJPEG(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
			assertKeyForm: func(t *testing.T, key, fileName string) {
				// The malicious filename must NOT influence the storage path.
				assert.NotContains(t, key, "..", "no path traversal in key")
				assert.NotContains(t, key, "passwd", "no leaked filename in key")
				assert.NotContains(t, key, "etc/", "no leaked path in key")
				assert.True(t, strings.HasSuffix(key, ".jpg"), "extension comes from magic bytes only")
			},
		},
		{
			name:     "client lies about Content-Type — magic bytes win",
			userID:   &uid,
			fileName: "fake.jpg",
			mime:     "text/html", // client claim
			content:  validJPEG(), // real magic bytes
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			// The category-mismatch check rejects this even though the
			// magic bytes are valid — defense in depth.
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:     "upload failure",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  validJPEG(),
			setupMocks: func(s *mockStorageService, _ *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "", errors.New("storage error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "upload_failed",
		},
		{
			name:     "profile not found after upload",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  validJPEG(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return nil, profile.ErrProfileNotFound
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "profile_error",
		},
		{
			name:     "profile update failure",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  validJPEG(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
				p.updateFn = func(_ context.Context, _ *profile.Profile) error {
					return errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "update_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := &mockStorageService{}
			profiles := &mockProfileRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(storage, profiles)
			}

			// Capture the storage key for assertions.
			capturedKey := ""
			origUpload := storage.uploadFn
			storage.uploadFn = func(ctx context.Context, key string, r io.Reader, ct string, sz int64) (string, error) {
				capturedKey = key
				if origUpload != nil {
					return origUpload(ctx, key, r, ct, sz)
				}
				return "https://storage.example.com/" + key, nil
			}

			h := newTestUploadHandler(storage, profiles)

			req := buildMultipartRequest(
				http.MethodPost, "/api/v1/upload/photo",
				"file", tc.fileName, tc.mime, tc.content,
			)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.UploadPhoto(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code, "body=%s", rec.Body.String())

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}

			if tc.wantStatus == http.StatusOK {
				var resp map[string]string
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.NotEmpty(t, resp["url"])
				if tc.assertKeyForm != nil {
					tc.assertKeyForm(t, capturedKey, tc.fileName)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UploadPhoto file too large (separate test: needs oversized body)
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto_FileTooLarge(t *testing.T) {
	uid := uuid.New()
	// 6 MB exceeds the 5 MB limit
	oversized := bytes.Repeat([]byte{0xFF}, 6<<20)

	storage := &mockStorageService{}
	profiles := &mockProfileRepo{}
	h := newTestUploadHandler(storage, profiles)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/upload/photo",
		"file", "big.jpg", "image/jpeg", oversized,
	)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UploadPhoto(rec, req)
	// Streaming upload (post-G120) surfaces oversized payloads as
	// 413 Payload Too Large (RFC 7231 §6.5.11). The legacy
	// ParseMultipartForm path returned 400 for the same condition.
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

// ---------------------------------------------------------------------------
// UploadVideo tests
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadVideo(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		fileName   string
		mime       string
		content    []byte
		setupMocks func(*mockStorageService, *mockProfileRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success MP4",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  validMP4(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "success WebM",
			userID:   &uid,
			fileName: "intro.webm",
			mime:     "video/webm",
			content:  validWebM(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			fileName:   "intro.mp4",
			mime:       "video/mp4",
			content:    validMP4(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "PDF in video endpoint rejected",
			userID:     &uid,
			fileName:   "doc.pdf",
			mime:       "application/pdf",
			content:    validPDF(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:       "JPEG in video endpoint rejected",
			userID:     &uid,
			fileName:   "thumb.jpg",
			mime:       "image/jpeg",
			content:    validJPEG(),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   "invalid_type",
		},
		{
			name:     "upload failure",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  validMP4(),
			setupMocks: func(s *mockStorageService, _ *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "", errors.New("storage error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "upload_failed",
		},
		{
			name:     "profile not found after upload",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  validMP4(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return nil, profile.ErrProfileNotFound
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "profile_error",
		},
		{
			name:     "profile update failure",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  validMP4(),
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/" + key, nil
				}
				p.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
				p.updateFn = func(_ context.Context, _ *profile.Profile) error {
					return errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "update_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := &mockStorageService{}
			profiles := &mockProfileRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(storage, profiles)
			}
			h := newTestUploadHandler(storage, profiles)

			req := buildMultipartRequest(
				http.MethodPost, "/api/v1/upload/video",
				"file", tc.fileName, tc.mime, tc.content,
			)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.UploadVideo(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code, "body=%s", rec.Body.String())

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}

			if tc.wantStatus == http.StatusOK {
				var resp map[string]string
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.NotEmpty(t, resp["url"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Property test — randomised filenames must NEVER leak into the storage key.
// Closes SEC-21: a fuzzy filename input cannot influence the bucket path.
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto_FilenameRandomization_PropertyStyle(t *testing.T) {
	uid := uuid.New()

	// Hostile / pathological filenames the attacker might try. We omit
	// control-byte and CRLF-bearing names because Go's multipart parser
	// rejects them BEFORE our handler runs (which is also a fine outcome
	// — a 400 from the parser is just as safe as a 200 with random key).
	maliciousNames := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32\\drivers\\etc\\hosts",
		"normal.jpg",
		"%2e%2e%2fevil.jpg",
		"NUL.png",
		"con.jpg",
		"file with spaces and special chars !@#$%^&*().png",
		"very_long_" + strings.Repeat("a", 200) + ".jpg",
		"trailing/slash/test.jpg",
	}

	for _, name := range maliciousNames {
		t.Run("filename="+name, func(t *testing.T) {
			storage := &mockStorageService{}
			profiles := &mockProfileRepo{}

			capturedKey := ""
			storage.uploadFn = func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
				capturedKey = key
				return "https://storage.example.com/" + key, nil
			}
			profiles.getByOrgIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
				return testProfile(uid), nil
			}

			h := newTestUploadHandler(storage, profiles)

			req := buildMultipartRequest(
				http.MethodPost, "/api/v1/upload/photo",
				"file", name, "image/jpeg", validJPEG(),
			)
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
			ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.UploadPhoto(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

			// Strong guarantees on the captured storage key:
			//   - no path traversal sequences
			//   - no injection of CRLF
			//   - extension fixed to .jpg (from magic bytes, NOT from name)
			//   - prefix shape: profiles/<uuid>/photo/<uuid>.jpg
			assert.NotContains(t, capturedKey, "..", "no path traversal: %s", capturedKey)
			assert.NotContains(t, capturedKey, "\r")
			assert.NotContains(t, capturedKey, "\n")
			assert.True(t, strings.HasSuffix(capturedKey, ".jpg"),
				"extension must come from magic bytes: %s", capturedKey)
			// Bucket prefix must respect profiles/<orgID>/photo/...
			assert.True(t, strings.HasPrefix(capturedKey, "profiles/"+uid.String()+"/photo/"),
				"unexpected key prefix: %s", capturedKey)
		})
	}
}
