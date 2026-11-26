package postgres

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/textnorm"
)

// WriteSearchDoc upserts search_doc in one step (generated search_text + GIN).
func (db *DB) WriteSearchDoc(ctx context.Context, doc store.SearchDoc) error {
	if doc.Updated == 0 {
		doc.Updated = now()
	}
	_, err := db.q().ExecContext(ctx, `
		INSERT INTO search_doc(kind, owner_id, lang, title, body, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (kind, owner_id) DO UPDATE
		  SET lang = excluded.lang,
		      title = excluded.title,
		      body = excluded.body,
		      updated_at = excluded.updated_at`,
		doc.Kind, doc.OwnerID, doc.Lang, doc.Title, doc.Body, doc.Updated,
	)
	return err
}

func (db *DB) DeleteSearchDoc(ctx context.Context, kind string, ownerID int64) error {
	_, err := db.q().ExecContext(ctx, `DELETE FROM search_doc WHERE kind=$1 AND owner_id=$2`, kind, ownerID)
	return err
}

// SearchCandidates returns pg_trgm / ILIKE hits. Score is only meaningful within this call.
func (db *DB) SearchCandidates(ctx context.Context, query, kind string, limit int) ([]store.Candidate, error) {
	if limit <= 0 {
		limit = 20
	}
	q := textnorm.NormalizeQuery(query)
	if q == "" {
		return nil, nil
	}

	// Short queries (≤2 runes): ILIKE only — pg_trgm is unreliable below 3 chars.
	if utf8.RuneCountInString(q) <= 2 {
		return db.searchILIKE(ctx, q, kind, limit)
	}

	hits, err := db.searchTrgm(ctx, q, kind, limit)
	if err != nil {
		return db.searchILIKE(ctx, q, kind, limit)
	}
	if len(hits) > 0 {
		return hits, nil
	}
	return db.searchILIKE(ctx, q, kind, limit)
}

func (db *DB) searchTrgm(ctx context.Context, q, kind string, limit int) ([]store.Candidate, error) {
	pat := "%" + escapeLike(q) + "%"
	sqlQ := `
		SELECT kind, owner_id, similarity(search_text, $1) AS score
		FROM search_doc
		WHERE search_text ILIKE $2 ESCAPE '\'`
	args := []any{q, pat}
	argN := 3
	if kind != "" {
		sqlQ += fmt.Sprintf(` AND kind=$%d`, argN)
		args = append(args, kind)
		argN++
	}
	sqlQ += fmt.Sprintf(` ORDER BY score DESC LIMIT $%d`, argN)
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

func (db *DB) searchILIKE(ctx context.Context, q, kind string, limit int) ([]store.Candidate, error) {
	pat := "%" + escapeLike(q) + "%"
	sqlQ := `SELECT kind, owner_id, 1.0::float8 AS score FROM search_doc WHERE (title ILIKE $1 ESCAPE '\' OR body ILIKE $2 ESCAPE '\')`
	args := []any{pat, pat}
	argN := 3
	if kind != "" {
		sqlQ += fmt.Sprintf(` AND kind=$%d`, argN)
		args = append(args, kind)
		argN++
	}
	sqlQ += fmt.Sprintf(` LIMIT $%d`, argN)
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

func (db *DB) Doctor(ctx context.Context) (store.DoctorReport, error) {
	rep := store.DoctorReport{Notices: []string{}}
	_ = db.q().QueryRowContext(ctx, `SELECT v FROM meta WHERE k='schema_version'`).Scan(&rep.SchemaVersion)
	_ = db.q().QueryRowContext(ctx, `SELECT v FROM meta WHERE k='fts_mode'`).Scan(&rep.FTSMode)

	var hasTrgm bool
	err := db.q().QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_trgm')`).Scan(&hasTrgm)
	rep.FTSAvailable = err == nil && hasTrgm
	if !rep.FTSAvailable {
		rep.Notices = append(rep.Notices, "degraded:pg_trgm_unavailable")
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
