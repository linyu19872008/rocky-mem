package core_test

import (
	"context"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/errs"
)

func contestedPair(t *testing.T, svc *core.Service) (*core.RecordResult, *core.RecordResult, int64) {
	t.Helper()
	ctx := context.Background()
	a, err := svc.Record(ctx, core.RecordInput{
		Title: "Side A", ExpType: "pitfall", Symptom: "same symptom", Fix: "fix alpha",
		AppliesWhen: "file-mounted config", InvalidWhen: "api-read config",
		Outcome: "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "a-evidence"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Record(ctx, core.RecordInput{
		Title: "Side B", ExpType: "pitfall", Symptom: "same symptom", Fix: "fix beta",
		AppliesWhen: "api-read config", InvalidWhen: "file-mounted config",
		Outcome: "success", Confidence: 0.8, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "b-evidence"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only the second write knows it collided; the first was already stored.
	if b.Conflict == nil {
		t.Fatalf("expected auto conflict on second write, got a=%+v b=%+v", a.Conflict, b.Conflict)
	}
	group, _ := b.Conflict["group"].(string)
	if group == "" {
		t.Fatalf("no conflict group: %+v", b.Conflict)
	}
	return a, b, 0
}

func TestResolveRequiresNoteAndGroup(t *testing.T) {
	_, svc := openSvc(t)
	ctx := context.Background()
	a, _, _ := contestedPair(t, svc)

	if _, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "adopt",
	}); err == nil || errs.As(err).Code != errs.FieldRequired {
		t.Fatalf("expected note required, got %v", err)
	}

	solo, err := svc.Record(ctx, core.RecordInput{
		Title: "Lonely", ExpType: "pitfall", Symptom: "unrelated", Fix: "x",
		AppliesWhen: "y", Outcome: "success", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "lonely"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: solo.UID, Action: "adopt", Note: "n",
	}); err == nil {
		t.Fatal("expected error for entry outside a conflict group")
	}
}

func TestResolveAdoptFalsifiesPeers(t *testing.T) {
	st, svc := openSvc(t)
	ctx := context.Background()
	a, b, _ := contestedPair(t, svc)

	out, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "adopt", Note: "A is the file-mounted case we actually had",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Kept != a.UID || len(out.Falsified) != 1 || out.Falsified[0] != b.UID {
		t.Fatalf("result: %+v", out)
	}
	ea, _ := st.GetExperienceByUID(ctx, a.UID)
	eb, _ := st.GetExperienceByUID(ctx, b.UID)
	if ea.Trust != "active" || eb.Trust != "falsified" {
		t.Fatalf("trusts: %s / %s", ea.Trust, eb.Trust)
	}
	if got, _ := st.SearchCandidates(ctx, "fix beta", "experience", 5); len(got) != 0 {
		t.Fatalf("falsified peer still indexed: %+v", got)
	}
	// The ruling is recorded, not just applied.
	res, err := st.LatestResolution(ctx, out.Group)
	if err != nil || res == nil || res.Action != "adopt" {
		t.Fatalf("resolution: %+v %v", res, err)
	}
}

func TestResolveSplitKeepsBothActive(t *testing.T) {
	st, svc := openSvc(t)
	ctx := context.Background()
	a, b, _ := contestedPair(t, svc)

	out, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "split",
		Note: "A holds for file mounts, B for api-read configs",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Active) != 2 {
		t.Fatalf("expected both active: %+v", out)
	}
	ea, _ := st.GetExperienceByUID(ctx, a.UID)
	eb, _ := st.GetExperienceByUID(ctx, b.UID)
	if ea.Trust != "active" || eb.Trust != "active" {
		t.Fatalf("trusts: %s / %s", ea.Trust, eb.Trust)
	}
	links, err := st.ListLinks(ctx, "experience", ea.ID)
	if err != nil || len(links) == 0 {
		t.Fatalf("expected complements link: %+v %v", links, err)
	}
	if links[0].Relation != "complements" {
		t.Fatalf("relation: %+v", links[0])
	}
}

func TestResolveSplitRejectsEmptyOrIdenticalConditions(t *testing.T) {
	_, svc := openSvc(t)
	ctx := context.Background()
	a, b, _ := contestedPairNoConditions(t, svc)

	// Splitting without conditions must be refused: "depends" with no
	// conditions is worse than leaving the pair contested.
	_, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "split", Note: "n",
		KeepAppliesWhen: "", KeepInvalidWhen: "",
		PeerAppliesWhen: "", PeerInvalidWhen: "",
	})
	if err == nil || errs.As(err).Code != errs.FieldInvalid {
		t.Fatalf("expected field invalid, got %v", err)
	}

	_, err = svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "split", Note: "n",
		KeepAppliesWhen: "same", KeepInvalidWhen: "x",
		PeerAppliesWhen: "same", PeerInvalidWhen: "y",
	})
	if err == nil {
		t.Fatalf("expected identical conditions rejected, b=%s", b.UID)
	}
}

func TestResolveRejectBothDeferReopen(t *testing.T) {
	st, svc := openSvc(t)
	ctx := context.Background()

	a, b, _ := contestedPair(t, svc)
	out, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "defer", Note: "need prod logs", ReviewAfterDays: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ReviewAfter == 0 || len(out.Contested) != 2 {
		t.Fatalf("defer: %+v", out)
	}
	ea, _ := st.GetExperienceByUID(ctx, a.UID)
	if ea.Trust != "contested" {
		t.Fatalf("defer changed trust: %s", ea.Trust)
	}

	if _, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "reject_both", Note: "both were guesswork",
	}); err != nil {
		t.Fatal(err)
	}
	ea, _ = st.GetExperienceByUID(ctx, a.UID)
	eb, _ := st.GetExperienceByUID(ctx, b.UID)
	if ea.Trust != "falsified" || eb.Trust != "falsified" {
		t.Fatalf("reject_both: %s / %s", ea.Trust, eb.Trust)
	}

	// A later report can bring the pair back into question.
	if _, err := svc.ReviewResolve(ctx, core.ReviewResolveInput{
		UID: a.UID, Action: "reopen", Note: "new evidence contradicts the ruling",
	}); err != nil {
		t.Fatal(err)
	}
	ea, _ = st.GetExperienceByUID(ctx, a.UID)
	if ea.Trust != "contested" {
		t.Fatalf("reopen: %s", ea.Trust)
	}
}

func contestedPairNoConditions(t *testing.T, svc *core.Service) (*core.RecordResult, *core.RecordResult, int64) {
	t.Helper()
	ctx := context.Background()
	a, err := svc.Record(ctx, core.RecordInput{
		Title: "Bare A", ExpType: "pitfall", Symptom: "bare symptom", Fix: "fix one",
		Outcome: "unknown", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "bare-a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Record(ctx, core.RecordInput{
		Title: "Bare B", ExpType: "pitfall", Symptom: "bare symptom", Fix: "fix two",
		Outcome: "unknown", Confidence: 0.9, Scope: "global",
		Evidence: []core.EvidenceIn{{Type: "quote", Text: "bare-b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Conflict == nil {
		t.Fatalf("expected conflict: %+v", b)
	}
	return a, b, 0
}
