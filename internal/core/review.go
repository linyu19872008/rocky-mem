package core

import (
	"context"
	"strings"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/store"
)

// ReviewItem is a proposed/contested experience awaiting human action.
type ReviewItem struct {
	UID           string  `json:"uid"`
	Title         string  `json:"title"`
	Trust         string  `json:"trust"`
	ExpType       string  `json:"exp_type"`
	Confidence    float64 `json:"confidence"`
	Symptom       string  `json:"symptom,omitempty"`
	AppliesWhen   string  `json:"applies_when"`
	ConflictGroup string  `json:"conflict_group,omitempty"`
	TCreated      int64   `json:"t_created"`
}

// ReviewList returns experiences in the given trusts (default proposed+contested).
func (s *Service) ReviewList(ctx context.Context, trusts []string, limit int) ([]ReviewItem, error) {
	if len(trusts) == 0 {
		trusts = []string{"proposed", "contested"}
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.Store.ListExperiencesByTrust(ctx, trusts, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ReviewItem, 0, len(rows))
	for _, e := range rows {
		out = append(out, ReviewItem{
			UID: e.UID, Title: e.Title, Trust: e.Trust, ExpType: e.ExpType,
			Confidence: e.Confidence, Symptom: e.Symptom, AppliesWhen: e.AppliesWhen,
			ConflictGroup: e.ConflictGroup, TCreated: e.TCreated,
		})
	}
	return out, nil
}

// ReviewResult is the outcome of approve/reject.
type ReviewResult struct {
	UID    string `json:"uid"`
	Trust  string `json:"trust"`
	Status string `json:"status"`
}

// ReviewApprove promotes proposed/contested → active.
func (s *Service) ReviewApprove(ctx context.Context, uid string) (*ReviewResult, error) {
	uid = strings.TrimSpace(uid)
	e, err := s.Store.GetExperienceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errs.New(errs.NotFound, "experience not found", map[string]any{"uid": uid})
	}
	if e.Trust != "proposed" && e.Trust != "contested" {
		return nil, errs.New(errs.FieldInvalid, "only proposed or contested entries can be approved", map[string]any{
			"trust": e.Trust,
		})
	}
	if err := s.Store.UpdateExperienceTrust(ctx, e.ID, "active"); err != nil {
		return nil, err
	}
	// Ensure indexed.
	_ = s.Store.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "experience", OwnerID: e.ID, Lang: e.Lang, Title: e.Title, Body: experienceSearchBody(*e),
	})
	_ = s.Store.InsertAudit(ctx, "verify", "experience", &e.ID, e.UID, "review:approve")
	return &ReviewResult{UID: e.UID, Trust: "active", Status: "approved"}, nil
}

// ReviewReject archives an entry (human rejection).
func (s *Service) ReviewReject(ctx context.Context, uid, reason string) (*ReviewResult, error) {
	uid = strings.TrimSpace(uid)
	e, err := s.Store.GetExperienceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errs.New(errs.NotFound, "experience not found", map[string]any{"uid": uid})
	}
	if err := s.Store.UpdateExperienceTrust(ctx, e.ID, "archived"); err != nil {
		return nil, err
	}
	// Archiving removes the entry from retrieval; leaving the search doc behind
	// makes `doctor` report drift and keeps feeding dead candidates.
	_ = s.Store.DeleteSearchDoc(ctx, "experience", e.ID)
	detail := "review:reject"
	if reason != "" {
		detail += ":" + reason
	}
	_ = s.Store.InsertAudit(ctx, "reject", "experience", &e.ID, e.UID, detail)
	return &ReviewResult{UID: e.UID, Trust: "archived", Status: "rejected"}, nil
}

// ReviewResolveInput is one human adjudication of a conflict group.
//
// Actions:
//
//	adopt        keep one entry (default: the one addressed), falsify the rest
//	split        both hold, under different conditions; both become active
//	reject_both  neither holds; both become falsified
//	defer        still unknown; stays contested, revisited after ReviewAfterDays
//	reopen       a previous ruling is questioned again; members return to contested
type ReviewResolveInput struct {
	UID    string `json:"uid"`
	Action string `json:"action"`
	Note   string `json:"note"`

	KeepUID string `json:"keep,omitempty"`
	PeerUID string `json:"peer,omitempty"`

	KeepAppliesWhen string `json:"keep_applies_when,omitempty"`
	KeepInvalidWhen string `json:"keep_invalid_when,omitempty"`
	PeerAppliesWhen string `json:"peer_applies_when,omitempty"`
	PeerInvalidWhen string `json:"peer_invalid_when,omitempty"`

	ReviewAfterDays int `json:"review_after_days,omitempty"`
}

// ReviewResolveResult reports what the ruling did.
type ReviewResolveResult struct {
	Action       string   `json:"action"`
	Group        string   `json:"group,omitempty"`
	Kept         string   `json:"kept,omitempty"`
	Active       []string `json:"active,omitempty"`
	Falsified    []string `json:"falsified,omitempty"`
	Contested    []string `json:"contested,omitempty"`
	ResolutionID int64    `json:"resolution_id"`
	ReviewAfter  int64    `json:"review_after,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// ReviewResolve applies a human ruling to a conflict group. Rulings are
// append-only: every call records a conflict_resolution row, so a group can be
// re-opened and its history stays readable.
func (s *Service) ReviewResolve(ctx context.Context, in ReviewResolveInput) (*ReviewResolveResult, error) {
	uid := strings.TrimSpace(in.UID)
	action := strings.ToLower(strings.TrimSpace(in.Action))
	note := strings.TrimSpace(in.Note)
	if uid == "" {
		return nil, errs.New(errs.FieldRequired, "uid required", map[string]any{"field": "uid"})
	}
	switch action {
	case "adopt", "split", "reject_both", "defer", "reopen":
	default:
		return nil, errs.New(errs.FieldInvalid, "action must be adopt, split, reject_both, defer or reopen",
			map[string]any{"action": in.Action})
	}
	if note == "" {
		return nil, errs.New(errs.FieldRequired, "note is required: a ruling without a reason is a guess",
			map[string]any{"field": "note"})
	}

	e, err := s.Store.GetExperienceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errs.New(errs.NotFound, "experience not found", map[string]any{"uid": uid})
	}
	if e.ConflictGroup == "" {
		return nil, errs.New(errs.FieldInvalid, "entry is not part of a conflict group",
			map[string]any{"uid": uid})
	}
	members, err := s.Store.FindExperiencesByConflictGroup(ctx, e.ConflictGroup)
	if err != nil {
		return nil, err
	}
	if len(members) < 2 {
		return nil, errs.New(errs.FieldInvalid, "conflict group has fewer than two members",
			map[string]any{"group": e.ConflictGroup})
	}

	out := &ReviewResolveResult{Action: action, Group: e.ConflictGroup}
	now := time.Now().Unix()
	var reviewAfter *int64

	switch action {
	case "adopt":
		keepUID := firstNonEmpty(strings.TrimSpace(in.KeepUID), uid)
		keep, ok := findMember(members, keepUID)
		if !ok {
			return nil, errs.New(errs.FieldInvalid, "keep uid is not a member of the group",
				map[string]any{"keep": keepUID, "group": e.ConflictGroup})
		}
		if err := s.setTrust(ctx, *keep, "active", &out.Active); err != nil {
			return nil, err
		}
		for i := range members {
			m := members[i]
			if m.UID == keep.UID {
				continue
			}
			if err := s.setTrust(ctx, m, "falsified", &out.Falsified); err != nil {
				return nil, err
			}
		}
		out.Kept = keep.UID
		if ev, err := s.Store.ListEvidence(ctx, "experience", keep.ID); err == nil && len(ev) == 0 {
			out.Warnings = append(out.Warnings,
				"kept entry has no evidence; the ruling itself is now the only support for it")
		}

	case "split":
		if len(members) != 2 {
			return nil, errs.New(errs.FieldInvalid,
				"split needs exactly two entries; groups with more must be resolved pairwise first",
				map[string]any{"group": e.ConflictGroup, "members": len(members)})
		}
		other := members[0]
		if other.UID == uid {
			other = members[1]
		}
		if peer := strings.TrimSpace(in.PeerUID); peer != "" {
			m, ok := findMember(members, peer)
			if !ok {
				return nil, errs.New(errs.FieldInvalid, "peer uid is not a member of the group",
					map[string]any{"peer": peer})
			}
			other = *m
		}
		self, ok := findMember(members, uid)
		if !ok {
			return nil, errs.New(errs.NotFound, "experience not found", map[string]any{"uid": uid})
		}

		selfApplies := firstNonEmpty(strings.TrimSpace(in.KeepAppliesWhen), self.AppliesWhen)
		selfInvalid := firstNonEmpty(strings.TrimSpace(in.KeepInvalidWhen), self.InvalidWhen)
		peerApplies := firstNonEmpty(strings.TrimSpace(in.PeerAppliesWhen), other.AppliesWhen)
		peerInvalid := firstNonEmpty(strings.TrimSpace(in.PeerInvalidWhen), other.InvalidWhen)

		missing := []string{}
		if selfApplies == "" {
			missing = append(missing, "keep_applies_when")
		}
		if selfInvalid == "" {
			missing = append(missing, "keep_invalid_when")
		}
		if peerApplies == "" {
			missing = append(missing, "peer_applies_when")
		}
		if peerInvalid == "" {
			missing = append(missing, "peer_invalid_when")
		}
		if len(missing) > 0 {
			return nil, errs.New(errs.FieldInvalid,
				"split requires each entry to say when it applies AND when it does not; fill: "+strings.Join(missing, ", "),
				map[string]any{"missing": missing})
		}
		if strings.EqualFold(selfApplies, peerApplies) {
			return nil, errs.New(errs.FieldInvalid,
				"the two conditions are identical — this is not a split, it is the same claim",
				map[string]any{"applies_when": selfApplies})
		}

		if err := s.applyConditions(ctx, *self, selfApplies, selfInvalid); err != nil {
			return nil, err
		}
		if err := s.applyConditions(ctx, other, peerApplies, peerInvalid); err != nil {
			return nil, err
		}
		if err := s.setTrust(ctx, *self, "active", &out.Active); err != nil {
			return nil, err
		}
		if err := s.setTrust(ctx, other, "active", &out.Active); err != nil {
			return nil, err
		}
		// Record that they are complementary, in both directions.
		for _, pair := range [][2]store.Experience{{*self, other}, {other, *self}} {
			if err := s.Store.InsertLink(ctx, store.Link{
				SrcType: "experience", SrcID: pair[0].ID,
				DstType: "experience", DstID: pair[1].ID,
				Relation: "complements", Note: note,
			}); err != nil {
				return nil, err
			}
		}

	case "reject_both":
		for i := range members {
			if err := s.setTrust(ctx, members[i], "falsified", &out.Falsified); err != nil {
				return nil, err
			}
		}

	case "defer":
		if in.ReviewAfterDays <= 0 {
			return nil, errs.New(errs.FieldInvalid, "review_after_days must be positive",
				map[string]any{"review_after_days": in.ReviewAfterDays})
		}
		when := now + int64(in.ReviewAfterDays)*86400
		reviewAfter = &when
		out.ReviewAfter = when
		for i := range members {
			out.Contested = append(out.Contested, members[i].UID)
		}

	case "reopen":
		for i := range members {
			if err := s.setTrust(ctx, members[i], "contested", &out.Contested); err != nil {
				return nil, err
			}
		}
	}

	actor, _ := s.Store.EnsureLocalUser(ctx)
	var keptID *int64
	if out.Kept != "" {
		if m, ok := findMember(members, out.Kept); ok {
			keptID = &m.ID
		}
	}
	rec, err := s.Store.InsertResolution(ctx, store.ConflictResolution{
		GroupID: out.Group, Action: action, KeptID: keptID, Note: note,
		ActorID: &actor, ReviewAfter: reviewAfter, CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	out.ResolutionID = rec.ID
	_ = s.Store.InsertAudit(ctx, "resolve", "experience", &e.ID, e.UID,
		"conflict:"+action+":"+out.Group+":"+note)
	return out, nil
}

// setTrust moves one entry and keeps the derived index in sync.
func (s *Service) setTrust(ctx context.Context, e store.Experience, trust string, bucket *[]string) error {
	if err := s.Store.UpdateExperienceTrust(ctx, e.ID, trust); err != nil {
		return err
	}
	switch trust {
	case "archived", "falsified":
		if err := s.Store.DeleteSearchDoc(ctx, "experience", e.ID); err != nil {
			return err
		}
	default:
		e.Trust = trust
		if err := s.Store.WriteSearchDoc(ctx, store.SearchDoc{
			Kind: "experience", OwnerID: e.ID, Lang: e.Lang, Title: e.Title,
			Body: experienceSearchBody(e),
		}); err != nil {
			return err
		}
	}
	if bucket != nil {
		*bucket = append(*bucket, e.UID)
	}
	return nil
}

func (s *Service) applyConditions(ctx context.Context, e store.Experience, appliesWhen, invalidWhen string) error {
	if err := s.Store.UpdateExperienceConditions(ctx, e.ID, appliesWhen, invalidWhen); err != nil {
		return err
	}
	e.AppliesWhen = appliesWhen
	e.InvalidWhen = invalidWhen
	return s.Store.WriteSearchDoc(ctx, store.SearchDoc{
		Kind: "experience", OwnerID: e.ID, Lang: e.Lang, Title: e.Title,
		Body: experienceSearchBody(e),
	})
}

func findMember(members []store.Experience, uid string) (*store.Experience, bool) {
	for i := range members {
		if members[i].UID == uid {
			return &members[i], true
		}
	}
	return nil, false
}
