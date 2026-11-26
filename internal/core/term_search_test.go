package core_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/store/sqlite"
)

func TestTermMapExpandsChineseQuery(t *testing.T) {
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
	_, err = svc.Record(ctx, core.RecordInput{
		Title:       "ConfigMap hot reload lost during rollout",
		ExpType:     "pitfall",
		Symptom:     "config reverts after kubectl rollout restart",
		Fix:         "checksum annotation",
		AppliesWhen: "K8s file-mounted ConfigMap",
		InvalidWhen: "API read",
		Outcome:     "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "observed stale mount"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Chinese query should expand via 滚动更新 / 配置 → ConfigMap / rolling
	pack, err := svc.Search(ctx, core.SearchInput{
		Query: "滚动更新后配置丢失", IncludeProposed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.UIDs) == 0 || !strings.Contains(pack.Pack, "ConfigMap") {
		t.Fatalf("term expansion miss: %+v", pack)
	}
}

func TestShortCJKLikeFallback(t *testing.T) {
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
	_, err = svc.Record(ctx, core.RecordInput{
		Title:       "探针配置不当导致 CrashLoop",
		ExpType:     "pitfall",
		Symptom:     "liveness 探针过早",
		Fix:         "startupProbe",
		AppliesWhen: "K8s JVM",
		Outcome:     "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "CrashLoop from probe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	pack, err := svc.Search(ctx, core.SearchInput{Query: "探针", IncludeProposed: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.UIDs) == 0 {
		t.Fatalf("2-char CJK LIKE miss: %+v", pack)
	}
}
