package core

import (
	"context"
	"strings"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/store"
)

// FeedbackInput matches mem_feedback.
type FeedbackInput struct {
	UID     string `json:"uid"`
	Outcome string `json:"outcome"` // helpful | wrong | stale
	Note    string `json:"note,omitempty"`
}

// FeedbackResult is the acknowledgement.
type FeedbackResult struct {
	UID       string `json:"uid"`
	Type      string `json:"type"`
	Outcome   string `json:"outcome"`
	Trust     string `json:"trust,omitempty"`
	UseCount  int    `json:"use_count,omitempty"`
	Effect    string `json:"effect"`
	StaleHits int    `json:"stale_hits,omitempty"`
}

// Feedback applies a helpful / wrong / stale signal.
func (s *Service) Feedback(ctx context.Context, in FeedbackInput) (*FeedbackResult, error) {
	uid := strings.TrimSpace(in.UID)
	outcome := strings.TrimSpace(in.Outcome)
	if uid == "" || outcome == "" {
		return nil, errs.New(errs.FieldRequired, "uid and outcome are required", nil)
	}
	switch outcome {
	case "helpful", "wrong", "stale":
	default:
		return nil, errs.New(errs.FieldInvalid, "outcome must be helpful, wrong, or stale", map[string]any{"field": "outcome"})
	}
	if outcome == "wrong" && strings.TrimSpace(in.Note) == "" {
		return nil, errs.New(errs.FieldRequired, "note is required when outcome=wrong", map[string]any{"field": "note"})
	}

	actor, _ := s.Store.EnsureLocalUser(ctx)

	if strings.HasPrefix(uid, "e_") {
		return s.feedbackExperience(ctx, uid, outcome, in.Note, actor)
	}
	if strings.HasPrefix(uid, "m_") {
		return s.feedbackMemory(ctx, uid, outcome, in.Note, actor)
	}
	return nil, errs.New(errs.FieldInvalid, "uid must start with m_ or e_", map[string]any{"field": "uid"})
}

func (s *Service) feedbackExperience(ctx context.Context, uid, outcome, note string, actor int64) (*FeedbackResult, error) {
	e, err := s.Store.GetExperienceByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errs.New(errs.NotFound, "experience not found", map[string]any{"uid": uid})
	}

	var result *FeedbackResult
	err = s.Store.WithTx(ctx, func(st store.Store) error {
		_, err := st.InsertFeedback(ctx, store.Feedback{
			OwnerType: "experience", OwnerID: e.ID, OwnerUID: e.UID,
			Outcome: outcome, Note: note, ActorID: &actor,
		})
		if err != nil {
			return err
		}
		trust := e.Trust
		effect := ""
		staleHits := 0
		switch outcome {
		case "helpful":
			if err := st.BumpUse(ctx, "experience", e.ID, time.Now().Unix()); err != nil {
				return err
			}
			effect = "bumped use_count"
			updated, _ := st.GetExperience(ctx, e.ID)
			use := e.UseCount + 1
			if updated != nil {
				use = updated.UseCount
				trust = updated.Trust
			}
			result = &FeedbackResult{
				UID: e.UID, Type: "experience", Outcome: outcome,
				Trust: trust, UseCount: use, Effect: effect,
			}
			_ = st.InsertAudit(ctx, "write", "experience", &e.ID, e.UID, "feedback:helpful")
			return nil
		case "wrong":
			if trust == "active" || trust == "proposed" {
				if err := st.UpdateExperienceTrust(ctx, e.ID, "contested"); err != nil {
					return err
				}
				trust = "contested"
				// Refresh search_doc so ranking sees contested trust via re-fetch on search;
				// body unchanged, but we rewrite to keep doctor happy.
				if err := st.WriteSearchDoc(ctx, store.SearchDoc{
					Kind: "experience", OwnerID: e.ID, Lang: e.Lang,
					Title: e.Title, Body: experienceSearchBody(*e),
				}); err != nil {
					return err
				}
			}
			effect = "demoted to contested"
			result = &FeedbackResult{
				UID: e.UID, Type: "experience", Outcome: outcome,
				Trust: trust, Effect: effect,
			}
			_ = st.InsertAudit(ctx, "write", "experience", &e.ID, e.UID, "feedback:wrong")
			return nil
		case "stale":
			since := time.Now().Add(-90 * 24 * time.Hour).Unix()
			n, err := st.CountFeedback(ctx, "experience", e.ID, "stale", since)
			if err != nil {
				return err
			}
			staleHits = n
			effect = "stale flag recorded"
			if n >= 3 && trust != "stale" && trust != "archived" && trust != "falsified" {
				if err := st.UpdateExperienceTrust(ctx, e.ID, "stale"); err != nil {
					return err
				}
				trust = "stale"
				effect = "trust moved to stale after 3 reports in 90d"
				_ = st.WriteSearchDoc(ctx, store.SearchDoc{
					Kind: "experience", OwnerID: e.ID, Lang: e.Lang,
					Title: e.Title, Body: experienceSearchBody(*e),
				})
			}
			result = &FeedbackResult{
				UID: e.UID, Type: "experience", Outcome: outcome,
				Trust: trust, Effect: effect, StaleHits: staleHits,
			}
			_ = st.InsertAudit(ctx, "write", "experience", &e.ID, e.UID, "feedback:stale")
			return nil
		}
		return nil
	})
	return result, err
}

func (s *Service) feedbackMemory(ctx context.Context, uid, outcome, note string, actor int64) (*FeedbackResult, error) {
	m, err := s.Store.GetMemoryByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errs.New(errs.NotFound, "memory not found", map[string]any{"uid": uid})
	}
	if m.Status != "active" {
		active, err := s.Store.GetActiveMemoryByLineage(ctx, m.Lineage)
		if err != nil {
			return nil, err
		}
		if active != nil {
			m = active
		}
	}

	var result *FeedbackResult
	err = s.Store.WithTx(ctx, func(st store.Store) error {
		_, err := st.InsertFeedback(ctx, store.Feedback{
			OwnerType: "memory", OwnerID: m.ID, OwnerUID: m.UID,
			Outcome: outcome, Note: note, ActorID: &actor,
		})
		if err != nil {
			return err
		}
		effect := "recorded"
		use := m.UseCount
		if outcome == "helpful" {
			if err := st.BumpUse(ctx, "memory", m.ID, time.Now().Unix()); err != nil {
				return err
			}
			use++
			effect = "bumped use_count"
		} else if outcome == "wrong" {
			effect = "noted; memory stays active until a human edits or archives it"
		} else {
			effect = "stale flag recorded on memory"
		}
		result = &FeedbackResult{
			UID: m.UID, Type: "memory", Outcome: outcome,
			UseCount: use, Effect: effect,
		}
		_ = st.InsertAudit(ctx, "write", "memory", &m.ID, m.UID, "feedback:"+outcome)
		return nil
	})
	return result, err
}
