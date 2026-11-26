package store

import (
	"fmt"

	"github.com/rocky-mem/rocky-mem/internal/config"
)

// Opener opens a store given the ParseDSN "rest" (path or full postgres DSN).
type Opener func(rest string) (Store, error)

var openers = map[string]Opener{}

// RegisterOpener registers a backend. Called from sqlite/postgres init().
func RegisterOpener(driver string, fn Opener) {
	openers[driver] = fn
}

// OpenDSN parses dsn and opens the matching registered backend.
// Returns the Store, driver name ("sqlite"|"postgres"), and rest
// (filesystem path for sqlite, full DSN for postgres).
// Callers must import the backend packages (or rely on blank imports) so
// RegisterOpener runs.
func OpenDSN(dsn string) (Store, string, string, error) {
	driver, rest, err := config.ParseDSN(dsn)
	if err != nil {
		return nil, "", "", err
	}
	fn, ok := openers[driver]
	if !ok {
		return nil, driver, rest, fmt.Errorf("store driver %q is not registered (blank-import the backend)", driver)
	}
	st, err := fn(rest)
	if err != nil {
		return nil, driver, rest, err
	}
	return st, driver, rest, nil
}
