package core_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/store/sqlite"
)

func TestRememberIdempotentAndCAS(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	a, err := svc.Remember(ctx, core.RememberInput{
		Title: "pkg", Body: "use pnpm", MemKey: "pkg", MemType: "pref", Scope: "global",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Remember(ctx, core.RememberInput{
		Title: "pkg", Body: "use pnpm", MemKey: "pkg", MemType: "pref", Scope: "global",
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != "unchanged" || b.UID != a.UID || b.Rev != a.Rev {
		t.Fatalf("expected unchanged same rev: %+v vs %+v", a, b)
	}

	bad := 999
	_, err = svc.Remember(ctx, core.RememberInput{
		Title: "pkg", Body: "use yarn", MemKey: "pkg", MemType: "pref", Scope: "global", Rev: &bad,
	})
	if err == nil {
		t.Fatal("expected CAS conflict")
	}
	if e := errs.As(err); e.Code != errs.WriteConflict {
		t.Fatalf("want WRITE_CONFLICT got %v", e)
	}

	rev := a.Rev
	c, err := svc.Remember(ctx, core.RememberInput{
		Title: "pkg", Body: "use yarn", MemKey: "pkg", MemType: "pref", Scope: "global", Rev: &rev,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != "updated" || c.SupersededUID != a.UID {
		t.Fatalf("expected supersede: %+v", c)
	}
}

func TestRecordIdempotent(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	in := core.RecordInput{
		Title: "dup lesson", ExpType: "pitfall",
		Symptom: "same symptom", Fix: "same fix",
		AppliesWhen: "tests", Outcome: "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "evidence one"}},
	}
	a, err := svc.Record(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Record(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != "unchanged" || b.UID != a.UID {
		t.Fatalf("expected unchanged duplicate: %+v vs %+v", a, b)
	}
}

func TestReviewApproveReject(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	rec, err := svc.Record(ctx, core.RecordInput{
		Title: "needs review", ExpType: "pitfall", Symptom: "x", Fix: "y",
		AppliesWhen: "", Outcome: "success", Confidence: 0.4, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "ev"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Trust != "proposed" {
		t.Fatalf("expected proposed, got %s", rec.Trust)
	}
	list, err := svc.ReviewList(ctx, nil, 10)
	if err != nil || len(list) == 0 {
		t.Fatalf("review list: %v %#v", err, list)
	}
	ap, err := svc.ReviewApprove(ctx, rec.UID)
	if err != nil {
		t.Fatal(err)
	}
	if ap.Trust != "active" {
		t.Fatalf("%+v", ap)
	}

	rec2, err := svc.Record(ctx, core.RecordInput{
		Title: "reject me", ExpType: "pitfall", Symptom: "a", Fix: "b",
		AppliesWhen: "", Confidence: 0.3, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "ev2"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rj, err := svc.ReviewReject(ctx, rec2.UID, "noise")
	if err != nil {
		t.Fatal(err)
	}
	if rj.Trust != "archived" {
		t.Fatalf("%+v", rj)
	}
}

func TestReindexAndPurge(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	rec, err := svc.Record(ctx, core.RecordInput{
		Title: "purge target", ExpType: "howto", Symptom: "s", Fix: "f",
		AppliesWhen: "tests", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "ev"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ri, err := svc.Reindex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ri.ExperienceIndexed < 1 {
		t.Fatalf("%+v", ri)
	}
	ex, err := svc.ExportL1(ctx, "md")
	if err != nil || ex.Count < 1 {
		t.Fatalf("%v %+v", err, ex)
	}
	pr, err := svc.Purge(ctx, core.PurgeInput{UID: rec.UID})
	if err != nil {
		t.Fatal(err)
	}
	if pr.Experiences != 1 {
		t.Fatalf("%+v", pr)
	}
}

func testService(t *testing.T) *core.Service {
	t.Helper()
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		t.Fatal(err)
	}
	return &core.Service{Store: st}
}
