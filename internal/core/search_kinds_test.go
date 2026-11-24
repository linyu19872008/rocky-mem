package core_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestSearchChunkNotImplemented(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_, _ = st.EnsureLocalUser(ctx)
	svc := core.NewService(st)
	_, err = svc.Search(ctx, core.SearchInput{Query: "x", Kinds: []string{"chunk"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.As(err).Code != errs.NotImplemented {
		t.Fatalf("got %v", err)
	}
}
