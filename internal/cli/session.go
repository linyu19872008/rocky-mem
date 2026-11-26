package cli

import (
	"context"
	"fmt"

	"github.com/linyu19872008/rocky-mem/internal/client"
	"github.com/linyu19872008/rocky-mem/internal/config"
	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/daemon"
	"github.com/linyu19872008/rocky-mem/internal/store"
	_ "github.com/linyu19872008/rocky-mem/internal/store/postgres"
	_ "github.com/linyu19872008/rocky-mem/internal/store/sqlite"
)

type session struct {
	API    core.MemAPI
	Store  store.Store // non-nil only in local mode
	close  func() error
	Remote bool
}

func (s *session) Close() error {
	if s == nil || s.close == nil {
		return nil
	}
	return s.close()
}

// openSession prefers a healthy daemon; otherwise opens the store locally.
// SQLite takes a flock; postgres skips flock and uses hashed discovery paths.
func openSession() (*session, error) {
	// No explicit DSN: prefer the daemon that is actually running, wherever it
	// keeps its database. Without this, `rockymem mcp` opens the default store
	// beside the default DSN while a daemon serves a different one.
	if flagDSN == "" {
		if r, ok := daemon.ReadRegistry(); ok {
			if sess := dialDaemon(r.URL, r.Token); sess != nil {
				return sess, nil
			}
		}
	}
	driver, rest, err := config.ParseDSN(flagDSN)
	if err != nil {
		return nil, err
	}
	if base, tok, ok := daemon.ResolveURLForDriver(driver, rest); ok {
		if sess := dialDaemon(base, tok); sess != nil {
			return sess, nil
		}
		if base2, tok2, ok2 := daemon.ResolveURLForDriver(driver, rest); ok2 && (base2 != base || tok2 != tok) {
			if sess := dialDaemon(base2, tok2); sess != nil {
				return sess, nil
			}
		}
	}

	paths, err := daemon.PathsForDriver(driver, rest)
	if err != nil {
		return nil, err
	}

	var unlock func() error
	if driver == "sqlite" {
		lock, err := daemon.Acquire(paths.Lock)
		if err != nil {
			return nil, fmt.Errorf("%w (or start `rockymem serve` and ensure discovery files are readable)", err)
		}
		unlock = lock.Unlock
	} else {
		unlock = func() error { return nil }
	}

	st, _, _, err := store.OpenDSN(flagDSN)
	if err != nil {
		_ = unlock()
		return nil, err
	}
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		_ = st.Close()
		_ = unlock()
		return nil, err
	}
	svc := core.NewService(st)
	return &session{
		API:   svc,
		Store: st,
		close: func() error {
			err1 := st.Close()
			err2 := unlock()
			if err1 != nil {
				return err1
			}
			return err2
		},
	}, nil
}

func dialDaemon(base, tok string) *session {
	c := client.New(base, tok)
	ctx, cancel := context.WithTimeout(context.Background(), client.DialTimeout)
	defer cancel()
	if err := c.Health(ctx); err != nil {
		return nil
	}
	return &session{API: c, Remote: true, close: func() error { return nil }}
}
