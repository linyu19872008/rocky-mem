package conformance_test

import (
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/store/conformance"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestSQLiteConformance(t *testing.T) {
	conformance.Suite(t, func(t *testing.T) store.Store {
		t.Helper()
		st, err := sqlite.Open(filepath.Join(t.TempDir(), "c.db"))
		if err != nil {
			t.Fatal(err)
		}
		return st
	})
}
