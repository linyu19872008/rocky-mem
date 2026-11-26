package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/linyu19872008/rocky-mem/internal/store"
)

const sourceCols = `id, uid, kind, uri, title, format, text_hash, char_count, chunk_count,
lang, sensitivity, status, superseded_by, project_id, actor_id, created_at, t_last_used, use_count`

func scanSource(sc interface{ Scan(...any) error }) (store.Source, error) {
	var s store.Source
	var sup, proj, actor, last sql.NullInt64
	if err := sc.Scan(&s.ID, &s.UID, &s.Kind, &s.URI, &s.Title, &s.Format, &s.TextHash,
		&s.CharCount, &s.ChunkCount, &s.Lang, &s.Sensitivity, &s.Status,
		&sup, &proj, &actor, &s.CreatedAt, &last, &s.UseCount); err != nil {
		return store.Source{}, err
	}
	if sup.Valid {
		v := sup.Int64
		s.SupersededBy = &v
	}
	if proj.Valid {
		v := proj.Int64
		s.ProjectID = &v
	}
	if actor.Valid {
		v := actor.Int64
		s.ActorID = &v
	}
	if last.Valid {
		v := last.Int64
		s.TLastUsed = &v
	}
	return s, nil
}

func inQ(n int) string {
	if n <= 0 {
		return "''"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func (db *DB) InsertSource(ctx context.Context, s store.Source) (store.Source, error) {
	if s.Status == "" {
		s.Status = "active"
	}
	if s.Lang == "" {
		s.Lang = "und"
	}
	if s.Sensitivity == 0 {
		s.Sensitivity = 2
	}
	if s.CreatedAt == 0 {
		s.CreatedAt = now()
	}
	res, err := db.q().ExecContext(ctx,
		`INSERT INTO source(uid, kind, uri, title, format, text_hash, char_count, chunk_count,
		                    lang, sensitivity, status, project_id, actor_id, created_at, use_count)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,0)`,
		s.UID, s.Kind, s.URI, s.Title, s.Format, s.TextHash, s.CharCount, s.ChunkCount,
		s.Lang, s.Sensitivity, s.Status, s.ProjectID, s.ActorID, s.CreatedAt)
	if err != nil {
		return store.Source{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.Source{}, err
	}
	s.ID = id
	return s, nil
}

func (db *DB) GetSourceByTextHash(ctx context.Context, hash string) (*store.Source, error) {
	s, err := scanSource(db.q().QueryRowContext(ctx,
		`SELECT `+sourceCols+` FROM source WHERE text_hash=?`, hash))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) GetSourceByUID(ctx context.Context, uid string) (*store.Source, error) {
	s, err := scanSource(db.q().QueryRowContext(ctx,
		`SELECT `+sourceCols+` FROM source WHERE uid=?`, uid))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) ArchiveSource(ctx context.Context, id, replacedBy, ts int64) error {
	if err := db.archiveChunkDocs(ctx, `id=?`, id); err != nil {
		return err
	}
	_, err := db.q().ExecContext(ctx,
		`UPDATE source SET status='archived', superseded_by=? WHERE id=? AND status='active'`,
		replacedBy, id)
	return err
}

// archiveChunkDocs drops search rows for chunks of the sources matching pred.
// Archived documents must vanish from retrieval, otherwise the candidate set
// gets crowded out by generations nobody should see.
func (db *DB) archiveChunkDocs(ctx context.Context, pred string, arg any) error {
	_, err := db.q().ExecContext(ctx,
		`DELETE FROM search_doc WHERE kind='chunk' AND owner_id IN (
		   SELECT c.id FROM chunk c JOIN source s ON s.id=c.source_id
		    WHERE s.`+pred+` AND s.status='active')`, arg)
	return err
}

func (db *DB) ArchiveSourceByURI(ctx context.Context, projectID *int64, uri string, keepID int64) (int, error) {
	if _, err := db.q().ExecContext(ctx,
		`DELETE FROM search_doc WHERE kind='chunk' AND owner_id IN (
		   SELECT c.id FROM chunk c JOIN source s ON s.id=c.source_id
		    WHERE s.uri=? AND s.status='active' AND s.id<>? AND s.project_id IS ?)`,
		uri, keepID, nullInt64(projectID)); err != nil {
		return 0, err
	}
	res, err := db.q().ExecContext(ctx,
		`UPDATE source SET status='archived', superseded_by=?
		  WHERE uri=? AND status='active' AND id<>? AND project_id IS ?`,
		keepID, uri, keepID, nullInt64(projectID))
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (db *DB) ListSourcesByIDs(ctx context.Context, ids []int64) ([]store.Source, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := db.q().QueryContext(ctx,
		`SELECT `+sourceCols+` FROM source WHERE id IN (`+inQ(len(ids))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (db *DB) InsertChunks(ctx context.Context, cs []store.Chunk) ([]store.Chunk, error) {
	out := make([]store.Chunk, 0, len(cs))
	for _, c := range cs {
		if c.Sensitivity == 0 {
			c.Sensitivity = 2
		}
		res, err := db.q().ExecContext(ctx,
			`INSERT INTO chunk(uid, source_id, ord, heading, text, char_count, sensitivity)
			 VALUES (?,?,?,?,?,?,?)`,
			c.UID, c.SourceID, c.Ord, nullStr(c.Heading), c.Text, c.CharCount, c.Sensitivity)
		if err != nil {
			return nil, fmt.Errorf("insert chunk %d: %w", c.Ord, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		c.ID = id
		out = append(out, c)
	}
	return out, nil
}

const chunkCols = `id, uid, source_id, ord, COALESCE(heading,''), text, char_count, sensitivity`

func scanChunk(sc interface{ Scan(...any) error }) (store.Chunk, error) {
	var c store.Chunk
	err := sc.Scan(&c.ID, &c.UID, &c.SourceID, &c.Ord, &c.Heading, &c.Text, &c.CharCount, &c.Sensitivity)
	return c, err
}

func (db *DB) GetChunkByUID(ctx context.Context, uid string) (*store.Chunk, error) {
	c, err := scanChunk(db.q().QueryRowContext(ctx,
		`SELECT `+chunkCols+` FROM chunk WHERE uid=?`, uid))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) ListChunksBySource(ctx context.Context, sourceID int64) ([]store.Chunk, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT `+chunkCols+` FROM chunk WHERE source_id=? ORDER BY ord`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Chunk
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (db *DB) ListActiveChunksByIDs(ctx context.Context, ids []int64) ([]store.Chunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := db.q().QueryContext(ctx,
		`SELECT c.id, c.uid, c.source_id, c.ord, COALESCE(c.heading,''), c.text, c.char_count, c.sensitivity
		   FROM chunk c JOIN source s ON s.id = c.source_id
		  WHERE c.id IN (`+inQ(len(ids))+`) AND s.status='active'`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Chunk
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (db *DB) DeleteSource(ctx context.Context, id int64) (int, error) {
	rows, err := db.q().QueryContext(ctx, `SELECT id FROM chunk WHERE source_id=?`, id)
	if err != nil {
		return 0, err
	}
	var ids []int64
	for rows.Next() {
		var cid int64
		if err := rows.Scan(&cid); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, cid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) > 0 {
		args := make([]any, len(ids))
		for i, cid := range ids {
			args[i] = cid
		}
		if _, err := db.q().ExecContext(ctx,
			`DELETE FROM search_doc WHERE kind='chunk' AND owner_id IN (`+inQ(len(ids))+`)`, args...); err != nil {
			return 0, err
		}
	}
	if _, err := db.q().ExecContext(ctx, `DELETE FROM source WHERE id=?`, id); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (db *DB) CorpusStats(ctx context.Context) (store.CorpusStats, error) {
	var st store.CorpusStats
	err := db.q().QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM source),
		       (SELECT COUNT(*) FROM source WHERE status='active'),
		       (SELECT COUNT(*) FROM chunk),
		       (SELECT COUNT(*) FROM chunk c JOIN source s ON s.id=c.source_id WHERE s.status='active'),
		       (SELECT COUNT(*) FROM search_doc WHERE kind='chunk')`).
		Scan(&st.Sources, &st.ActiveSources, &st.Chunks, &st.ActiveChunks, &st.ChunkDocs)
	return st, err
}

func (db *DB) ListAllActiveChunks(ctx context.Context) ([]store.Chunk, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT c.id, c.uid, c.source_id, c.ord, COALESCE(c.heading,''), c.text, c.char_count, c.sensitivity
		   FROM chunk c JOIN source s ON s.id = c.source_id
		  WHERE s.status='active'
		  ORDER BY c.source_id, c.ord`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Chunk
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
