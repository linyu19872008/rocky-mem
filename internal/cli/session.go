package cli

import (
	"context"
	"fmt"

	"github.com/rocky-mem/rocky-mem/internal/client"
	"github.com/rocky-mem/rocky-mem/internal/config"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/daemon"
	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
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

// openSession prefers a healthy daemon; otherwise opens SQLite under flock.
func openSession() (*session, error) {
	_, path, err := config.ParseDSN(flagDSN)
	if err != nil {
		return nil, err
	}
	if base, tok, ok := daemon.ResolveURL(path); ok {
		c := client.New(base, tok)
		ctx, cancel := context.WithTimeout(context.Background(), client.DialTimeout)
		err := c.Health(ctx)
		cancel()
		if err == nil {
			return &session{API: c, Remote: true, close: func() error { return nil }}, nil
		}
	}

	paths := daemon.PathsForDB(path)
	lock, err := daemon.Acquire(paths.Lock)
	if err != nil {
		return nil, fmt.Errorf("%w (or start `rockymem serve` and ensure discovery files are readable)", err)
	}
	st, err := sqlite.Open(path)
	if err != nil {
		_ = lock.Unlock()
		return nil, err
	}
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		_ = st.Close()
		_ = lock.Unlock()
		return nil, err
	}
	svc := core.NewService(st)
	return &session{
		API:   svc,
		Store: st,
		close: func() error {
			err1 := st.Close()
			err2 := lock.Unlock()
			if err1 != nil {
				return err1
			}
			return err2
		},
	}, nil
}
