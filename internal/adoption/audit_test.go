package adoption_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/adoption"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestAnalyzeAuditAndFinalize(t *testing.T) {
	rows := []store.AuditRow{
		{TS: 100, Action: "read", Detail: "mem_context"},
		{TS: 101, Action: "read", Detail: "mem_search:boom"},
		{TS: 102, Action: "write", TargetType: "experience", Detail: "published"},
		{TS: 103, Action: "write", TargetType: "memory", Detail: ""},
		{TS: 104, Action: "write", TargetType: "experience", Detail: "feedback:helpful"},
	}
	v := adoption.AnalyzeAudit(rows, 100)
	if v.ReadCount != 2 || v.WriteCount != 3 {
		t.Fatalf("counts: %+v", v)
	}
	if v.SearchMentions != 1 || v.ContextMentions != 1 || v.RecordMentions != 1 || v.FeedbackMentions != 1 {
		t.Fatalf("mentions: %+v", v)
	}
	final := adoption.FinalizeVerdict(v, true, true)
	if final.Status != "pass" {
		t.Fatalf("status=%s notes=%v", final.Status, final.Notes)
	}
	fail := adoption.FinalizeVerdict(v, false, true)
	if fail.Status != "fail" {
		t.Fatalf("expected fail when protocol bad, got %s", fail.Status)
	}
}

func TestLoadAndAnalyzeAfterWorkflow(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "rocky-mem.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	_, _ = st.EnsureLocalUser(ctx)
	svc := &core.Service{Store: st}
	since := time.Now().Unix() - 1
	if _, err := adoption.RunWorkflow(ctx, svc, dir, "audit-marker-abc"); err != nil {
		t.Fatal(err)
	}
	root := repoRoot(t)
	protocolOK := adoption.CheckProtocolFiles(root) == nil
	v, err := adoption.LoadAndAnalyze(ctx, st, since, protocolOK, true)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != "pass" {
		t.Fatalf("verdict=%+v", v)
	}
	if v.ReadCount < 1 || v.WriteCount < 1 {
		t.Fatalf("expected audit activity: %+v", v)
	}
	md := adoption.RenderVerdictMarkdown(v)
	for _, want := range []string{"Tier-0 adoption verdict", "status:", "Live agent checklist"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestListAuditEmpty(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "rocky-mem.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rows, err := st.ListAudit(context.Background(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("want empty, got %d", len(rows))
	}
}
