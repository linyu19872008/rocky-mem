package core_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestRecordSearchRoundTrip(t *testing.T) {
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

	_, err = svc.Remember(ctx, core.RememberInput{
		Title: "package manager", Body: "Use pnpm. Never npm.",
		MemKey: "package_manager", MemType: "pref", Scope: "global", Pinned: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	rec, err := svc.Record(ctx, core.RecordInput{
		Title:       "ConfigMap hot reload lost during rollout",
		ExpType:     "pitfall",
		Symptom:     "config reverts after kubectl rollout restart",
		RootCause:   "file-mounted ConfigMap update lag ~60s",
		Fix:         "add checksum annotation on Pod template",
		AppliesWhen: "K8s + file-mounted ConfigMap + short rollout window",
		InvalidWhen: "config read via ConfigMap API",
		Outcome:     "success",
		Confidence:  0.85,
		Scope:       "global",
		Evidence:    []core.EvidenceIn{{Type: "quote", Text: "kubectl rollout restart still lost ConfigMap mount"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Trust != "active" {
		t.Fatalf("expected active, got %s (%s)", rec.Trust, rec.Reason)
	}

	pack, err := svc.Search(ctx, core.SearchInput{
		Query: "ConfigMap rollout", IncludeProposed: true, BudgetTokens: 1200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pack.Pack == "" || len(pack.UIDs) == 0 {
		t.Fatalf("empty pack: %+v", pack)
	}
	if !contains(pack.Pack, "Applies") && !contains(pack.Pack, "Preferences") {
		t.Fatalf("pack missing sections: %s", pack.Pack)
	}

	doc, err := st.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.SearchDocOK {
		t.Fatalf("search_doc drift: %+v", doc)
	}
	if doc.SchemaVersion != "4" {
		t.Fatalf("expected schema 4, got %s", doc.SchemaVersion)
	}

	got, err := svc.Get(ctx, core.GetInput{UID: rec.UID, IncludeRaw: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "experience" || len(got.Evidence) == 0 {
		t.Fatalf("mem_get incomplete: %+v", got)
	}

	fb, err := svc.Feedback(ctx, core.FeedbackInput{
		UID: rec.UID, Outcome: "wrong", Note: "ConfigMap was API-read, advice misapplied",
	})
	if err != nil {
		t.Fatal(err)
	}
	if fb.Trust != "contested" {
		t.Fatalf("expected contested after wrong, got %+v", fb)
	}

	got2, err := svc.Get(ctx, core.GetInput{UID: rec.UID})
	if err != nil {
		t.Fatal(err)
	}
	if got2.Entry["trust"] != "contested" {
		t.Fatalf("trust not updated: %+v", got2.Entry)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
