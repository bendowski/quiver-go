// Package testutil provides shared helpers for quiver tests.
package testutil

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"

	lbstore "github.com/bendowski/quiver/store/ladybug"
)

// NewMemStore returns an in-memory LadyBug Store suitable for unit tests.
// The test is failed immediately if the store cannot be opened.
func NewMemStore(t *testing.T) *lbstore.Store {
	t.Helper()
	s, err := lbstore.Open("")
	if err != nil {
		t.Fatalf("testutil.NewMemStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// NewMemFs returns a new in-memory Afero filesystem.
func NewMemFs() afero.Fs {
	return afero.NewMemMapFs()
}

// WriteGoFile writes src to path on fs, creating any intermediate directories.
func WriteGoFile(t *testing.T, fs afero.Fs, path string, src string) {
	t.Helper()
	if err := fs.MkdirAll(dirOf(path), 0o755); err != nil {
		t.Fatalf("WriteGoFile MkdirAll %s: %v", path, err)
	}
	if err := afero.WriteFile(fs, path, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteGoFile %s: %v", path, err)
	}
}

// dirOf returns the directory component of a file path.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

// FormatCount returns a human-readable "N items" string for test failure messages.
func FormatCount(n int, label string) string {
	return fmt.Sprintf("%d %s", n, label)
}
