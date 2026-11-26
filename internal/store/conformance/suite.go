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
	t.Run("CorpusSourceChunksSearch", func(t *testing.T) { testCorpusSourceChunksSearch(t, open) })
	t.Run("CorpusArchiveHidesChunks", func(t *testing.T) { testCorpusArchiveHidesChunks(t, open) })
	t.Run("CorpusDeleteCascades", func(t *testing.T) { testCorpusDeleteCascades(t, open) })
	t.Run("ConflictResolutionAppendOnly", func(t *testing.T) { testConflictResolutionAppendOnly(t, open) })
	t.Run("LinkUpsertAndLookup", func(t *testing.T) { testLinkUpsertAndLookup(t, open) })
	t.Run("ConditionsUpdateRewritesIndex", func(t *testing.T) { testConditionsUpdateRewritesIndex(t, open) })
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
		FixHash:    textnorm.Hash("checksum"), AppliesWhen: "k8s", Outcome: "success",
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

func seedSource(t *testing.T, st store.Store, actor int64, hash string) store.Source {
	t.Helper()
	src, err := st.InsertSource(context.Background(), store.Source{
		UID: uid.Source(), Kind: "file", URI: "rollout.md", Title: "Rollout notes",
		Format: "plain", TextHash: hash, CharCount: 120, ChunkCount: 2,
		Lang: "en", Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func testCorpusSourceChunksSearch(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	src := seedSource(t, st, actor, "hash-rollout")

	got, err := st.GetSourceByTextHash(ctx, "hash-rollout")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != src.ID || got.Status != "active" {
		t.Fatalf("lookup by hash: %+v", got)
	}

	chunks, err := st.InsertChunks(ctx, []store.Chunk{
		{UID: uid.Chunk(), SourceID: src.ID, Ord: 0, Heading: "1",
			Text: "checksum annotation forces pod recreation", CharCount: 44, Sensitivity: 2},
		{UID: uid.Chunk(), SourceID: src.ID, Ord: 1,
			Text: "mounted configmap lags sixty seconds", CharCount: 37, Sensitivity: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 || chunks[0].ID == 0 || chunks[1].ID == 0 {
		t.Fatalf("inserted chunks: %+v", chunks)
	}

	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "chunk", OwnerID: chunks[0].ID, Lang: "en",
		Title: "Rollout notes", Body: chunks[0].Text,
	}); err != nil {
		t.Fatal(err)
	}

	cands, err := st.SearchCandidates(ctx, "checksum annotation", "chunk", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) == 0 || cands[0].Kind != "chunk" || cands[0].OwnerID != chunks[0].ID {
		t.Fatalf("chunk search: %+v", cands)
	}

	byUID, err := st.GetChunkByUID(ctx, chunks[1].UID)
	if err != nil || byUID == nil || byUID.Ord != 1 {
		t.Fatalf("get chunk by uid: %+v %v", byUID, err)
	}

	stats, err := st.CorpusStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sources != 1 || stats.ActiveSources != 1 || stats.Chunks != 2 || stats.ChunkDocs != 1 {
		t.Fatalf("stats: %+v", stats)
	}
}

func testCorpusArchiveHidesChunks(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	src := seedSource(t, st, actor, "hash-archive")
	chunks, err := st.InsertChunks(ctx, []store.Chunk{
		{UID: uid.Chunk(), SourceID: src.ID, Ord: 0, Text: "alpha beta", CharCount: 10, Sensitivity: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := []int64{chunks[0].ID}
	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "chunk", OwnerID: chunks[0].ID, Lang: "en", Title: src.Title, Body: chunks[0].Text,
	}); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.SearchCandidates(ctx, "alpha beta", "chunk", 5); len(got) == 0 {
		t.Fatal("chunk should be searchable before archiving")
	}

	active, err := st.ListActiveChunksByIDs(ctx, ids)
	if err != nil || len(active) != 1 {
		t.Fatalf("before archive: %d %v", len(active), err)
	}
	if err := st.ArchiveSource(ctx, src.ID, src.ID, 0); err != nil {
		t.Fatal(err)
	}
	active, err = st.ListActiveChunksByIDs(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("archived source still returns %d chunks", len(active))
	}
	if got, _ := st.SearchCandidates(ctx, "alpha beta", "chunk", 5); len(got) != 0 {
		t.Fatalf("archived chunk still indexed: %+v", got)
	}
	all, err := st.ListChunksBySource(ctx, src.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("chunks should survive archiving: %d %v", len(all), err)
	}
}

func testCorpusDeleteCascades(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	src := seedSource(t, st, actor, "hash-delete")
	chunks, err := st.InsertChunks(ctx, []store.Chunk{
		{UID: uid.Chunk(), SourceID: src.ID, Ord: 0, Text: "gamma delta", CharCount: 11, Sensitivity: 2},
		{UID: uid.Chunk(), SourceID: src.ID, Ord: 1, Text: "epsilon zeta", CharCount: 12, Sensitivity: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chunks {
		if err := st.WriteSearchDoc(ctx, store.SearchDoc{
			Kind: "chunk", OwnerID: c.ID, Lang: "en", Title: src.Title, Body: c.Text,
		}); err != nil {
			t.Fatal(err)
		}
	}

	n, err := st.DeleteSource(ctx, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("deleted chunk count %d, want 2", n)
	}
	gone, _ := st.GetSourceByUID(ctx, src.UID)
	if gone != nil {
		t.Fatalf("source still present: %+v", gone)
	}
	stats, err := st.CorpusStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sources != 0 || stats.Chunks != 0 || stats.ChunkDocs != 0 {
		t.Fatalf("stats after delete: %+v", stats)
	}
}

func seedExperience(t *testing.T, st store.Store, actor int64, title string) store.Experience {
	t.Helper()
	e, err := st.InsertExperience(context.Background(), store.Experience{
		UID: uid.Experience(), ExpType: "pitfall", Scope: "global",
		Title: title, Symptom: "seed symptom", FixMD: "seed fix", Trust: "contested",
		SymptomKey:  textnorm.SymptomKey("seed symptom", "und"),
		FixHash:     textnorm.Hash("seed fix " + title),
		AppliesWhen: "seed", Outcome: "unknown", Confidence: 0.8,
		Lang: "und", Sensitivity: 2, ActorID: &actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func testConflictResolutionAppendOnly(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	a := seedExperience(t, st, actor, "Resolution A")

	first, err := st.InsertResolution(ctx, store.ConflictResolution{
		GroupID: "cg_seed", Action: "defer", Note: "need logs", ActorID: &actor, CreatedAt: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 {
		t.Fatal("resolution id not assigned")
	}
	second, err := st.InsertResolution(ctx, store.ConflictResolution{
		GroupID: "cg_seed", Action: "adopt", Note: "A holds", KeptID: &a.ID, ActorID: &actor, CreatedAt: 200,
	})
	if err != nil {
		t.Fatal(err)
	}

	latest, err := st.LatestResolution(ctx, "cg_seed")
	if err != nil || latest == nil {
		t.Fatalf("latest: %+v %v", latest, err)
	}
	if latest.ID != second.ID || latest.Action != "adopt" || latest.KeptID == nil || *latest.KeptID != a.ID {
		t.Fatalf("latest resolution: %+v", latest)
	}
	all, err := st.ListResolutions(ctx, "cg_seed")
	if err != nil || len(all) != 2 {
		t.Fatalf("history: %d %v", len(all), err)
	}
	if all[0].Action != "defer" || all[1].Action != "adopt" {
		t.Fatalf("history order: %+v", all)
	}
}

func testLinkUpsertAndLookup(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	a := seedExperience(t, st, actor, "Link A")
	b := seedExperience(t, st, actor, "Link B")

	link := store.Link{
		SrcType: "experience", SrcID: a.ID,
		DstType: "experience", DstID: b.ID,
		Relation: "complements", Note: "different conditions",
	}
	if err := st.InsertLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	// Same tuple twice must stay one row (unique index + DO NOTHING).
	if err := st.InsertLink(ctx, link); err != nil {
		t.Fatal(err)
	}

	fromA, err := st.ListLinks(ctx, "experience", a.ID)
	if err != nil || len(fromA) != 1 || fromA[0].Relation != "complements" {
		t.Fatalf("from A: %+v %v", fromA, err)
	}
	// Lookup is symmetric: the peer sees the same relation.
	fromB, err := st.ListLinks(ctx, "experience", b.ID)
	if err != nil || len(fromB) != 1 || fromB[0].SrcID != a.ID {
		t.Fatalf("from B: %+v %v", fromB, err)
	}
}

func testConditionsUpdateRewritesIndex(t *testing.T, open OpenFunc) {
	st := open(t)
	defer st.Close()
	ctx := context.Background()
	actor, _ := st.EnsureLocalUser(ctx)
	e := seedExperience(t, st, actor, "Condition A")
	if err := st.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "experience", OwnerID: e.ID, Lang: "und", Title: e.Title, Body: "seed symptom",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateExperienceConditions(ctx, e.ID, "file mounted", "api read"); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetExperienceByUID(ctx, e.UID)
	if err != nil || got == nil {
		t.Fatalf("get: %+v %v", got, err)
	}
	if got.AppliesWhen != "file mounted" || got.InvalidWhen != "api read" {
		t.Fatalf("conditions: %+v", got)
	}
	if got.Rev < 2 {
		t.Fatalf("rev not bumped: %d", got.Rev)
	}
}
