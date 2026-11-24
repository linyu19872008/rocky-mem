package core

import (
	"context"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/errs"
	"github.com/rocky-mem/rocky-mem/internal/store"
)

// ReviewItem is a proposed/contested experience awaiting human action.
type ReviewItem struct {
	UID         string  `json:"uid"`
	Title       string  `json:"title"`
	Trust       string  `json:"trust"`
	ExpType     string  `json:"exp_type"`
	Confidence  float64 `json:"confidence"`
	AppliesWhen string  `json:"applies_when"`
	TCreated    int64   `json:"t_created"`
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
			Confidence: e.Confidence, AppliesWhen: e.AppliesWhen, TCreated: e.TCreated,
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
	detail := "review:reject"
	if reason != "" {
		detail += ":" + reason
	}
	_ = s.Store.InsertAudit(ctx, "reject", "experience", &e.ID, e.UID, detail)
	return &ReviewResult{UID: e.UID, Trust: "archived", Status: "rejected"}, nil
}
