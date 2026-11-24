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

// Migration is one ordered SQL script.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// SQLiteMigrations returns migrations sorted by version ascending.
func SQLiteMigrations() ([]Migration, error) {
	entries, err := SQLiteFS.ReadDir("migrations/sqlite")
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
		body, err := SQLiteFS.ReadFile("migrations/sqlite/" + e.Name())
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
