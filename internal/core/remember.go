package core

import (
	"context"
	"strings"

	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/store"
	"github.com/linyu19872008/rocky-mem/internal/textnorm"
	"github.com/linyu19872008/rocky-mem/internal/uid"
)

// RememberInput matches mem_remember.
type RememberInput struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	MemKey      string   `json:"mem_key,omitempty"`
	MemType     string   `json:"mem_type,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	ScopeRef    string   `json:"scope_ref,omitempty"`
	Pinned      bool     `json:"pinned,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Rev         *int     `json:"rev,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SourceQuote string   `json:"source_quote,omitempty"`
	Cwd         string   `json:"cwd,omitempty"`
}

// RememberResult is the write acknowledgement.
type RememberResult struct {
	UID           string `json:"uid"`
	Lineage       string `json:"lineage"`
	Status        string `json:"status"`
	Rev           int    `json:"rev"`
	SupersededUID string `json:"superseded_uid,omitempty"`
	ReviewRequired bool  `json:"review_required"`
}

// Remember upserts a memory row.
func (s *Service) Remember(ctx context.Context, in RememberInput) (*RememberResult, error) {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Body) == "" {
		return nil, errs.New(errs.FieldRequired, "title and body are required", nil)
	}
	if in.MemType == "" {
		in.MemType = "fact"
	}
	proj, resolved, err := s.resolveProject(ctx, in.Cwd)
	if err != nil && in.Cwd != "" {
		return nil, err
	}
	scope, ref := writeScope(in.Scope, in.ScopeRef, resolved)
	title := textnorm.NFC(strings.TrimSpace(in.Title))
	body := textnorm.NFC(strings.TrimSpace(in.Body))

	var result *RememberResult
	err = s.Store.WithTx(ctx, func(st store.Store) error {
		svc := &Service{Store: st}
		actor, err := st.EnsureLocalUser(ctx)
		if err != nil {
			return err
		}

		var supersededUID string
		status := "created"
		mUID := uid.Memory()
		lineage := mUID
		rev := 1

		if in.MemKey != "" {
			cur, err := st.GetActiveMemoryByKey(ctx, scope, ref, in.MemKey)
			if err != nil {
				return err
			}
			if cur != nil {
				if textnorm.Hash(cur.Title+"\n"+cur.BodyMD) == textnorm.Hash(title+"\n"+body) &&
					cur.Pinned == in.Pinned && cur.Priority == in.Priority {
					result = &RememberResult{
						UID: cur.UID, Lineage: cur.Lineage, Status: "unchanged", Rev: cur.Rev,
					}
					return nil
				}
				if in.Rev != nil && *in.Rev != cur.Rev {
					return errs.New(errs.WriteConflict, "memory revision mismatch", map[string]any{
						"expected_rev": *in.Rev, "actual_rev": cur.Rev,
					})
				}
				expected := cur.Rev
				if in.Rev != nil {
					expected = *in.Rev
				}
				next := store.Memory{
					UID: mUID, Lineage: cur.Lineage, Scope: scope, ScopeRef: ref,
					MemKey: cur.MemKey, MemType: in.MemType, Title: title, BodyMD: body,
					Lang: "und", Pinned: in.Pinned, Priority: in.Priority, Status: "active",
					Sensitivity: 2, ActorID: &actor, Rev: 1,
				}
				newM, err := st.SupersedeMemory(ctx, cur.ID, expected, next)
				if err != nil {
					return errs.New(errs.WriteConflict, err.Error(), nil)
				}
				if err := st.WriteSearchDoc(ctx, store.SearchDoc{
					Kind: "memory", OwnerID: newM.ID, Lang: newM.Lang,
					Title: newM.Title, Body: memorySearchBody(newM),
				}); err != nil {
					return err
				}
				if len(in.Tags) > 0 {
					_ = st.SetTags(ctx, "memory", newM.ID, in.Tags)
				}
				if in.SourceQuote != "" {
					var pid *int64
					if proj != nil {
						pid = &proj.ID
					}
					raw, err := svc.putRaw(ctx, in.SourceQuote, "manual", "", in.Cwd, pid)
					if err == nil {
						_, _ = st.InsertEvidence(ctx, store.Evidence{
							OwnerType: "memory", OwnerID: newM.ID, RawID: raw.ID, Role: "support",
						})
					}
				}
				_ = st.InsertAudit(ctx, "update", "memory", &newM.ID, newM.UID, "supersede")
				result = &RememberResult{
					UID: newM.UID, Lineage: newM.Lineage, Status: "updated",
					Rev: newM.Rev, SupersededUID: cur.UID,
				}
				return nil
			}
		}

		var keyPtr *string
		if in.MemKey != "" {
			k := in.MemKey
			keyPtr = &k
		}
		m := store.Memory{
			UID: mUID, Lineage: lineage, Scope: scope, ScopeRef: ref,
			MemKey: keyPtr, MemType: in.MemType, Title: title, BodyMD: body,
			Lang: "und", Pinned: in.Pinned, Priority: in.Priority, Status: "active",
			Sensitivity: 2, ActorID: &actor, Rev: rev,
		}
		m, err = st.InsertMemory(ctx, m)
		if err != nil {
			return err
		}
		if err := st.WriteSearchDoc(ctx, store.SearchDoc{
			Kind: "memory", OwnerID: m.ID, Lang: m.Lang, Title: m.Title, Body: memorySearchBody(m),
		}); err != nil {
			return err
		}
		if len(in.Tags) > 0 {
			_ = st.SetTags(ctx, "memory", m.ID, in.Tags)
		}
		if in.SourceQuote != "" {
			var pid *int64
			if proj != nil {
				pid = &proj.ID
			}
			raw, err := svc.putRaw(ctx, in.SourceQuote, "manual", "", in.Cwd, pid)
			if err == nil {
				_, _ = st.InsertEvidence(ctx, store.Evidence{
					OwnerType: "memory", OwnerID: m.ID, RawID: raw.ID, Role: "support",
				})
			}
		}
		_ = st.InsertAudit(ctx, "write", "memory", &m.ID, m.UID, "")
		_ = supersededUID
		result = &RememberResult{UID: m.UID, Lineage: m.Lineage, Status: status, Rev: m.Rev}
		return nil
	})
	return result, err
}
