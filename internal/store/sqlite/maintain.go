package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/rocky-mem/rocky-mem/internal/store"
)

func (db *DB) ListExperiencesByTrust(ctx context.Context, trusts []string, limit int) ([]store.Experience, error) {
	if len(trusts) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	ph := make([]string, len(trusts))
	args := make([]any, 0, len(trusts)+1)
	for i, t := range trusts {
		ph[i] = "?"
		args = append(args, t)
	}
	args = append(args, limit)
	q := experienceSelect + ` WHERE trust IN (` + strings.Join(ph, ",") + `) ORDER BY t_created DESC LIMIT ?`
	rows, err := db.q().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExperienceRows(rows)
}

func (db *DB) CountExperiencesByTrust(ctx context.Context, trusts []string) (map[string]int, error) {
	out := map[string]int{}
	if len(trusts) == 0 {
		return out, nil
	}
	ph := make([]string, len(trusts))
	args := make([]any, len(trusts))
	for i, t := range trusts {
		ph[i] = "?"
		args[i] = t
		out[t] = 0
	}
	q := `SELECT trust, COUNT(*) FROM experience WHERE trust IN (` + strings.Join(ph, ",") + `) GROUP BY trust`
	rows, err := db.q().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var trust string
		var n int
		if err := rows.Scan(&trust, &n); err != nil {
			return nil, err
		}
		out[trust] = n
	}
	return out, rows.Err()
}

func (db *DB) ListAllActiveMemory(ctx context.Context) ([]store.Memory, error) {
	rows, err := db.q().QueryContext(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE status='active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRows(rows)
}

func (db *DB) ListIndexableExperiences(ctx context.Context) ([]store.Experience, error) {
	rows, err := db.q().QueryContext(ctx, experienceSelect+` WHERE trust <> 'archived'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExperienceRows(rows)
}

func (db *DB) FindExperienceDuplicate(ctx context.Context, title, symptom, fix string) (*store.Experience, error) {
	return db.scanExperience(ctx, experienceSelect+`
		WHERE title=? AND ifnull(symptom,'')=? AND ifnull(fix_md,'')=? AND trust <> 'archived'
		ORDER BY t_created DESC LIMIT 1`, title, symptom, fix)
}

func (db *DB) ClearSearchDocs(ctx context.Context) (int, error) {
	var n int
	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM search_doc`).Scan(&n)
	if _, err := db.q().ExecContext(ctx, `DELETE FROM search_doc_fts`); err != nil {
		return 0, err
	}
	if _, err := db.q().ExecContext(ctx, `DELETE FROM search_doc`); err != nil {
		return 0, err
	}
	return n, nil
}

func (db *DB) Purge(ctx context.Context, f store.PurgeFilter) (store.PurgeResult, error) {
	var out store.PurgeResult
	if f.UID != "" {
		return db.purgeUID(ctx, f.UID)
	}
	if f.Before > 0 {
		memIDs, err := db.ids(ctx, `SELECT id FROM memory WHERE created_at < ?`, f.Before)
		if err != nil {
			return out, err
		}
		expIDs, err := db.ids(ctx, `SELECT id FROM experience WHERE t_created < ?`, f.Before)
		if err != nil {
			return out, err
		}
		return db.purgeIDs(ctx, memIDs, expIDs)
	}
	if f.ProjectScopeRef != "" {
		memIDs, err := db.ids(ctx, `SELECT id FROM memory WHERE scope='project' AND scope_ref=?`, f.ProjectScopeRef)
		if err != nil {
			return out, err
		}
		expIDs, err := db.ids(ctx, `SELECT id FROM experience WHERE scope='project' AND scope_ref=?`, f.ProjectScopeRef)
		if err != nil {
			return out, err
		}
		return db.purgeIDs(ctx, memIDs, expIDs)
	}
	return out, fmt.Errorf("empty purge filter")
}

func (db *DB) purgeUID(ctx context.Context, uid string) (store.PurgeResult, error) {
	if strings.HasPrefix(uid, "m_") {
		m, err := db.GetMemoryByUID(ctx, uid)
		if err != nil || m == nil {
			if m == nil && err == nil {
				return store.PurgeResult{}, fmt.Errorf("memory not found")
			}
			return store.PurgeResult{}, err
		}
		return db.purgeIDs(ctx, []int64{m.ID}, nil)
	}
	if strings.HasPrefix(uid, "e_") {
		e, err := db.GetExperienceByUID(ctx, uid)
		if err != nil || e == nil {
			if e == nil && err == nil {
				return store.PurgeResult{}, fmt.Errorf("experience not found")
			}
			return store.PurgeResult{}, err
		}
		return db.purgeIDs(ctx, nil, []int64{e.ID})
	}
	return store.PurgeResult{}, fmt.Errorf("uid must start with m_ or e_")
}

func (db *DB) ids(ctx context.Context, q string, args ...any) ([]int64, error) {
	rows, err := db.q().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (db *DB) purgeIDs(ctx context.Context, memIDs, expIDs []int64) (store.PurgeResult, error) {
	var out store.PurgeResult
	for _, id := range memIDs {
		n, err := db.deleteOwner(ctx, "memory", id)
		if err != nil {
			return out, err
		}
		out.Evidence += n
		if err := db.DeleteSearchDoc(ctx, "memory", id); err != nil {
			return out, err
		}
		out.SearchDocs++
		if _, err := db.q().ExecContext(ctx, `DELETE FROM memory WHERE id=?`, id); err != nil {
			return out, err
		}
		out.Memories++
	}
	for _, id := range expIDs {
		n, err := db.deleteOwner(ctx, "experience", id)
		if err != nil {
			return out, err
		}
		out.Evidence += n
		if err := db.DeleteSearchDoc(ctx, "experience", id); err != nil {
			return out, err
		}
		out.SearchDocs++
		if _, err := db.q().ExecContext(ctx, `DELETE FROM experience WHERE id=?`, id); err != nil {
			return out, err
		}
		out.Experiences++
	}
	return out, nil
}

func (db *DB) deleteOwner(ctx context.Context, ownerType string, ownerID int64) (int, error) {
	_, _ = db.q().ExecContext(ctx, `DELETE FROM tag WHERE owner_type=? AND owner_id=?`, ownerType, ownerID)
	res, err := db.q().ExecContext(ctx, `DELETE FROM evidence WHERE owner_type=? AND owner_id=?`, ownerType, ownerID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
