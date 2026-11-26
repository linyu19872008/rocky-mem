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
	"github.com/rocky-mem/rocky-mem/internal/store"
	_ "github.com/rocky-mem/rocky-mem/internal/store/postgres"
	_ "github.com/rocky-mem/rocky-mem/internal/store/sqlite"
	"github.com/spf13/cobra"
)

func cmdServe() *cobra.Command {
	var bind string
	var noWeb bool
	var rotateToken bool
	c := &cobra.Command{
		Use:   "serve",
		Short: "Run the local daemon (exclusive store owner + /api/v1)",
		RunE: func(cmd *cobra.Command, args []string) error {
			driver, rest, err := config.ParseDSN(flagDSN)
			if err != nil {
				return err
			}
			paths, err := daemon.PathsForDriver(driver, rest)
			if err != nil {
				return err
			}

			var unlock func() error
			if driver == "sqlite" {
				lock, err := daemon.Acquire(paths.Lock)
				if err != nil {
					return err
				}
				unlock = lock.Unlock
			} else {
				unlock = func() error { return nil }
			}
			defer unlock()

			st, _, _, err := store.OpenDSN(flagDSN)
			if err != nil {
				return err
			}
			defer st.Close()
			if _, err := st.EnsureLocalUser(context.Background()); err != nil {
				return err
			}

			if rotateToken {
				_ = os.Remove(paths.Token)
			}
			token, minted, err := daemon.LoadOrCreateToken(paths.Token)
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
			_ = daemon.WriteRegistry("http://"+addr, token, flagDSN)
			defer daemon.ClearRegistry()

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
			tokenNote := "reused"
			if minted {
				tokenNote = "created"
			}
			fmt.Fprintf(os.Stderr, "rockymem serve listening on %s\n  db: %s\n  token file (MCP/CLI): %s (%s)\n", baseURL, rest, paths.Token, tokenNote)
			fmt.Fprintf(os.Stderr, "  MCP: %s/mcp (Streamable HTTP, Bearer; stateless)\n", baseURL)
			fmt.Fprintf(os.Stderr, "  MCP: rockymem mcp (stdio proxy, no token argument needed)\n")
			fmt.Fprintf(os.Stderr, "  capabilities: GET %s/api/v1/doctor\n", baseURL)
			if !noWeb {
				fmt.Fprintf(os.Stderr, "  open in browser (cookie auth, no Bearer needed):\n")
				fmt.Fprintf(os.Stderr, "    %s/\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/review\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/entry\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/settings\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/overview\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/ingest\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/library\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/remember\n", baseURL)
				fmt.Fprintf(os.Stderr, "    %s/record\n", baseURL)
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
	c.Flags().BoolVar(&rotateToken, "rotate-token", false, "mint a new bearer token (invalidates long-lived MCP/CLI until reload)")
	return c
}
