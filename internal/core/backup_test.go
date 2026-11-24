package core_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestBackupAndRestore(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "live.db")
	st, err := sqlite.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		t.Fatal(err)
	}
	svc := core.NewService(st)
	_, err = svc.Remember(ctx, core.RememberInput{
		Title: "pref", Body: "backup me", MemKey: "bk", Scope: "global", MemType: "pref",
	})
	if err != nil {
		t.Fatal(err)
	}
	snap := filepath.Join(dir, "snap.db")
	res, err := svc.Backup(ctx, snap)
	if err != nil {
		t.Fatal(err)
	}
	if res.Bytes < 1000 {
		t.Fatalf("tiny backup: %+v", res)
	}
	_ = st.Close()

	dest := filepath.Join(dir, "restored.db")
	if err := core.RestoreFile(snap, dest); err != nil {
		t.Fatal(err)
	}
	st2, err := sqlite.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := core.NewService(st2)
	pack, err := svc2.Search(ctx, core.SearchInput{Query: "backup me", IncludeProposed: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.UIDs) == 0 {
		t.Fatalf("restored search empty: %+v", pack)
	}
}
