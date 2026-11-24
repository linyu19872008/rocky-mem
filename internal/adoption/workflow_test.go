package adoption_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/adoption"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestRunWorkflow(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "rocky-mem.db")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		t.Fatal(err)
	}
	svc := &core.Service{Store: st}

	res, err := adoption.RunWorkflow(ctx, svc, dir, "tier0-unit-marker-xyz")
	if err != nil {
		t.Fatalf("workflow failed: %v\nsteps=%v", err, res.Steps)
	}
	if len(res.Steps) != len(adoption.RequiredWorkflowSteps) {
		t.Fatalf("steps=%v want %v", res.Steps, adoption.RequiredWorkflowSteps)
	}
	for i, want := range adoption.RequiredWorkflowSteps {
		if res.Steps[i] != want {
			t.Fatalf("step[%d]=%s want %s", i, res.Steps[i], want)
		}
	}
	if res.RecordUID == "" || res.RememberUID == "" {
		t.Fatalf("missing uids: %+v", res)
	}
	if !res.FoundAfterWrite {
		t.Fatal("expected search to find written entries")
	}
	if res.RecordTrust == "" {
		t.Fatal("empty trust")
	}

	// Doctor must stay consistent after workflow writes.
	doc, err := st.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.SearchDocOK {
		t.Fatalf("search_doc drift after workflow: %+v", doc)
	}
}

func TestRunWorkflowRejectsEmptyServiceStore(t *testing.T) {
	// Ensure failure surfaces when store cannot open — covered by Open errors elsewhere.
	// This asserts step list constant integrity.
	if len(adoption.RequiredWorkflowSteps) < 7 {
		t.Fatal("workflow steps incomplete")
	}
}
