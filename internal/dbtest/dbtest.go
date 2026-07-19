// Package dbtest provides test helpers backed by a real LadyBug database.
//
// It is separate from internal/testutil so that tests of pure packages can
// use testutil without compiling the cgo-backed store; importing dbtest
// requires the native LadyBug library.
package dbtest

import (
	"testing"

	lbstore "github.com/bendowski/quiver/store/ladybug"
)

// NewMemStore returns an in-memory LadyBug Store suitable for unit tests.
// The test is failed immediately if the store cannot be opened.
func NewMemStore(t *testing.T) *lbstore.Store {
	t.Helper()
	s, err := lbstore.Open("")
	if err != nil {
		t.Fatalf("dbtest.NewMemStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
