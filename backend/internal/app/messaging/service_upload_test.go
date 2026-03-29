package messaging

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/message"
)

func TestGetPresignedUploadURL_DisallowedExtensions(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{name: "shell script", filename: "hack.sh"},
		{name: "batch file", filename: "run.bat"},
		{name: "cmd file", filename: "run.cmd"},
		{name: "powershell", filename: "run.ps1"},
		{name: "php file", filename: "shell.php"},
		{name: "jsp file", filename: "page.jsp"},
		{name: "unknown ext", filename: "file.xyz"},
		{name: "no extension", filename: "noext"},
		{name: "double extension", filename: "safe.pdf.exe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil, nil)

			result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
				UserID:      uuid.New(),
				Filename:    tt.filename,
				ContentType: "application/octet-stream",
			})

			assert.ErrorIs(t, err, message.ErrInvalidFileType)
			assert.Empty(t, result.UploadURL)
		})
	}
}

func TestGetPresignedUploadURL_CaseInsensitiveExt(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{name: "uppercase JPG", filename: "photo.JPG"},
		{name: "mixed case Png", filename: "image.Png"},
		{name: "uppercase PDF", filename: "doc.PDF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil, nil)

			result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
				UserID:      uuid.New(),
				Filename:    tt.filename,
				ContentType: "application/octet-stream",
			})

			require.NoError(t, err)
			assert.NotEmpty(t, result.UploadURL)
			assert.NotEmpty(t, result.FileKey)
		})
	}
}

func TestGetPresignedUploadURL_PreservesExtInKey(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)

	result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
		UserID:      uuid.New(),
		Filename:    "report.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	})

	require.NoError(t, err)
	assert.Contains(t, result.FileKey, ".xlsx")
}

func TestGetPresignedUploadURL_MediaTypes(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{name: "gif animation", filename: "anim.gif"},
		{name: "webp image", filename: "photo.webp"},
		{name: "svg icon", filename: "icon.svg"},
		{name: "ogg audio", filename: "sample.ogg"},
		{name: "wav audio", filename: "sound.wav"},
		{name: "webm video", filename: "clip.webm"},
		{name: "m4a audio", filename: "voice.m4a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil, nil)

			result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
				UserID:      uuid.New(),
				Filename:    tt.filename,
				ContentType: "application/octet-stream",
			})

			require.NoError(t, err)
			assert.NotEmpty(t, result.UploadURL)
		})
	}
}

func TestGetPresignedUploadURL_ArchiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{name: "tar archive", filename: "backup.tar"},
		{name: "gz compressed", filename: "data.gz"},
		{name: "rar archive", filename: "files.rar"},
		{name: "7z archive", filename: "bundle.7z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(nil, nil, nil, nil, nil, nil)

			result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
				UserID:      uuid.New(),
				Filename:    tt.filename,
				ContentType: "application/octet-stream",
			})

			require.NoError(t, err)
			assert.NotEmpty(t, result.UploadURL)
		})
	}
}
