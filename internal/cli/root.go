// Package cli implements the rockymem command line.
package cli

import (
	"fmt"
	"os"

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
		cmdInit(), cmdServe(), cmdMCP(), cmdSearch(), cmdShow(), cmdRemember(), cmdRecord(),
		cmdFeedback(), cmdReview(), cmdReindex(), cmdPurge(), cmdExport(), cmdEval(),
		cmdAdoptVerify(), cmdDoctor(), cmdVersion(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("rockymem 0.1.0-tier1")
		},
	}
}
