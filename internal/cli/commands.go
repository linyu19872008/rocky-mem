package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/linyu19872008/rocky-mem/internal/client"
	"github.com/linyu19872008/rocky-mem/internal/config"
	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/mcpserver"
	"github.com/linyu19872008/rocky-mem/internal/store"
	_ "github.com/linyu19872008/rocky-mem/internal/store/postgres"
	_ "github.com/linyu19872008/rocky-mem/internal/store/sqlite"
	"github.com/spf13/cobra"
)

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialise the store (sqlite or postgres) and print its path/DSN",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, driver, rest, err := store.OpenDSN(flagDSN)
			if err != nil {
				return err
			}
			defer st.Close()
			if _, err := st.EnsureLocalUser(context.Background()); err != nil {
				return err
			}
			home, _ := config.Home()
			fmt.Printf("initialised %s store\n  dsn: %s\n  home: %s\n", driver, rest, home)
			return nil
		},
	}
}

func cmdMCP() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP server on stdio (for Cursor / other clients)",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			server := mcpserver.New(sess.API)
			return server.Run(context.Background(), &mcp.StdioTransport{})
		},
	}
}

func cmdSearch() *cobra.Command {
	var cwd string
	var asJSON bool
	c := &cobra.Command{
		Use:   "search <query>",
		Short: "Search memory and experience",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			out, err := sess.API.Search(context.Background(), core.SearchInput{
				Query: strings.Join(args, " "), Cwd: cwd, IncludeProposed: true,
			})
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			fmt.Print(out.Pack)
			return nil
		},
	}
	c.Flags().StringVar(&cwd, "cwd", "", "working directory for project scope")
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return c
}

func cmdRemember() *cobra.Command {
	var memKey, memType, scope, cwd string
	var pinned bool
	c := &cobra.Command{
		Use:   "remember <title> <body>",
		Short: "Write a standing memory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			out, err := sess.API.Remember(context.Background(), core.RememberInput{
				Title: args[0], Body: args[1], MemKey: memKey, MemType: memType,
				Scope: scope, Pinned: pinned, Cwd: cwd,
			})
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&memKey, "key", "", "stable mem_key for upsert")
	c.Flags().StringVar(&memType, "type", "pref", "pref|fact|constraint|glossary")
	c.Flags().StringVar(&scope, "scope", "", "global|project|session")
	c.Flags().StringVar(&cwd, "cwd", "", "working directory")
	c.Flags().BoolVar(&pinned, "pinned", false, "pin in context packs")
	return c
}

func cmdRecord() *cobra.Command {
	var expType, symptom, root, fix, applies, invalid, outcome, cwd, evidence, scope string
	var confidence float64
	c := &cobra.Command{
		Use:   "record <title>",
		Short: "Record an experience (requires --evidence)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if evidence == "" {
				return fmt.Errorf("--evidence is required")
			}
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			if cwd == "" {
				cwd, _ = os.Getwd()
			}
			if expType == "" {
				expType = "pitfall"
			}
			out, err := sess.API.Record(context.Background(), core.RecordInput{
				Title: args[0], ExpType: expType, Symptom: symptom, RootCause: root,
				Fix: fix, AppliesWhen: applies, InvalidWhen: invalid, Outcome: outcome,
				Confidence: confidence, Scope: scope, Cwd: cwd,
				Evidence: []core.EvidenceIn{{Type: "quote", Text: evidence}},
			})
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&expType, "type", "pitfall", "howto|pitfall|decision|postmortem")
	c.Flags().StringVar(&symptom, "symptom", "", "observable symptom")
	c.Flags().StringVar(&root, "root-cause", "", "root cause")
	c.Flags().StringVar(&fix, "fix", "", "fix markdown")
	c.Flags().StringVar(&applies, "applies-when", "", "when this applies")
	c.Flags().StringVar(&invalid, "invalid-when", "", "when this does not apply")
	c.Flags().StringVar(&outcome, "outcome", "success", "success|failed|unknown")
	c.Flags().Float64Var(&confidence, "confidence", 0.8, "0..1")
	c.Flags().StringVar(&evidence, "evidence", "", "verbatim evidence quote")
	c.Flags().StringVar(&scope, "scope", "", "global|project|session (default: project from --cwd)")
	c.Flags().StringVar(&cwd, "cwd", "", "working directory")
	return c
}

func cmdShow() *cobra.Command {
	var includeRaw bool
	c := &cobra.Command{
		Use:   "show <uid>",
		Short: "Show one memory or experience (mem_get)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.Get(context.Background(), core.GetInput{UID: args[0], IncludeRaw: includeRaw})
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().BoolVar(&includeRaw, "raw", false, "include evidence excerpts")
	return c
}

func cmdFeedback() *cobra.Command {
	var outcome, note string
	c := &cobra.Command{
		Use:   "feedback <uid>",
		Short: "Report helpful|wrong|stale on an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if outcome == "" {
				return fmt.Errorf("--outcome is required")
			}
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.Feedback(context.Background(), core.FeedbackInput{
				UID: args[0], Outcome: outcome, Note: note,
			})
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&outcome, "outcome", "", "helpful|wrong|stale")
	c.Flags().StringVar(&note, "note", "", "required when outcome=wrong")
	return c
}

func cmdDoctor() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Integrity and environment diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			var rep store.DoctorReport
			if sess.Remote {
				c, ok := sess.API.(*client.Client)
				if !ok {
					return fmt.Errorf("remote doctor unavailable")
				}
				rep, err = c.Doctor(context.Background())
			} else {
				rep, err = sess.Store.Doctor(context.Background())
			}
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rep)
		},
	}
}
