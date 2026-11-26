package core_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/store/sqlite"
)

func TestBrowseListsMemoryAndExperience(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		t.Fatal(err)
	}
	svc := core.NewService(st)

	mem, err := svc.Remember(ctx, core.RememberInput{
		Title: "Use pnpm", Body: "Never npm or yarn in this repo.", MemKey: "pref.package_manager",
	})
	if err != nil {
		t.Fatal(err)
	}
	exp, err := svc.Record(ctx, core.RecordInput{
		Title: "ConfigMap lag", ExpType: "pitfall", Symptom: "config not updating",
		Fix: "annotate checksum", AppliesWhen: "k8s rollout", Outcome: "success",
		Confidence: 0.8, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "checksum annotation fixed it"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if exp.Trust == "proposed" || exp.Trust == "contested" {
		if _, err := svc.ReviewApprove(ctx, exp.UID); err != nil {
			t.Fatal(err)
		}
	}

	all, err := svc.Browse(ctx, core.BrowseInput{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if all.Total < 2 {
		t.Fatalf("want >=2 items, got %+v", all)
	}
	var sawMem, sawExp bool
	for _, it := range all.Items {
		if it.UID == mem.UID {
			sawMem = true
			if it.Kind != "memory" {
				t.Fatalf("kind %s", it.Kind)
			}
		}
		if it.UID == exp.UID {
			sawExp = true
			if it.Kind != "experience" || it.Trust != "active" {
				t.Fatalf("%+v", it)
			}
		}
	}
	if !sawMem || !sawExp {
		t.Fatalf("missing entries: mem=%v exp=%v items=%+v", sawMem, sawExp, all.Items)
	}

	onlyMem, err := svc.Browse(ctx, core.BrowseInput{Kinds: []string{"memory"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range onlyMem.Items {
		if it.Kind != "memory" {
			t.Fatalf("unexpected %s", it.Kind)
		}
	}

	filtered, err := svc.Browse(ctx, core.BrowseInput{Q: "pnpm"})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total < 1 {
		t.Fatal("expected pnpm hit")
	}
}
