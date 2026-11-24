package conformance

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/textnorm"
	"github.com/rocky-mem/rocky-mem/internal/uid"
)

// OpenFunc constructs a fresh empty Store for one test case. Caller closes it.
type OpenFunc func(t *testing.T) store.Store

// Suite runs the dialect-neutral contract against open.
func Suite(t *testing.T, open OpenFunc) {
	t.Helper()
	t.Run("LocalUser", func(t *testing.T) { testLocalUser(t, open) })
	t.Run("MemoryKeySupersede", func(t *testing.T) { testMemoryKeySupersede(t, open) })
	t.Run("MemorySupersedeRevConflict", func(t *testing.T) { testMemorySupersedeRevConflict(t, open) })
	t.Run("ExperienceSearchDoc", func(t *testing.T) { testExperienceSearchDoc(t, open) })
	t.Run("ExperienceDuplicate", func(t *testing.T) { testExperienceDuplicate(t, open) })
	t.Run("RawDedupByHash", func(t *testing.T) { testRawDedupByHash(t, open) })
	t.Run("FeedbackAndTrust", func(t *testing.T) { testFeedbackAndTrust(t, open) })
	t.Run("PurgeByUID", func(t *testing.T) { testPurgeByUID(t, open) })
	t.Run("ProjectRootCoalesce", func(t *testing.T) { testProjectRootCoalesce(t, open) })
	t.Run("DoctorSearchDocOK", func(t *testing.T) { testDoctorSearchDocOK(t, open) })
}

func testLocalUser(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	a, err := st.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || a != 1 {
		t.Fatalf("local user ids %d %d", a, b)
	}
}

func testMemoryKeySupersede(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	key := "pkg"
	m1, err := st.InsertMemory(ctx, store.Memory{
		UID: uid.Memory(), Lineage: uid.Memory(), Scope: "global", MemKey: &key,
		MemType: "pref", Title: "pkg", BodyMD: "pnpm", Lang: "und", Status: "active",
		Sensitivity: 2, ActorID: &actor, Rev: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "memory", OwnerID: m1.ID, Lang: "und", Title: m1.Title, Body: m1.BodyMD,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetActiveMemoryByKey(ctx, "global", "", key)
	if err != nil || got == nil || got.UID != m1.UID {
		t.Fatalf("get by key: %v %+v", err, got)
	}
	next := store.Memory{
		UID: uid.Memory(), Lineage: m1.Lineage, Scope: "global", MemKey: &key,
		MemType: "pref", Title: "pkg", BodyMD: "yarn", Lang: "und", Status: "active",
		Sensitivity: 2, ActorID: &actor, Rev: 1,
	}
	m2, err := st.SupersedeMemory(ctx, m1.ID, m1.Rev, next)
	if err != nil {
		t.Fatal(err)
	}
	active, _ := st.GetActiveMemoryByKey(ctx, "global", "", key)
	if active == nil || active.UID != m2.UID || active.BodyMD != "yarn" {
		t.Fatalf("supersede active: %+v", active)
	}
	old, _ := st.GetMemoryByUID(ctx, m1.UID)
	if old == nil || old.Status != "superseded" {
		t.Fatalf("old status: %+v", old)
	}
}

func testExperienceSearchDoc(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	e, err := st.InsertExperience(ctx, store.Experience{
		UID: uid.Experience(), ExpType: "pitfall", Scope: "global",
		Title: "ConfigMap lag", Symptom: "stale mount", FixMD: "checksum",
		SymptomKey: textnorm.SymptomKey("stale mount", "und"),
		FixHash: textnorm.Hash("checksum"), AppliesWhen: "k8s", Outcome: "success",
		Confidence: 0.9, Trust: "active", Lang: "und", Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "experience", OwnerID: e.ID, Lang: "und", Title: e.Title,
		Body: e.Title + "\n" + e.Symptom + "\n" + e.FixMD,
	}); err != nil {
		t.Fatal(err)
	}
	cands, err := st.SearchCandidates(ctx, "ConfigMap", "experience", 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range cands {
		if c.OwnerID == e.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("search miss: %+v", cands)
	}
}

func testExperienceDuplicate(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	title, symptom, fix := "dup", "same", "fix"
	e, err := st.InsertExperience(ctx, store.Experience{
		UID: uid.Experience(), ExpType: "howto", Scope: "global",
		Title: title, Symptom: symptom, FixMD: fix, Trust: "active",
		Outcome: "success", Confidence: 0.8, Lang: "und", Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	dup, err := st.FindExperienceDuplicate(ctx, title, symptom, fix)
	if err != nil || dup == nil || dup.UID != e.UID {
		t.Fatalf("dup: %v %+v", err, dup)
	}
}

func testMemorySupersedeRevConflict(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	key := "cas"
	lineage := uid.Memory()
	m1, err := st.InsertMemory(ctx, store.Memory{
		UID: lineage, Lineage: lineage, Scope: "global", MemKey: &key,
		MemType: "pref", Title: "cas", BodyMD: "v1", Lang: "und", Status: "active",
		Sensitivity: 2, ActorID: &actor, Rev: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	next := store.Memory{
		UID: uid.Memory(), Lineage: lineage, Scope: "global", MemKey: &key,
		MemType: "pref", Title: "cas", BodyMD: "v2", Lang: "und", Status: "active",
		Sensitivity: 2, ActorID: &actor, Rev: 1,
	}
	if _, err := st.SupersedeMemory(ctx, m1.ID, m1.Rev+99, next); err == nil {
		t.Fatal("expected rev conflict")
	}
	active, _ := st.GetActiveMemoryByKey(ctx, "global", "", key)
	if active == nil || active.UID != m1.UID {
		t.Fatalf("active changed on conflict: %+v", active)
	}
}

func testRawDedupByHash(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	body := []byte("identical evidence body")
	h := textnorm.Hash(string(body))
	r1, err := st.InsertRaw(ctx, store.Raw{
		UID: uid.Raw(), Hash: h, Channel: "agent", Body: body, Lang: "und",
		Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.GetRawByHash(ctx, h)
	if err != nil || got == nil || got.ID != r1.ID {
		t.Fatalf("get by hash: %v %+v", err, got)
	}
}

func testFeedbackAndTrust(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	e, err := st.InsertExperience(ctx, store.Experience{
		UID: uid.Experience(), ExpType: "pitfall", Scope: "global",
		Title: "fb", Symptom: "s", FixMD: "f", Trust: "active",
		Outcome: "success", Confidence: 0.9, Lang: "und", Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertFeedback(ctx, store.Feedback{
		OwnerType: "experience", OwnerID: e.ID, OwnerUID: e.UID,
		Outcome: "wrong", Note: "misapplied", ActorID: &actor,
	}); err != nil {
		t.Fatal(err)
	}
	n, err := st.CountFeedback(ctx, "experience", e.ID, "wrong", 0)
	if err != nil || n != 1 {
		t.Fatalf("count wrong=%d err=%v", n, err)
	}
	if err := st.UpdateExperienceTrust(ctx, e.ID, "contested"); err != nil {
		t.Fatal(err)
	}
	got, _ := st.GetExperienceByUID(ctx, e.UID)
	if got == nil || got.Trust != "contested" {
		t.Fatalf("trust: %+v", got)
	}
}

func testPurgeByUID(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	e, err := st.InsertExperience(ctx, store.Experience{
		UID: uid.Experience(), ExpType: "howto", Scope: "global",
		Title: "purge me", Trust: "active", Outcome: "success",
		Confidence: 0.8, Lang: "und", Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "experience", OwnerID: e.ID, Lang: "und", Title: e.Title, Body: e.Title,
	}); err != nil {
		t.Fatal(err)
	}
	res, err := st.Purge(ctx, store.PurgeFilter{UID: e.UID})
	if err != nil {
		t.Fatal(err)
	}
	if res.Experiences != 1 {
		t.Fatalf("purge: %+v", res)
	}
	gone, _ := st.GetExperienceByUID(ctx, e.UID)
	if gone != nil {
		t.Fatalf("still present: %+v", gone)
	}
}

func testProjectRootCoalesce(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "repo")
	a, err := st.UpsertProject(ctx, store.Project{
		Name: "repo", RemoteURLNorm: "local/aaaaaaaa", RootHint: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.UpsertProject(ctx, store.Project{
		Name: "org/repo", RemoteURLNorm: "github.com/org/repo", RootHint: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("expected coalesce %d != %d", a.ID, b.ID)
	}
	if b.RemoteURLNorm != "github.com/org/repo" {
		t.Fatalf("remote not upgraded: %s", b.RemoteURLNorm)
	}
}

func testDoctorSearchDocOK(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	m, err := st.InsertMemory(ctx, store.Memory{
		UID: uid.Memory(), Lineage: uid.Memory(), Scope: "global",
		MemType: "fact", Title: "t", BodyMD: "b", Lang: "und", Status: "active",
		Sensitivity: 2, ActorID: &actor, Rev: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "memory", OwnerID: m.ID, Lang: "und", Title: "t", Body: "b",
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := st.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.SearchDocOK {
		t.Fatalf("doctor: %+v", rep)
	}
}
