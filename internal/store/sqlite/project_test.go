package sqlite_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/store"
	"github.com/linyu19872008/rocky-mem/internal/store/sqlite"
)

func TestUpsertProjectMergesLocalAndRemoteSameRoot(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	root := "/tmp/demo-repo"

	local, err := st.UpsertProject(ctx, store.Project{
		Name: "demo-repo", RemoteURLNorm: "local/aaaaaaaa", RootHint: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.InsertExperience(ctx, store.Experience{
		UID: "e_test_merge", ExpType: "pitfall", Scope: "project",
		ScopeRef: strconv.FormatInt(local.ID, 10), Title: "seed lesson", Trust: "active",
		Lang: "und", Confidence: 0.9, Outcome: "success", Sensitivity: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate the historical bug path: a second row already created for the remote.
	dup, err := st.UpsertProject(ctx, store.Project{
		Name: "other", RemoteURLNorm: "local/bbbbbbbb", RootHint: "/tmp/unrelated",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = dup
	// Force-insert collision: remote identity without going through upgrade (raw SQL via second local then).
	// Instead: create remote on same root by first inserting a divergent remote row, then merging.
	// Direct path: upgrade in place when only local exists.
	remote, err := st.UpsertProject(ctx, store.Project{
		Name: "org/demo-repo", RemoteURLNorm: "github.com/org/demo-repo", RootHint: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if remote.ID != local.ID {
		t.Fatalf("expected in-place upgrade id=%d got %d", local.ID, remote.ID)
	}
	if remote.RemoteURLNorm != "github.com/org/demo-repo" {
		t.Fatalf("remote not upgraded: %s", remote.RemoteURLNorm)
	}

	exps, err := st.ListExperiencesByTrust(ctx, []string{"active"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) != 1 || exps[0].ScopeRef != strconv.FormatInt(remote.ID, 10) {
		t.Fatalf("experience scope not preserved: %+v", exps)
	}
}

func TestUpsertProjectMergesOrphanDuplicate(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	root := "/tmp/same-checkout"

	a, err := st.UpsertProject(ctx, store.Project{
		Name: "a", RemoteURLNorm: "local/11111111", RootHint: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.InsertExperience(ctx, store.Experience{
		UID: "e_orphan", ExpType: "howto", Scope: "project",
		ScopeRef: strconv.FormatInt(a.ID, 10), Title: "orphan", Trust: "active",
		Lang: "und", Confidence: 0.8, Outcome: "success", Sensitivity: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Insert remote as a NEW row by temporarily using a different root, then fix root_hint
	// to simulate the pre-fix double-insert bug.
	b, err := st.UpsertProject(ctx, store.Project{
		Name: "org/repo", RemoteURLNorm: "github.com/org/repo", RootHint: root + "-tmp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == a.ID {
		t.Fatal("expected distinct ids for setup")
	}
	// Point both at the same root (the bug state), then upsert remote again to trigger merge.
	db := st // Store interface — use Upsert which merges when root matches keep remote
	// Re-upsert with correct root: finds by remote (b), then merges orphans with same root.
	// First align b's root_hint via upsert.
	kept, err := db.UpsertProject(ctx, store.Project{
		Name: "org/repo", RemoteURLNorm: "github.com/org/repo", RootHint: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if kept.ID != b.ID {
		t.Fatalf("keep id=%d want %d", kept.ID, b.ID)
	}

	exps, err := st.ListExperiencesByTrust(ctx, []string{"active"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) != 1 {
		t.Fatalf("got %d exps", len(exps))
	}
	if exps[0].ScopeRef != strconv.FormatInt(kept.ID, 10) {
		t.Fatalf("scope_ref=%s want %d", exps[0].ScopeRef, kept.ID)
	}
}
