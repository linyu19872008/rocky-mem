package postgres

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
	_, err = db.q().ExecContext(ctx, `
		INSERT INTO app_user(id, email, display_name, status, created_at)
		OVERRIDING SYSTEM VALUE
		VALUES (1, NULL, 'local', 'active', $1)`, ts)
	if err != nil {
		return 0, err
	}
	// Keep identity sequence ahead of the explicit id=1 seed.
	_, _ = db.q().ExecContext(ctx, `
		SELECT setval(pg_get_serial_sequence('app_user', 'id'),
			GREATEST(1, (SELECT COALESCE(MAX(id), 1) FROM app_user)))`)
	return 1, nil
}

func (db *DB) UpsertProject(ctx context.Context, p store.Project) (store.Project, error) {
	if p.RemoteURLNorm != "" {
		if existing, err := db.getProjectByRemote(ctx, p.RemoteURLNorm); err != nil {
			return store.Project{}, err
		} else if existing != nil {
			if p.RootHint != "" && p.RootHint != existing.RootHint {
				_, _ = db.q().ExecContext(ctx, `UPDATE project SET root_hint=$1 WHERE id=$2`, p.RootHint, existing.ID)
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

	if p.RootHint != "" {
		if existing, err := db.getProjectByRootHint(ctx, p.RootHint); err != nil {
			return store.Project{}, err
		} else if existing != nil {
			if p.RemoteURLNorm != "" && existing.RemoteURLNorm != p.RemoteURLNorm {
				if strings.HasPrefix(existing.RemoteURLNorm, "local/") || existing.RemoteURLNorm == "" {
					_, err := db.q().ExecContext(ctx,
						`UPDATE project SET remote_url_norm=$1, name=$2 WHERE id=$3`,
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
	err := db.q().QueryRowContext(ctx, `
		INSERT INTO project(name, remote_url_norm, root_hint, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		p.Name, p.RemoteURLNorm, nullStr(p.RootHint), nullInt64(p.CreatedBy), p.CreatedAt,
	).Scan(&p.ID)
	if err != nil {
		return store.Project{}, err
	}
	if p.RootHint != "" {
		if err := db.mergeProjectsSameRoot(ctx, p.ID, p.RootHint); err != nil {
			return store.Project{}, err
		}
	}
	return p, nil
}

func (db *DB) getProjectByRemote(ctx context.Context, remote string) (*store.Project, error) {
	return db.scanProject(ctx, `SELECT id, name, remote_url_norm, root_hint, created_by, created_at FROM project WHERE remote_url_norm=$1`, remote)
}

func (db *DB) getProjectByRootHint(ctx context.Context, root string) (*store.Project, error) {
	return db.scanProject(ctx, `SELECT id, name, remote_url_norm, root_hint, created_by, created_at FROM project WHERE root_hint=$1 ORDER BY id ASC LIMIT 1`, root)
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

func (db *DB) mergeProjectsSameRoot(ctx context.Context, keepID int64, rootHint string) error {
	rows, err := db.q().QueryContext(ctx,
		`SELECT id FROM project WHERE root_hint=$1 AND id<>$2`, rootHint, keepID)
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
			`UPDATE experience SET scope_ref=$1 WHERE scope='project' AND scope_ref=$2`, keepRef, oldRef); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx,
			`UPDATE memory SET scope_ref=$1 WHERE scope='project' AND scope_ref=$2`, keepRef, oldRef); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx,
			`UPDATE raw SET project_id=$1 WHERE project_id=$2`, keepID, oid); err != nil {
			return err
		}
		if _, err := db.q().ExecContext(ctx, `DELETE FROM project WHERE id=$1`, oid); err != nil {
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
		`SELECT id, name, remote_url_norm, root_hint, created_by, created_at FROM project WHERE id=$1`, id,
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
	err := db.q().QueryRowContext(ctx, `
		INSERT INTO raw(uid, hash, channel, source_ref, actor_id, agent, session_id, cwd, project_id,
		                mime, lang, body, redaction, rulepack_version, sensitivity, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id`,
		r.UID, r.Hash, r.Channel, nullStr(r.SourceRef), nullInt64(r.ActorID), nullStr(r.Agent),
		nullStr(r.SessionID), nullStr(r.Cwd), nullInt64(r.ProjectID), nullStr(""),
		r.Lang, r.Body, nullStr(r.Redaction), nullStr(r.RulePack), r.Sensitivity, r.CreatedAt,
	).Scan(&r.ID)
	if err != nil {
		return store.Raw{}, err
	}
	return r, nil
}

func (db *DB) GetRawByHash(ctx context.Context, hash string) (*store.Raw, error) {
	return db.scanRaw(ctx, `SELECT id, uid, hash, channel, source_ref, actor_id, agent, session_id, cwd,
		project_id, lang, body, redaction, rulepack_version, sensitivity, created_at FROM raw WHERE hash=$1`, hash)
}

func (db *DB) GetRaw(ctx context.Context, id int64) (*store.Raw, error) {
	return db.scanRaw(ctx, `SELECT id, uid, hash, channel, source_ref, actor_id, agent, session_id, cwd,
		project_id, lang, body, redaction, rulepack_version, sensitivity, created_at FROM raw WHERE id=$1`, id)
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
	err := db.q().QueryRowContext(ctx, `
		INSERT INTO memory(uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		                   pinned, priority, status, sensitivity, actor_id, rev, created_at, use_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,0)
		RETURNING id`,
		m.UID, m.Lineage, m.Scope, m.ScopeRef, key, m.MemType, m.Title, m.BodyMD, m.Lang,
		m.Pinned, m.Priority, m.Status, m.Sensitivity, nullInt64(m.ActorID), m.Rev, m.CreatedAt,
	).Scan(&m.ID)
	if err != nil {
		return store.Memory{}, err
	}
	return m, nil
}

func (db *DB) GetMemoryByUID(ctx context.Context, uid string) (*store.Memory, error) {
	return db.scanMemory(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE uid=$1`, uid)
}

func (db *DB) GetActiveMemoryByKey(ctx context.Context, scope, scopeRef, key string) (*store.Memory, error) {
	return db.scanMemory(ctx, `
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE scope=$1 AND scope_ref=$2 AND mem_key=$3 AND status='active'`, scope, scopeRef, key)
}

func (db *DB) scanMemory(ctx context.Context, q string, args ...any) (*store.Memory, error) {
	var m store.Memory
	var memKey sql.NullString
	var actorID, tLast sql.NullInt64
	err := db.q().QueryRowContext(ctx, q, args...).Scan(
		&m.ID, &m.UID, &m.Lineage, &m.Scope, &m.ScopeRef, &memKey, &m.MemType, &m.Title, &m.BodyMD, &m.Lang,
		&m.Pinned, &m.Priority, &m.Status, &m.Sensitivity, &actorID, &m.Rev, &m.CreatedAt, &tLast, &m.UseCount,
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
	argN := 1
	for _, s := range scopes {
		clauses = append(clauses, fmt.Sprintf("(scope=$%d AND scope_ref=$%d)", argN, argN+1))
		args = append(args, s.Scope, s.ScopeRef)
		argN += 2
	}
	q := fmt.Sprintf(`
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE status='active' AND (%s)
		ORDER BY pinned DESC, priority DESC, created_at DESC LIMIT $%d`, strings.Join(clauses, " OR "), argN)
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
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	q := fmt.Sprintf(`
		SELECT id, uid, lineage, scope, scope_ref, mem_key, mem_type, title, body_md, lang,
		       pinned, priority, status, sensitivity, actor_id, rev, created_at, t_last_used, use_count
		FROM memory WHERE id IN (%s)`, dollarList(len(ids), 1))
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
		if err := rows.Scan(
			&m.ID, &m.UID, &m.Lineage, &m.Scope, &m.ScopeRef, &memKey, &m.MemType, &m.Title, &m.BodyMD, &m.Lang,
			&m.Pinned, &m.Priority, &m.Status, &m.Sensitivity, &actorID, &m.Rev, &m.CreatedAt, &tLast, &m.UseCount,
		); err != nil {
			return nil, err
		}
		m.MemKey = strPtr(memKey)
		m.ActorID = int64Ptr(actorID)
		m.TLastUsed = int64Ptr(tLast)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (db *DB) SupersedeMemory(ctx context.Context, oldID int64, expectedRev int, next store.Memory) (store.Memory, error) {
	res, err := db.q().ExecContext(ctx,
		`UPDATE memory SET status='superseded', rev=rev+1 WHERE id=$1 AND rev=$2 AND status='active'`,
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
	_, err = db.q().ExecContext(ctx, `UPDATE memory SET superseded_by=$1 WHERE id=$2`, next.ID, oldID)
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
	err := db.q().QueryRowContext(ctx, `
		INSERT INTO experience(uid, exp_type, scope, scope_ref, title, symptom, symptom_key, root_cause,
		                       fix_md, fix_hash, applies_when, invalid_when, outcome, confidence, trust,
		                       conflict_group, lang, sensitivity, actor_id, rev, t_created, use_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,0)
		RETURNING id`,
		e.UID, e.ExpType, e.Scope, e.ScopeRef, e.Title, nullStr(e.Symptom), nullStr(e.SymptomKey),
		nullStr(e.RootCause), nullStr(e.FixMD), nullStr(e.FixHash), nullStr(e.AppliesWhen),
		nullStr(e.InvalidWhen), e.Outcome, e.Confidence, e.Trust, nullStr(e.ConflictGroup),
		e.Lang, e.Sensitivity, nullInt64(e.ActorID), e.Rev, e.TCreated,
	).Scan(&e.ID)
	if err != nil {
		return store.Experience{}, err
	}
	return e, nil
}

func (db *DB) GetExperienceByUID(ctx context.Context, uid string) (*store.Experience, error) {
	return db.scanExperience(ctx, experienceSelect+` WHERE uid=$1`, uid)
}

func (db *DB) GetExperience(ctx context.Context, id int64) (*store.Experience, error) {
	return db.scanExperience(ctx, experienceSelect+` WHERE id=$1`, id)
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
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	q := experienceSelect + ` WHERE id IN (` + dollarList(len(ids), 1) + `)`
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
	rows, err := db.q().QueryContext(ctx, experienceSelect+` WHERE symptom_key=$1 AND trust NOT IN ('archived','falsified')`, key)
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
			`UPDATE experience SET trust='contested', conflict_group=$1 WHERE id=$2`, group, id); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) InsertEvidence(ctx context.Context, ev store.Evidence) (store.Evidence, error) {
	if ev.CreatedAt == 0 {
		ev.CreatedAt = now()
	}
	err := db.q().QueryRowContext(ctx,
		`INSERT INTO evidence(owner_type, owner_id, raw_id, quote, role, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		ev.OwnerType, ev.OwnerID, ev.RawID, nullStr(ev.Quote), ev.Role, ev.CreatedAt,
	).Scan(&ev.ID)
	if err != nil {
		return store.Evidence{}, err
	}
	return ev, nil
}

func (db *DB) ListEvidence(ctx context.Context, ownerType string, ownerID int64) ([]store.Evidence, error) {
	rows, err := db.q().QueryContext(ctx,
		`SELECT id, owner_type, owner_id, raw_id, quote, role, created_at FROM evidence WHERE owner_type=$1 AND owner_id=$2`,
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
	_, err := db.q().ExecContext(ctx, `DELETE FROM tag WHERE owner_type=$1 AND owner_id=$2`, ownerType, ownerID)
	if err != nil {
		return err
	}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, err := db.q().ExecContext(ctx,
			`INSERT INTO tag(owner_type, owner_id, tag) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			ownerType, ownerID, t); err != nil {
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
		_, err := db.q().ExecContext(ctx, `UPDATE memory SET use_count=use_count+1, t_last_used=$1 WHERE id=$2`, ts, id)
		return err
	case "experience":
		_, err := db.q().ExecContext(ctx, `UPDATE experience SET use_count=use_count+1, t_last_used=$1 WHERE id=$2`, ts, id)
		return err
	default:
		return fmt.Errorf("unknown kind %q", kind)
	}
}

func (db *DB) InsertAudit(ctx context.Context, action, targetType string, targetID *int64, targetUID, detail string) error {
	_, err := db.q().ExecContext(ctx,
		`INSERT INTO audit(ts, actor_id, action, target_type, target_id, target_uid, detail) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
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
		FROM audit WHERE ts >= $1 ORDER BY ts ASC, id ASC LIMIT $2`, sinceUnix, limit)
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
