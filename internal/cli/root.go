// Package cli implements the rockymem command line.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/rocky-mem/rocky-mem/internal/config"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
	"github.com/spf13/cobra"
)

var (
	flagDSN string
)

// Execute runs the root command.
func Execute() {
	root := &cobra.Command{
		Use:   "rockymem",
		Short: "Local-first memory and experience service for coding agents",
	}
	root.PersistentFlags().StringVar(&flagDSN, "dsn", "", "sqlite://path (default: $ROCKYMEM_HOME/rocky-mem.db)")

	root.AddCommand(
		cmdInit(), cmdMCP(), cmdSearch(), cmdShow(), cmdRemember(), cmdRecord(),
		cmdFeedback(), cmdAdoptVerify(), cmdDoctor(), cmdVersion(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openStore() (store.Store, error) {
	_, path, err := config.ParseDSN(flagDSN)
	if err != nil {
		return nil, err
	}
	return sqlite.Open(path)
}

func openService() (*core.Service, store.Store, error) {
	st, err := openStore()
	if err != nil {
		return nil, nil, err
	}
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	return &core.Service{Store: st}, st, nil
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("rockymem 0.1.0-p0b")
		},
	}
}
