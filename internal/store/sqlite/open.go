package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/linyu19872008/rocky-mem/db"
	"github.com/linyu19872008/rocky-mem/internal/store"

	_ "modernc.org/sqlite"
)

func init() {
	store.RegisterOpener("sqlite", func(path string) (store.Store, error) {
		return Open(path)
	})
}

// DB implements store.Store for SQLite.
type DB struct {
	sql *sql.DB
	tx  *sql.Tx // non-nil while inside WithTx
	mu  sync.Mutex
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

// Open creates or opens a SQLite database at path and applies migrations.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	dbx := &DB{sql: sqlDB}
	if err := dbx.pragma(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := dbx.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return dbx, nil
}

func (db *DB) pragma() error {
	for _, p := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA trusted_schema=OFF",
	} {
		if _, err := db.sql.Exec(p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
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
		// Ensure version is recorded even if the script forgot the UPDATE.
		if _, err := db.sql.Exec(
			`INSERT INTO meta(k,v) VALUES('schema_version',?)
			 ON CONFLICT(k) DO UPDATE SET v=excluded.v`,
			strconv.Itoa(m.Version),
		); err != nil {
			return fmt.Errorf("set schema_version after %s: %w", m.Name, err)
		}
		current = m.Version
	}
	return nil
}

var loadMigrations = db.SQLiteMigrations

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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
