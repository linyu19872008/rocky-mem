package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/linyu19872008/rocky-mem/db"
	"github.com/linyu19872008/rocky-mem/internal/store"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	store.RegisterOpener("postgres", func(dsn string) (store.Store, error) {
		return Open(dsn)
	})
}

var _ store.Store = (*DB)(nil)

// DB implements store.Store for PostgreSQL (pg_trgm retrieval).
type DB struct {
	sql    *sql.DB
	tx     *sql.Tx // non-nil while inside WithTx
	schema string  // optional search_path schema (tests)
	mu     sync.Mutex
}

type querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (db *DB) q() querier {
	if db.tx != nil {
		return db.tx
	}
	return db.sql
}

var schemaNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Open creates or opens a PostgreSQL database at dsn and applies migrations.
func Open(dsn string) (*DB, error) {
	return open(dsn, "")
}

// OpenInSchema opens dsn, CREATE SCHEMA IF NOT EXISTS schema, sets search_path,
// then migrates. Used by conformance tests for isolation. schema must be a
// simple SQL identifier ([a-zA-Z_][a-zA-Z0-9_]*).
func OpenInSchema(dsn, schema string) (*DB, error) {
	if !schemaNameRe.MatchString(schema) {
		return nil, fmt.Errorf("invalid schema name %q", schema)
	}
	return open(dsn, schema)
}

func open(dsn, schema string) (*DB, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(0)

	dbx := &DB{sql: sqlDB, schema: schema}
	if schema != "" {
		if _, err := sqlDB.Exec(`CREATE SCHEMA IF NOT EXISTS ` + schema); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("create schema %s: %w", schema, err)
		}
		// pg_trgm must live somewhere the search_path always sees. The migration
		// creates it without a schema, which would install it into whichever
		// schema happens to be first — so the first isolated schema works and
		// every later one fails on `gin_trgm_ops`. Install it into public first.
		if _, err := sqlDB.Exec(`CREATE EXTENSION IF NOT EXISTS pg_trgm SCHEMA public`); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("create extension pg_trgm: %w", err)
		}
		// Keep public on the path so pg_trgm operator classes resolve.
		if _, err := sqlDB.Exec(`SET search_path TO ` + schema + `, public`); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("set search_path: %w", err)
		}
	}
	if err := dbx.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return dbx, nil
}

// DropSchema drops a test schema (CASCADE). No-op when schema is empty.
func (db *DB) DropSchema(ctx context.Context) error {
	if db.schema == "" {
		return nil
	}
	if !schemaNameRe.MatchString(db.schema) {
		return fmt.Errorf("invalid schema name %q", db.schema)
	}
	_, err := db.sql.ExecContext(ctx, `DROP SCHEMA IF EXISTS `+db.schema+` CASCADE`)
	return err
}

func (db *DB) migrate() error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	current := 0
	var ver string
	err = db.sql.QueryRow(`SELECT v FROM meta WHERE k='schema_version'`).Scan(&ver)
	if err == nil {
		current, _ = strconv.Atoi(ver)
	}

	for _, m := range migrations {
		if m.Version <= current {
			continue
		}
		if _, err := db.sql.Exec(m.SQL); err != nil {
			return fmt.Errorf("apply %s: %w", m.Name, err)
		}
		if _, err := db.sql.Exec(
			`INSERT INTO meta(k,v) VALUES('schema_version',$1)
			 ON CONFLICT(k) DO UPDATE SET v=excluded.v`,
			strconv.Itoa(m.Version),
		); err != nil {
			return fmt.Errorf("set schema_version after %s: %w", m.Name, err)
		}
		current = m.Version
	}
	return nil
}

var loadMigrations = db.PostgresMigrations

func (db *DB) Close() error { return db.sql.Close() }

func (db *DB) Ping(ctx context.Context) error { return db.sql.PingContext(ctx) }

func (db *DB) WithTx(ctx context.Context, fn func(store.Store) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.tx != nil {
		return fn(db)
	}
	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	db.tx = tx
	defer func() { db.tx = nil }()
	if err := fn(db); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func now() int64 { return time.Now().UTC().Unix() }

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func strPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func int64Ptr(ns sql.NullInt64) *int64 {
	if !ns.Valid {
		return nil
	}
	v := ns.Int64
	return &v
}

func dollarList(n, start int) string {
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = "$" + strconv.Itoa(start+i)
	}
	return joinComma(parts)
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
