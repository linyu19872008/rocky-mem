package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/api"
	"github.com/rocky-mem/rocky-mem/internal/config"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/daemon"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
	"github.com/spf13/cobra"
)

func cmdServe() *cobra.Command {
	var bind string
	var noWeb bool
	c := &cobra.Command{
		Use:   "serve",
		Short: "Run the local daemon (exclusive store owner + /api/v1)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := config.ParseDSN(flagDSN)
			if err != nil {
				return err
			}
			paths := daemon.PathsForDB(path)
			lock, err := daemon.Acquire(paths.Lock)
			if err != nil {
				return err
			}
			defer lock.Unlock()

			st, err := sqlite.Open(path)
			if err != nil {
				return err
			}
			defer st.Close()
			if _, err := st.EnsureLocalUser(context.Background()); err != nil {
				return err
			}

			token, err := daemon.LoadOrCreateToken(paths.Token)
			if err != nil {
				return err
			}
			ln, err := net.Listen("tcp", bind)
			if err != nil {
				return err
			}
			addr := ln.Addr().String()
			if err := daemon.WriteDiscovery(paths, addr, token); err != nil {
				_ = ln.Close()
				return err
			}
			defer daemon.ClearDiscovery(paths)

			svc := core.NewService(st)
			handler := (&api.Server{Svc: svc, Token: token, EnableWeb: !noWeb}).Handler()
			srv := &http.Server{
				Handler:           handler,
				ReadHeaderTimeout: 5 * time.Second,
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.Serve(ln)
			}()

			baseURL := "http://" + addr
			fmt.Fprintf(os.Stderr, "rockymem serve listening on %s\n  db: %s\n  token file (MCP/CLI): %s\n", baseURL, path, paths.Token)
			if !noWeb {
				fmt.Fprintf(os.Stderr, "  open in browser (cookie auth, no Bearer needed):\n")
				fmt.Fprintf(os.Stderr, "    %s/\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/review\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/entry\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/settings\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/overview\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/ingest   (stub)\n", baseURL)
			}

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			select {
			case <-sig:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(ctx)
				return nil
			case err := <-errCh:
				if err == http.ErrServerClosed {
					return nil
				}
				return err
			}
		},
	}
	c.Flags().StringVar(&bind, "bind", daemon.DefaultBind, "loopback listen address")
	c.Flags().BoolVar(&noWeb, "no-web", false, "disable embedded search/review UI")
	return c
}
