package postgres

import (
	"context"
	"fmt"

	"github.com/rocky-mem/rocky-mem/internal/store"
)

func (db *DB) GetActiveMemoryByLineage(ctx context.Context, lineage string) (*store.Memory, error) {
	return db.scanMemory(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE lineage=$1 AND status='active'
		ORDER BY created_at DESC LIMIT 1`, lineage)
}

func (db *DB) ListMemoryVersions(ctx context.Context, lineage string) ([]store.Memory, error) {
	rows, err := db.q().QueryContext(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE lineage=$1 ORDER BY created_at ASC`, lineage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRows(rows)
}

func (db *DB) FindExperiencesByConflictGroup(ctx context.Context, group string) ([]store.Experience, error) {
	if group == "" {
		return nil, nil
	}
	rows, err := db.q().QueryContext(ctx, experienceSelect+` WHERE conflict_group=$1`, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExperienceRows(rows)
}

func (db *DB) UpdateExperienceTrust(ctx context.Context, id int64, trust string) error {
	res, err := db.q().ExecContext(ctx, `UPDATE experience SET trust=$1, rev=rev+1 WHERE id=$2`, trust, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("experience %d not found", id)
	}
	if trust == "archived" || trust == "falsified" {
		return db.DeleteSearchDoc(ctx, "experience", id)
	}
	return nil
}

func (db *DB) ListTags(ctx context.Context, ownerType string, ownerID int64) ([]string, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT tag FROM tag WHERE owner_type=$1 AND owner_id=$2 ORDER BY tag`, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (db *DB) InsertFeedback(ctx context.Context, f store.Feedback) (store.Feedback, error) {
	if f.CreatedAt == 0 {
		f.CreatedAt = now()
	}
	err := db.q().QueryRowContext(ctx, `
		INSERT INTO feedback(owner_type, owner_id, owner_uid, outcome, note, actor_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		f.OwnerType, f.OwnerID, f.OwnerUID, f.Outcome, nullStr(f.Note), nullInt64(f.ActorID), f.CreatedAt,
	).Scan(&f.ID)
	if err != nil {
		return store.Feedback{}, err
	}
	return f, nil
}

func (db *DB) CountFeedback(ctx context.Context, ownerType string, ownerID int64, outcome string, sinceUnix int64) (int, error) {
	var n int
	err := db.q().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM feedback
		WHERE owner_type=$1 AND owner_id=$2 AND outcome=$3 AND created_at>=$4`,
		ownerType, ownerID, outcome, sinceUnix,
	).Scan(&n)
	return n, err
}
