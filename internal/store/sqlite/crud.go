package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/store"
)

func (db *DB) EnsureLocalUser(ctx context.Context) (int64, error) {
	var id int64
	err := db.q().QueryRowContext(ctx, `SELECT id FROM app_user WHERE id=1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	ts := now()
	_, err = db.q().ExecContext(ctx,
		`INSERT INTO app_user(id, email, display_name, status, created_at) VALUES (1, NULL, 'local', 'active', ?)`, ts)
	if err != nil {
		return 0, err
	}
	return 1, nil
}

func (db *DB) UpsertProject(ctx context.Context, p store.Project) (store.Project, error) {
	// 1) Prefer stable remote identity.
	if p.RemoteURLNorm != "" {
		if existing, err := db.getProjectByRemote(ctx, p.RemoteURLNorm); err != nil {
			return store.Project{}, err
		} else if existing != nil {
			if p.RootHint != "" && p.RootHint != existing.RootHint {
				_, _ = db.q().ExecContext(ctx, `UPDATE project SET root_hint=? WHERE id=?`, p.RootHint, existing.ID)
				existing.RootHint = p.RootHint
			}
			if p.RootHint != "" {
				if err := db.mergeProjectsSameRoot(ctx, existing.ID, p.RootHint); err != nil {
					return store.Project{}, err
				}
			}
			return *existing, nil
		}
	}

	// 2) Same checkout path already registered (e.g. earlier local/ hash before origin existed).
	if p.RootHint != "" {
		if existing, err := db.getProjectByRootHint(ctx, p.RootHint); err != nil {
			return store.Project{}, err
		} else if existing != nil {
			if p.RemoteURLNorm != "" && existing.RemoteURLNorm != p.RemoteURLNorm {
				// Upgrade local/… → real remote without changing id (keeps scope_ref stable).
				if strings.HasPrefix(existing.RemoteURLNorm, "local/") || existing.RemoteURLNorm == "" {
					_, err := db.q().ExecContext(ctx,
						`UPDATE project SET remote_url_norm=?, name=? WHERE id=?`,
						p.RemoteURLNorm, p.Name, existing.ID)
					if err != nil {
						return store.Project{}, err
					}
					existing.RemoteURLNorm = p.RemoteURLNorm
					existing.Name = p.Name
				}
			}
			return *existing, nil
		}
	}

	if p.CreatedAt == 0 {
		p.CreatedAt = now()
	}
	res, err := db.q().ExecContext(ctx,
		`INSERT INTO project(name, remote_url_norm, root_hint, created_by, created_at) VALUES (?,?,?,?,?)`,
		p.Name, p.RemoteURLNorm, nullStr(p.RootHint), nullInt64(p.CreatedBy), p.CreatedAt,
	)
	if err != nil {
		return store.Project{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.Project{}, err
	}
	p.ID = id
	if p.RootHint != "" {
		if err := db.mergeProjectsSameRoot(ctx, p.ID, p.RootHint); err != nil {
			return store.Project{}, err
		}
	}
	return p, nil
}

func (db *DB) getProjectByRemote(ctx context.Context, remote string) (*store.Project, error) {
	return db.scanProject(ctx, `SELECT id, name, remote_url_norm, root_hint, created_by, created_at FROM project WHERE remote_url_norm=?`, remote)
}

func (db *DB) getProjectByRootHint(ctx context.Context, root string) (*store.Project, error) {
	return db.scanProject(ctx, `SELECT id, name, remote_url_norm, root_hint, created_by, created_at FROM project WHERE root_hint=? ORDER BY id ASC LIMIT 1`, root)
}

func (db *DB) scanProject(ctx context.Context, q string, args ...any) (*store.Project, error) {
	var p store.Project
	var rootHint sql.NullString
	var createdBy sql.NullInt64
	err := db.q().QueryRowContext(ctx, q, args...).Scan(
		&p.ID, &p.Name, &p.RemoteURLNorm, &rootHint, &createdBy, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.RootHint = rootHint.String
	p.CreatedBy = int64Ptr(createdBy)
	return &p, nil
}

// mergeProjectsSameRoot rewrites scope_ref on memory/experience from duplicate
// project rows that share root_hint onto keepID, then deletes the duplicates.
func (db *DB) mergeProjectsSameRoot(ctx context.Context, keepID int64, rootHint string) error {
	rows, err := db.q().QueryContext(ctx,
		`SELECT id FROM project WHERE root_hint=? AND id<>?`, rootHint, keepID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var orphans []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		orphans = append(orphans, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	keepRef := fmt.Sprintf("%d", keepID)
	for _, oid := range orphans {
		oldRef := fmt.Sprintf("%d", oid)
		if _, err := db.q().ExecContext(ctx,
			`UPDATE experience SET scope_ref=? WHERE scope='project' AND scope_ref=?`, keepRef, oldRef); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx,
			`UPDATE memory SET scope_ref=? WHERE scope='project' AND scope_ref=?`, keepRef, oldRef); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx,
			`UPDATE raw SET project_id=? WHERE project_id=?`, keepID, oid); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx, `DELETE FROM project WHERE id=?`, oid); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) GetProject(ctx context.Context, id int64) (*store.Project, error) {
	var p store.Project
	var rootHint sql.NullString
	var createdBy sql.NullInt64
	err := db.q().QueryRowContext(ctx,
		`SELECT id, name, remote_url_norm, root_hint, created_by, created_at FROM project WHERE id=?`, id,
	).Scan(&p.ID, &p.Name, &p.RemoteURLNorm, &rootHint, &createdBy, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.RootHint = rootHint.String
	p.CreatedBy = int64Ptr(createdBy)
	return &p, nil
}

func (db *DB) InsertRaw(ctx context.Context, r store.Raw) (store.Raw, error) {
	if r.CreatedAt == 0 {
		r.CreatedAt = now()
	}
	res, err := db.q().ExecContext(ctx, `
		INSERT INTO raw(uid, hash, channel, source_ref, actor_id, agent, session_id, cwd, project_id,
		                mime, lang, body, redaction, rulepack_version, sensitivity, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.UID, r.Hash, r.Channel, nullStr(r.SourceRef), nullInt64(r.ActorID), nullStr(r.Agent),
		nullStr(r.SessionID), nullStr(r.Cwd), nullInt64(r.ProjectID), nullStr(""),
		r.Lang, r.Body, nullStr(r.Redaction), nullStr(r.RulePack), r.Sensitivity, r.CreatedAt,
	)
	if err != nil {
		return store.Raw{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.Raw{}, err
	}
	r.ID = id
	return r, nil
}

func (db *DB) GetRawByHash(ctx context.Context, hash string) (*store.Raw, error) {
	return db.scanRaw(ctx, `SELECT id, uid, hash, channel, source_ref, actor_id, agent, session_id, cwd,
		project_id, lang, body, redaction, rulepack_version, sensitivity, created_at FROM raw WHERE hash=?`, hash)
}

func (db *DB) GetRaw(ctx context.Context, id int64) (*store.Raw, error) {
	return db.scanRaw(ctx, `SELECT id, uid, hash, channel, source_ref, actor_id, agent, session_id, cwd,
		project_id, lang, body, redaction, rulepack_version, sensitivity, created_at FROM raw WHERE id=?`, id)
}

func (db *DB) scanRaw(ctx context.Context, q string, arg any) (*store.Raw, error) {
	var r store.Raw
	var sourceRef, agent, sessionID, cwd, redaction, rulePack sql.NullString
	var actorID, projectID sql.NullInt64
	err := db.q().QueryRowContext(ctx, q, arg).Scan(
		&r.ID, &r.UID, &r.Hash, &r.Channel, &sourceRef, &actorID, &agent, &sessionID, &cwd,
		&projectID, &r.Lang, &r.Body, &redaction, &rulePack, &r.Sensitivity, &r.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.SourceRef = sourceRef.String
	r.Agent = agent.String
	r.SessionID = sessionID.String
	r.Cwd = cwd.String
	r.Redaction = redaction.String
	r.RulePack = rulePack.String
	r.ActorID = int64Ptr(actorID)
	r.ProjectID = int64Ptr(projectID)
	return &r, nil
}

func (db *DB) InsertMemory(ctx context.Context, m store.Memory) (store.Memory, error) {
	if m.CreatedAt == 0 {
		m.CreatedAt = now()
	}
	if m.Rev == 0 {
		m.Rev = 1
	}
	var key any
	if m.MemKey != nil {
		key = *m.MemKey
	}
	res, err := db.q().ExecContext(ctx, `
		INSERT INTO memory(uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		                   pinned, priority, status, sensitivity, actor_id, rev, created_at, use_count)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0)`,
		m.UID, m.Lineage, m.Scope, m.ScopeRef, key, m.MemType, m.Title, m.BodyMD, m.Lang,
		boolToInt(m.Pinned), m.Priority, m.Status, m.Sensitivity, nullInt64(m.ActorID), m.Rev, m.CreatedAt,
	)
	if err != nil {
		return store.Memory{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.Memory{}, err
	}
	m.ID = id
	return m, nil
}

func (db *DB) GetMemoryByUID(ctx context.Context, uid string) (*store.Memory, error) {
	return db.scanMemory(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE uid=?`, uid)
}

func (db *DB) GetActiveMemoryByKey(ctx context.Context, scope, scopeRef, key string) (*store.Memory, error) {
	return db.scanMemory(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE scope=? AND scope_ref=? AND mem_key=? AND status='active'`, scope, scopeRef, key)
}

func (db *DB) scanMemory(ctx context.Context, q string, args ...any) (*store.Memory, error) {
	var m store.Memory
	var memKey sql.NullString
	var actorID, tLast sql.NullInt64
	var pinned int
	err := db.q().QueryRowContext(ctx, q, args...).Scan(
		&m.ID, &m.UID, &m.Lineage, &m.Scope, &m.ScopeRef, &memKey, &m.MemType, &m.Title, &m.BodyMD, &m.Lang,
		&pinned, &m.Priority, &m.Status, &m.Sensitivity, &actorID, &m.Rev, &m.CreatedAt, &tLast, &m.UseCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.MemKey = strPtr(memKey)
	m.ActorID = int64Ptr(actorID)
	m.TLastUsed = int64Ptr(tLast)
	m.Pinned = pinned == 1
	return &m, nil
}

func (db *DB) ListActiveMemory(ctx context.Context, scopes []store.ScopeFilter, limit int) ([]store.Memory, error) {
	if limit <= 0 {
		limit = 20
	}
	if len(scopes) == 0 {
		scopes = []store.ScopeFilter{{Scope: "global", ScopeRef: ""}}
	}
	var clauses []string
	var args []any
	for _, s := range scopes {
		clauses = append(clauses, "(scope=? AND scope_ref=?)")
		args = append(args, s.Scope, s.ScopeRef)
	}
	q := fmt.Sprintf(`
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE status='active' AND (%s)
		ORDER BY pinned DESC, priority DESC, created_at DESC LIMIT ?`, strings.Join(clauses, " OR "))
	args = append(args, limit)
	rows, err := db.q().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRows(rows)
}

func (db *DB) ListMemoriesByIDs(ctx context.Context, ids []int64) ([]store.Memory, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf(`
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := db.q().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemoryRows(rows)
}

func scanMemoryRows(rows *sql.Rows) ([]store.Memory, error) {
	var out []store.Memory
	for rows.Next() {
		var m store.Memory
		var memKey sql.NullString
		var actorID, tLast sql.NullInt64
		var pinned int
		if err := rows.Scan(
			&m.ID, &m.UID, &m.Lineage, &m.Scope, &m.ScopeRef, &memKey, &m.MemType, &m.Title, &m.BodyMD, &m.Lang,
			&pinned, &m.Priority, &m.Status, &m.Sensitivity, &actorID, &m.Rev, &m.CreatedAt, &tLast, &m.UseCount,
		); err != nil {
			return nil, err
		}
		m.MemKey = strPtr(memKey)
		m.ActorID = int64Ptr(actorID)
		m.TLastUsed = int64Ptr(tLast)
		m.Pinned = pinned == 1
		out = append(out, m)
	}
	return out, rows.Err()
}

func (db *DB) SupersedeMemory(ctx context.Context, oldID int64, expectedRev int, next store.Memory) (store.Memory, error) {
	res, err := db.q().ExecContext(ctx,
		`UPDATE memory SET status='superseded', rev=rev+1 WHERE id=? AND rev=? AND status='active'`,
		oldID, expectedRev,
	)
	if err != nil {
		return store.Memory{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return store.Memory{}, fmt.Errorf("write conflict on memory %d", oldID)
	}
	next, err = db.InsertMemory(ctx, next)
	if err != nil {
		return store.Memory{}, err
	}
	_, err = db.q().ExecContext(ctx, `UPDATE memory SET superseded_by=? WHERE id=?`, next.ID, oldID)
	if err != nil {
		return store.Memory{}, err
	}
	if err := db.DeleteSearchDoc(ctx, "memory", oldID); err != nil {
		return store.Memory{}, err
	}
	return next, nil
}

func (db *DB) InsertExperience(ctx context.Context, e store.Experience) (store.Experience, error) {
	if e.TCreated == 0 {
		e.TCreated = now()
	}
	if e.Rev == 0 {
		e.Rev = 1
	}
	res, err := db.q().ExecContext(ctx, `
		INSERT INTO experience(uid, exp_type, scope, scope_ref, title, symptom, symptom_key, root_cause,
		                       fix_md, fix_hash, applies_when, invalid_when, outcome, confidence, trust,
		                       conflict_group, lang, sensitivity, actor_id, rev, t_created, use_count)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0)`,
		e.UID, e.ExpType, e.Scope, e.ScopeRef, e.Title, nullStr(e.Symptom), nullStr(e.SymptomKey),
		nullStr(e.RootCause), nullStr(e.FixMD), nullStr(e.FixHash), nullStr(e.AppliesWhen),
		nullStr(e.InvalidWhen), e.Outcome, e.Confidence, e.Trust, nullStr(e.ConflictGroup),
		e.Lang, e.Sensitivity, nullInt64(e.ActorID), e.Rev, e.TCreated,
	)
	if err != nil {
		return store.Experience{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.Experience{}, err
	}
	e.ID = id
	return e, nil
}

func (db *DB) GetExperienceByUID(ctx context.Context, uid string) (*store.Experience, error) {
	return db.scanExperience(ctx, experienceSelect+` WHERE uid=?`, uid)
}

func (db *DB) GetExperience(ctx context.Context, id int64) (*store.Experience, error) {
	return db.scanExperience(ctx, experienceSelect+` WHERE id=?`, id)
}

const experienceSelect = `
	SELECT id, uid, exp_type, scope, scope_ref, title, symptom, symptom_key, root_cause, fix_md, fix_hash,
	       applies_when, invalid_when, outcome, confidence, trust, conflict_group, lang, sensitivity,
	       actor_id, rev, t_created, t_last_used, use_count
	FROM experience`

func (db *DB) scanExperience(ctx context.Context, q string, args ...any) (*store.Experience, error) {
	var e store.Experience
	var symptom, symptomKey, rootCause, fixMD, fixHash, applies, invalid, conflict sql.NullString
	var actorID, tLast sql.NullInt64
	err := db.q().QueryRowContext(ctx, q, args...).Scan(
		&e.ID, &e.UID, &e.ExpType, &e.Scope, &e.ScopeRef, &e.Title, &symptom, &symptomKey, &rootCause,
		&fixMD, &fixHash, &applies, &invalid, &e.Outcome, &e.Confidence, &e.Trust, &conflict,
		&e.Lang, &e.Sensitivity, &actorID, &e.Rev, &e.TCreated, &tLast, &e.UseCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Symptom = symptom.String
	e.SymptomKey = symptomKey.String
	e.RootCause = rootCause.String
	e.FixMD = fixMD.String
	e.FixHash = fixHash.String
	e.AppliesWhen = applies.String
	e.InvalidWhen = invalid.String
	e.ConflictGroup = conflict.String
	e.ActorID = int64Ptr(actorID)
	e.TLastUsed = int64Ptr(tLast)
	return &e, nil
}

func (db *DB) ListExperiencesByIDs(ctx context.Context, ids []int64) ([]store.Experience, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	q := experienceSelect + ` WHERE id IN (` + strings.Join(ph, ",") + `)`
	rows, err := db.q().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExperienceRows(rows)
}

func (db *DB) FindExperiencesBySymptomKey(ctx context.Context, key string) ([]store.Experience, error) {
	if key == "" {
		return nil, nil
	}
	rows, err := db.q().QueryContext(ctx, experienceSelect+` WHERE symptom_key=? AND trust NOT IN ('archived','falsified')`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExperienceRows(rows)
}

func scanExperienceRows(rows *sql.Rows) ([]store.Experience, error) {
	var out []store.Experience
	for rows.Next() {
		var e store.Experience
		var symptom, symptomKey, rootCause, fixMD, fixHash, applies, invalid, conflict sql.NullString
		var actorID, tLast sql.NullInt64
		if err := rows.Scan(
			&e.ID, &e.UID, &e.ExpType, &e.Scope, &e.ScopeRef, &e.Title, &symptom, &symptomKey, &rootCause,
			&fixMD, &fixHash, &applies, &invalid, &e.Outcome, &e.Confidence, &e.Trust, &conflict,
			&e.Lang, &e.Sensitivity, &actorID, &e.Rev, &e.TCreated, &tLast, &e.UseCount,
		); err != nil {
			return nil, err
		}
		e.Symptom = symptom.String
		e.SymptomKey = symptomKey.String
		e.RootCause = rootCause.String
		e.FixMD = fixMD.String
		e.FixHash = fixHash.String
		e.AppliesWhen = applies.String
		e.InvalidWhen = invalid.String
		e.ConflictGroup = conflict.String
		e.ActorID = int64Ptr(actorID)
		e.TLastUsed = int64Ptr(tLast)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (db *DB) MarkContested(ctx context.Context, ids []int64, group string) error {
	for _, id := range ids {
		if _, err := db.q().ExecContext(ctx,
			`UPDATE experience SET trust='contested', conflict_group=? WHERE id=?`, group, id); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) InsertEvidence(ctx context.Context, ev store.Evidence) (store.Evidence, error) {
	if ev.CreatedAt == 0 {
		ev.CreatedAt = now()
	}
	res, err := db.q().ExecContext(ctx,
		`INSERT INTO evidence(owner_type, owner_id, raw_id, quote, role, created_at) VALUES (?,?,?,?,?,?)`,
		ev.OwnerType, ev.OwnerID, ev.RawID, nullStr(ev.Quote), ev.Role, ev.CreatedAt,
	)
	if err != nil {
		return store.Evidence{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return store.Evidence{}, err
	}
	ev.ID = id
	return ev, nil
}

func (db *DB) ListEvidence(ctx context.Context, ownerType string, ownerID int64) ([]store.Evidence, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT id, owner_type, owner_id, raw_id, quote, role, created_at FROM evidence WHERE owner_type=? AND owner_id=?`,
		ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Evidence
	for rows.Next() {
		var ev store.Evidence
		var quote sql.NullString
		if err := rows.Scan(&ev.ID, &ev.OwnerType, &ev.OwnerID, &ev.RawID, &quote, &ev.Role, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Quote = quote.String
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (db *DB) SetTags(ctx context.Context, ownerType string, ownerID int64, tags []string) error {
	_, err := db.q().ExecContext(ctx, `DELETE FROM tag WHERE owner_type=? AND owner_id=?`, ownerType, ownerID)
	if err != nil {
		return err
	}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, err := db.q().ExecContext(ctx,
			`INSERT OR IGNORE INTO tag(owner_type, owner_id, tag) VALUES (?,?,?)`, ownerType, ownerID, t); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) BumpUse(ctx context.Context, kind string, id int64, ts int64) error {
	if ts == 0 {
		ts = time.Now().UTC().Unix()
	}
	switch kind {
	case "memory":
		_, err := db.q().ExecContext(ctx, `UPDATE memory SET use_count=use_count+1, t_last_used=? WHERE id=?`, ts, id)
		return err
	case "experience":
		_, err := db.q().ExecContext(ctx, `UPDATE experience SET use_count=use_count+1, t_last_used=? WHERE id=?`, ts, id)
		return err
	default:
		return fmt.Errorf("unknown kind %q", kind)
	}
}

func (db *DB) InsertAudit(ctx context.Context, action, targetType string, targetID *int64, targetUID, detail string) error {
	_, err := db.q().ExecContext(ctx,
		`INSERT INTO audit(ts, actor_id, action, target_type, target_id, target_uid, detail) VALUES (?,?,?,?,?,?,?)`,
		now(), 1, action, nullStr(targetType), nullInt64(targetID), nullStr(targetUID), nullStr(detail),
	)
	return err
}

func (db *DB) ListAudit(ctx context.Context, sinceUnix int64, limit int) ([]store.AuditRow, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := db.q().QueryContext(ctx, `
		SELECT id, ts, action, target_type, target_id, target_uid, detail
		FROM audit WHERE ts >= ? ORDER BY ts ASC, id ASC LIMIT ?`, sinceUnix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.AuditRow
	for rows.Next() {
		var r store.AuditRow
		var targetType, targetUID, detail sql.NullString
		var targetID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.TS, &r.Action, &targetType, &targetID, &targetUID, &detail); err != nil {
			return nil, err
		}
		r.TargetType = targetType.String
		r.TargetUID = targetUID.String
		r.Detail = detail.String
		r.TargetID = int64Ptr(targetID)
		out = append(out, r)
	}
	return out, rows.Err()
}
