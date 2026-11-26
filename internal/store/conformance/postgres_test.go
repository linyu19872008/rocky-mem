package conformance_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/store"
	"github.com/linyu19872008/rocky-mem/internal/store/conformance"
	"github.com/linyu19872008/rocky-mem/internal/store/postgres"
)

// TestPostgresConformance runs the shared Suite against PostgreSQL.
//
// Requires ROCKYMEM_TEST_PG_DSN (e.g. postgres://rocky:rocky@127.0.0.1:5432/rocky?sslmode=disable).
// Each case creates an isolated schema, migrates into it, then drops the schema.
// Skipped when the env var is unset.
func TestPostgresConformance(t *testing.T) {
	dsn := os.Getenv("ROCKYMEM_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("ROCKYMEM_TEST_PG_DSN unset")
	}
	conformance.Suite(t, func(t *testing.T) store.Store {
		t.Helper()
		schema := fmt.Sprintf("rocky_t_%d", time.Now().UnixNano())
		st, err := postgres.OpenInSchema(dsn, schema)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = st.DropSchema(context.Background())
			_ = st.Close()
		})
		return st
	})
}
