package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/linyu19872008/rocky-mem/internal/store"
	"github.com/linyu19872008/rocky-mem/internal/textnorm"
)

// WriteSearchDoc upserts search_doc and FTS in the three-step SQLite contract.
func (db *DB) WriteSearchDoc(ctx context.Context, doc store.SearchDoc) error {
	if doc.Updated == 0 {
		doc.Updated = now()
	}
	var id int64
	err := db.q().QueryRowContext(ctx,
		`SELECT id FROM search_doc WHERE kind=? AND owner_id=?`, doc.Kind, doc.OwnerID,
	).Scan(&id)
	switch {
	case err == nil:
		if _, err := db.q().ExecContext(ctx, `DELETE FROM search_doc_fts WHERE rowid=?`, id); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx,
			`UPDATE search_doc SET lang=?, title=?, body=?, updated_at=? WHERE id=?`,
			doc.Lang, doc.Title, doc.Body, doc.Updated, id,
		); err != nil {
			return err
		}
	case err == sql.ErrNoRows:
		res, err := db.q().ExecContext(ctx,
			`INSERT INTO search_doc(kind, owner_id, lang, title, body, updated_at) VALUES (?,?,?,?,?,?)`,
			doc.Kind, doc.OwnerID, doc.Lang, doc.Title, doc.Body, doc.Updated,
		)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
	default:
		return err
	}
	_, err = db.q().ExecContext(ctx,
		`INSERT INTO search_doc_fts(rowid, title, body) VALUES (?,?,?)`, id, doc.Title, doc.Body)
	return err
}

func (db *DB) DeleteSearchDoc(ctx context.Context, kind string, ownerID int64) error {
	var id int64
	err := db.q().QueryRowContext(ctx,
		`SELECT id FROM search_doc WHERE kind=? AND owner_id=?`, kind, ownerID,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := db.q().ExecContext(ctx, `DELETE FROM search_doc_fts WHERE rowid=?`, id); err != nil {
		return err
	}
	_, err = db.q().ExecContext(ctx, `DELETE FROM search_doc WHERE id=?`, id)
	return err
}

// SearchCandidates returns FTS hits. Score is only meaningful within this call.
func (db *DB) SearchCandidates(ctx context.Context, query, kind string, limit int) ([]store.Candidate, error) {
	if limit <= 0 {
		limit = 20
	}
	q := textnorm.NormalizeQuery(query)
	if q == "" {
		return nil, nil
	}

	// Short queries (≤2 runes): LIKE is required — FTS5 trigram is unreliable here.
	if utf8.RuneCountInString(q) <= 2 {
		return db.searchLike(ctx, q, kind, limit)
	}

	fts, err := db.searchFTS(ctx, q, kind, limit)
	if err != nil {
		return db.searchLike(ctx, q, kind, limit)
	}
	if len(fts) > 0 {
		return fts, nil
	}
	// Empty FTS → LIKE fallback (missed substrings / punctuation).
	return db.searchLike(ctx, q, kind, limit)
}

func (db *DB) searchFTS(ctx context.Context, q, kind string, limit int) ([]store.Candidate, error) {
	match := ftsMatchQuery(q)
	sqlQ := `
		SELECT sd.kind, sd.owner_id, bm25(search_doc_fts) AS score
		FROM search_doc_fts
		JOIN search_doc sd ON sd.id = search_doc_fts.rowid
		WHERE search_doc_fts MATCH ?`
	args := []any{match}
	if kind != "" {
		sqlQ += ` AND sd.kind=?`
		args = append(args, kind)
	}
	sqlQ += ` ORDER BY score LIMIT ?`
	args = append(args, limit)

	rows, err := db.q().QueryContext(ctx, sqlQ, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Candidate
	for rows.Next() {
		var c store.Candidate
		if err := rows.Scan(&c.Kind, &c.OwnerID, &c.Score); err != nil {
			return nil, err
		}
		c.Score = -c.Score
		out = append(out, c)
	}
	return out, rows.Err()
}

func (db *DB) searchLike(ctx context.Context, q, kind string, limit int) ([]store.Candidate, error) {
	pat := "%" + escapeLike(q) + "%"
	sqlQ := `SELECT kind, owner_id, 1.0 AS score FROM search_doc WHERE (title LIKE ? ESCAPE '\' OR body LIKE ? ESCAPE '\')`
	args := []any{pat, pat}
	if kind != "" {
		sqlQ += ` AND kind=?`
		args = append(args, kind)
	}
	sqlQ += ` LIMIT ?`
	args = append(args, limit)
	rows, err := db.q().QueryContext(ctx, sqlQ, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Candidate
	for rows.Next() {
		var c store.Candidate
		if err := rows.Scan(&c.Kind, &c.OwnerID, &c.Score); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func ftsMatchQuery(q string) string {
	q = strings.ReplaceAll(q, `"`, `""`)
	parts := strings.Fields(q)
	if len(parts) == 0 {
		return `""`
	}
	if len(parts) == 1 {
		return `"` + parts[0] + `"`
	}
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(`"` + p + `"`)
	}
	return b.String()
}

func (db *DB) Doctor(ctx context.Context) (store.DoctorReport, error) {
	rep := store.DoctorReport{Notices: []string{}}
	_ = db.q().QueryRowContext(ctx, `SELECT v FROM meta WHERE k='schema_version'`).Scan(&rep.SchemaVersion)
	_ = db.q().QueryRowContext(ctx, `SELECT v FROM meta WHERE k='fts_mode'`).Scan(&rep.FTSMode)

	var fts int
	err := db.q().QueryRowContext(ctx, `SELECT sqlite_compileoption_used('ENABLE_FTS5')`).Scan(&fts)
	rep.FTSAvailable = err == nil && fts == 1
	if !rep.FTSAvailable {
		rep.Notices = append(rep.Notices, "degraded:fts_unavailable")
	}

	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM memory WHERE status='active'`).Scan(&rep.ActiveMemory)
	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM experience WHERE trust <> 'archived'`).Scan(&rep.IndexableExp)
	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM search_doc WHERE kind='memory'`).Scan(&rep.MemoryDocCount)
	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM search_doc WHERE kind='experience'`).Scan(&rep.ExpDocCount)
	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM memory`).Scan(&rep.MemoryCount)
	_ = db.q().QueryRowContext(ctx, `SELECT COUNT(*) FROM experience`).Scan(&rep.ExperienceCount)

	// Corpus counters: chunk coverage must match chunks of active sources.
	if st, err := db.CorpusStats(ctx); err == nil {
		rep.SourceCount, rep.ActiveSources = st.Sources, st.ActiveSources
		rep.ChunkCount, rep.ActiveChunks, rep.ChunkDocCount = st.Chunks, st.ActiveChunks, st.ChunkDocs
	}
	rep.SearchDocOK = rep.MemoryDocCount == rep.ActiveMemory &&
		rep.ExpDocCount == rep.IndexableExp &&
		rep.ChunkDocCount == rep.ActiveChunks
	if !rep.SearchDocOK {
		rep.Notices = append(rep.Notices, fmt.Sprintf(
			"search_doc drift: memory %d/%d experience %d/%d chunk %d/%d",
			rep.MemoryDocCount, rep.ActiveMemory, rep.ExpDocCount, rep.IndexableExp,
			rep.ChunkDocCount, rep.ActiveChunks,
		))
	}
	return rep, nil
}
