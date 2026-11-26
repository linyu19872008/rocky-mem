// Package db embeds SQL migrations.
package db

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/sqlite/*.sql
var SQLiteFS embed.FS

//go:embed migrations/postgres/*.sql
var PostgresFS embed.FS

// Migration is one ordered SQL script.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// SQLiteMigrations returns migrations sorted by version ascending.
func SQLiteMigrations() ([]Migration, error) {
	return migrationsFrom(SQLiteFS, "migrations/sqlite")
}

// PostgresMigrations returns migrations sorted by version ascending.
func PostgresMigrations() ([]Migration, error) {
	return migrationsFrom(PostgresFS, "migrations/postgres")
}

func migrationsFrom(fs embed.FS, dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) < 1 {
			continue
		}
		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("migration name %q: %w", e.Name(), err)
		}
		body, err := fs.ReadFile(dir + "/" + e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: ver, Name: e.Name(), SQL: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// SQLiteInit is the P0 schema bootstrap script (migration 1).
func SQLiteInit() (string, error) {
	b, err := SQLiteFS.ReadFile("migrations/sqlite/0001_init.sql")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PostgresInit is the P0 PostgreSQL schema bootstrap script (migration 1).
func PostgresInit() (string, error) {
	b, err := PostgresFS.ReadFile("migrations/postgres/0001_init.sql")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
