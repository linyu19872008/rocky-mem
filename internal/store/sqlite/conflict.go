package sqlite

import (
	"context"
	"database/sql"

	"github.com/linyu19872008/rocky-mem/internal/store"
)

func (db *DB) UpdateExperienceConditions(ctx context.Context, id int64, appliesWhen, invalidWhen string) error {
	_, err := db.q().ExecContext(ctx,
		`UPDATE experience SET applies_when=?, invalid_when=?, rev=rev+1 WHERE id=?`,
		nullStr(appliesWhen), nullStr(invalidWhen), id)
	return err
}

func (db *DB) InsertResolution(ctx context.Context, r store.ConflictResolution) (store.ConflictResolution, error) {
	if r.CreatedAt == 0 {
		r.CreatedAt = now()
	}
	res, err := db.q().ExecContext(ctx,
		`INSERT INTO conflict_resolution(group_id, action, kept_id, note, actor_id, review_after, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		r.GroupID, r.Action, r.KeptID, r.Note, r.ActorID, r.ReviewAfter, r.CreatedAt)
	if err != nil {
		return store.ConflictResolution{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.ConflictResolution{}, err
	}
	r.ID = id
	return r, nil
}

const resolutionCols = `id, group_id, action, kept_id, note, actor_id, review_after, created_at`

func scanResolution(sc interface{ Scan(...any) error }) (store.ConflictResolution, error) {
	var r store.ConflictResolution
	var kept, actor, after sql.NullInt64
	if err := sc.Scan(&r.ID, &r.GroupID, &r.Action, &kept, &r.Note, &actor, &after, &r.CreatedAt); err != nil {
		return store.ConflictResolution{}, err
	}
	if kept.Valid {
		v := kept.Int64
		r.KeptID = &v
	}
	if actor.Valid {
		v := actor.Int64
		r.ActorID = &v
	}
	if after.Valid {
		v := after.Int64
		r.ReviewAfter = &v
	}
	return r, nil
}

func (db *DB) LatestResolution(ctx context.Context, group string) (*store.ConflictResolution, error) {
	r, err := scanResolution(db.q().QueryRowContext(ctx,
		`SELECT `+resolutionCols+` FROM conflict_resolution WHERE group_id=? ORDER BY created_at DESC, id DESC LIMIT 1`, group))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (db *DB) ListResolutions(ctx context.Context, group string) ([]store.ConflictResolution, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT `+resolutionCols+` FROM conflict_resolution WHERE group_id=? ORDER BY created_at ASC, id ASC`, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.ConflictResolution
	for rows.Next() {
		r, err := scanResolution(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (db *DB) InsertLink(ctx context.Context, l store.Link) error {
	if l.CreatedAt == 0 {
		l.CreatedAt = now()
	}
	_, err := db.q().ExecContext(ctx,
		`INSERT INTO link(src_type, src_id, dst_type, dst_id, relation, note, created_at)
		 VALUES (?,?,?,?,?,?,?)
		 ON CONFLICT(src_type, src_id, dst_type, dst_id, relation) DO NOTHING`,
		l.SrcType, l.SrcID, l.DstType, l.DstID, l.Relation, nullStr(l.Note), l.CreatedAt)
	return err
}

func (db *DB) ListLinks(ctx context.Context, ownerType string, ownerID int64) ([]store.Link, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT id, src_type, src_id, dst_type, dst_id, relation, COALESCE(note,''), created_at
		   FROM link
		  WHERE (src_type=? AND src_id=?) OR (dst_type=? AND dst_id=?)
		  ORDER BY created_at ASC`,
		ownerType, ownerID, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Link
	for rows.Next() {
		var l store.Link
		if err := rows.Scan(&l.ID, &l.SrcType, &l.SrcID, &l.DstType, &l.DstID, &l.Relation, &l.Note, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
