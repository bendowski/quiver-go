// Package testutil provides shared, cgo-free helpers for quiver tests.
// Helpers that need a real LadyBug database live in internal/dbtest, so
// importing this package never pulls in the native library.
package testutil

import (
	"path"
	"testing"

	"github.com/spf13/afero"
)

// NewMemFs returns a new in-memory Afero filesystem.
func NewMemFs() afero.Fs {
	return afero.NewMemMapFs()
}

// WriteGoFile writes src to the slash-separated filePath on fs, creating any
// intermediate directories.
func WriteGoFile(t *testing.T, fs afero.Fs, filePath string, src string) {
	t.Helper()
	if err := fs.MkdirAll(path.Dir(filePath), 0o755); err != nil {
		t.Fatalf("WriteGoFile MkdirAll %s: %v", filePath, err)
	}
	if err := afero.WriteFile(fs, filePath, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteGoFile %s: %v", filePath, err)
	}
}
