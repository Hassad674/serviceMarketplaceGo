package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTarGzDirectory_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("alpha"), 0600))
	nested := filepath.Join(tmp, "sub")
	require.NoError(t, os.MkdirAll(nested, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "b.txt"), []byte("bravo"), 0600))

	archive, err := tarGzDirectory(tmp)
	require.NoError(t, err)
	require.NotEmpty(t, archive)

	gr, err := gzip.NewReader(bytes.NewReader(archive))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	found := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, tr)
		require.NoError(t, err)
		found[hdr.Name] = buf.String()
	}
	assert.Equal(t, "alpha", found["a.txt"])
	assert.Equal(t, "bravo", found["sub/b.txt"])
}

func TestTarGzDirectory_MissingRoot(t *testing.T) {
	// Missing root is treated as "nothing to archive" — returns a
	// valid but empty .tar.gz rather than erroring so the snapshot
	// job does not fail fatally when running from a host that
	// cannot reach the Typesense volume.
	archive, err := tarGzDirectory("/nonexistent/path")
	require.NoError(t, err)
	require.NotEmpty(t, archive)

	gr, err := gzip.NewReader(bytes.NewReader(archive))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	_, err = tr.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestTarGzDirectory_RootIsFile(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "single.txt")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0600))

	_, err := tarGzDirectory(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}
