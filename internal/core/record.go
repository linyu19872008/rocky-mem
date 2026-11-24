package core

import (
	"context"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/textnorm"
	"github.com/rocky-mem/rocky-mem/internal/uid"
)

// EvidenceIn is one evidence item on mem_record.
type EvidenceIn struct {
	Type string `json:"type"` // quote|commit|path|url|session
	Ref  string `json:"ref,omitempty"`
	Text string `json:"text,omitempty"`
}

// RecordInput matches mem_record.
type RecordInput struct {
	Title       string       `json:"title"`
	ExpType     string       `json:"exp_type"`
	Symptom     string       `json:"symptom,omitempty"`
	RootCause   string       `json:"root_cause,omitempty"`
	Fix         string       `json:"fix,omitempty"`
	AppliesWhen string       `json:"applies_when,omitempty"`
	InvalidWhen string       `json:"invalid_when,omitempty"`
	Outcome     string       `json:"outcome,omitempty"`
	Confidence  float64      `json:"confidence,omitempty"`
	Scope       string       `json:"scope,omitempty"`
	ScopeRef    string       `json:"scope_ref,omitempty"`
	Evidence    []EvidenceIn `json:"evidence"`
	Tags        []string     `json:"tags,omitempty"`
	Cwd         string       `json:"cwd,omitempty"`
	Lang        string       `json:"lang,omitempty"`
}

// RecordResult is the write acknowledgement.
type RecordResult struct {
	UID            string         `json:"uid"`
	Rev            int            `json:"rev"`
	Status         string         `json:"status"`
	Trust          string         `json:"trust"`
	ReviewRequired bool           `json:"review_required"`
	Conflict       map[string]any `json:"conflict,omitempty"`
	Warnings       []string       `json:"warnings,omitempty"`
	Reason         string         `json:"reason,omitempty"`
}

// Record writes an experience claim.
func (s *Service) Record(ctx context.Context, in RecordInput) (*RecordResult, error) {
	if strings.TrimSpace(in.Title) == "" || in.ExpType == "" {
		return nil, errs.New(errs.FieldRequired, "title and exp_type are required", nil)
	}
	if len(in.Evidence) < 1 {
		return nil, errs.New(errs.EvidenceRequired, "at least 1 evidence item is required for an experience entry", map[string]any{"min": 1})
	}
	if in.Outcome == "" {
		in.Outcome = "unknown"
	}
	if in.Confidence == 0 {
		in.Confidence = 0.5
	}
	if in.Lang == "" {
		in.Lang = "und"
	}

	proj, resolved, err := s.resolveProject(ctx, in.Cwd)
	if err != nil && in.Cwd != "" {
		// Soft: fall back to session rather than failing agent writes in non-git dirs.
		resolved = &ScopeResolved{Scope: "session", ScopeRef: "default"}
		err = nil
	}
	scope, ref := writeScope(in.Scope, in.ScopeRef, resolved)
	title := textnorm.NFC(strings.TrimSpace(in.Title))
	symptom := textnorm.NFC(in.Symptom)
	fix := textnorm.NFC(in.Fix)

	var result *RecordResult
	err = s.Store.WithTx(ctx, func(st store.Store) error {
		svc := &Service{Store: st}
		actor, err := st.EnsureLocalUser(ctx)
		if err != nil {
			return err
		}

		if dup, err := st.FindExperienceDuplicate(ctx, title, symptom, fix); err != nil {
			return err
		} else if dup != nil {
			result = &RecordResult{
				UID: dup.UID, Rev: dup.Rev, Status: "unchanged", Trust: dup.Trust,
				ReviewRequired: dup.Trust == "proposed",
			}
			return nil
		}

		var pid *int64
		if proj != nil {
			pid = &proj.ID
		}

		var warnings []string
		var rawIDs []int64
		for _, ev := range in.Evidence {
			srcRef := ev.Ref
			channel := "agent"
			switch ev.Type {
			case "quote":
				channel = "agent"
			case "commit":
				channel = "git"
			case "path":
				channel = "file"
			case "url":
				channel = "web"
			case "session":
				channel = "transcript"
			default:
				channel = "agent"
			}
			text, warn := resolveEvidenceText(ev.Type, ev.Ref, ev.Text, in.Cwd)
			if warn != "" {
				warnings = append(warnings, warn)
			}
			if strings.TrimSpace(text) == "" {
				warnings = append(warnings, "empty evidence skipped")
				continue
			}
			raw, err := svc.putRaw(ctx, text, channel, srcRef, in.Cwd, pid)
			if err != nil {
				if e, ok := err.(*errs.E); ok && e.Code == errs.PolicyRejected {
					return err
				}
				warnings = append(warnings, err.Error())
				continue
			}
			rawIDs = append(rawIDs, raw.ID)
		}
		if len(rawIDs) == 0 {
			return errs.New(errs.EvidenceRequired, "no evidence could be persisted", map[string]any{"min": 1})
		}

		e := store.Experience{
			UID: uid.Experience(), ExpType: in.ExpType, Scope: scope, ScopeRef: ref,
			Title: title, Symptom: symptom,
			SymptomKey: textnorm.SymptomKey(symptom, in.Lang),
			RootCause: textnorm.NFC(in.RootCause), FixMD: fix,
			FixHash: textnorm.Hash(fix), AppliesWhen: textnorm.NFC(in.AppliesWhen),
			InvalidWhen: textnorm.NFC(in.InvalidWhen), Outcome: in.Outcome,
			Confidence: in.Confidence, Lang: in.Lang, Sensitivity: 2, ActorID: &actor,
		}

		trust, status, reason, review := qualityGate(e, len(rawIDs))
		e.Trust = trust

		var conflict map[string]any
		if e.SymptomKey != "" {
			peers, err := st.FindExperiencesBySymptomKey(ctx, e.SymptomKey)
			if err != nil {
				return err
			}
			var conflicting []store.Experience
			for _, p := range peers {
				if p.FixHash != "" && e.FixHash != "" && p.FixHash != e.FixHash {
					conflicting = append(conflicting, p)
				}
			}
			if len(conflicting) > 0 {
				group := uid.Conflict()
				e.Trust = "contested"
				e.ConflictGroup = group
				trust = "contested"
				status = "published"
				review = false
				reason = "same symptom_key, different fix_hash"
				ids := make([]int64, 0, len(conflicting)+1)
				uids := make([]string, 0, len(conflicting)+1)
				for _, p := range conflicting {
					ids = append(ids, p.ID)
					uids = append(uids, p.UID)
				}
				// insert first then mark all including new
				inserted, err := st.InsertExperience(ctx, e)
				if err != nil {
					return err
				}
				e = inserted
				ids = append(ids, e.ID)
				uids = append(uids, e.UID)
				if err := st.MarkContested(ctx, ids, group); err != nil {
					return err
				}
				conflict = map[string]any{"group": group, "uids": uids, "reason": reason}
				goto afterInsert
			}
		}

		{
			inserted, err := st.InsertExperience(ctx, e)
			if err != nil {
				return err
			}
			e = inserted
		}

	afterInsert:
		for _, rid := range rawIDs {
			if _, err := st.InsertEvidence(ctx, store.Evidence{
				OwnerType: "experience", OwnerID: e.ID, RawID: rid, Role: "support",
			}); err != nil {
				return err
			}
		}
		if err := st.WriteSearchDoc(ctx, store.SearchDoc{
			Kind: "experience", OwnerID: e.ID, Lang: e.Lang, Title: e.Title, Body: experienceSearchBody(e),
		}); err != nil {
			return err
		}
		if len(in.Tags) > 0 {
			_ = st.SetTags(ctx, "experience", e.ID, in.Tags)
		}
		_ = st.InsertAudit(ctx, "write", "experience", &e.ID, e.UID, status)

		result = &RecordResult{
			UID: e.UID, Rev: e.Rev, Status: status, Trust: trust,
			ReviewRequired: review, Conflict: conflict, Warnings: warnings, Reason: reason,
		}
		return nil
	})
	return result, err
}
