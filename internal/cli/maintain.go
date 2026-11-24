package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/config"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/daemon"
	"github.com/rocky-mem/rocky-mem/internal/eval"
	"github.com/spf13/cobra"
)

func cmdReview() *cobra.Command {
	root := &cobra.Command{
		Use:   "review",
		Short: "List / approve / reject proposed experiences",
	}
	root.AddCommand(cmdReviewList(), cmdReviewApprove(), cmdReviewReject())
	return root
}

func cmdReviewList() *cobra.Command {
	var trust string
	var limit int
	c := &cobra.Command{
		Use:   "list",
		Short: "List proposed/contested experiences",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			var trusts []string
			if trust != "" {
				trusts = strings.Split(trust, ",")
			}
			out, err := sess.API.ReviewList(context.Background(), trusts, limit)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&trust, "trust", "proposed,contested", "comma-separated trust values")
	c.Flags().IntVar(&limit, "limit", 50, "max rows")
	return c
}

func cmdReviewApprove() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <uid>",
		Short: "Promote proposed/contested → active",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.ReviewApprove(context.Background(), args[0])
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
}

func cmdReviewReject() *cobra.Command {
	var reason string
	c := &cobra.Command{
		Use:   "reject <uid>",
		Short: "Archive an experience (human rejection)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.ReviewReject(context.Background(), args[0], reason)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&reason, "reason", "", "rejection note")
	return c
}

func cmdReindex() *cobra.Command {
	return &cobra.Command{
		Use:   "reindex",
		Short: "Rebuild search_doc from memory/experience tables",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.Reindex(context.Background())
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
}

func cmdPurge() *cobra.Command {
	var uid, project, beforeStr string
	c := &cobra.Command{
		Use:   "purge",
		Short: "Delete entries by uid, age, or project scope_ref",
		RunE: func(cmd *cobra.Command, args []string) error {
			var before int64
			if beforeStr != "" {
				if strings.HasSuffix(beforeStr, "d") {
					days, err := strconv.Atoi(strings.TrimSuffix(beforeStr, "d"))
					if err != nil {
						return fmt.Errorf("invalid --before: %w", err)
					}
					before = time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()
				} else {
					n, err := strconv.ParseInt(beforeStr, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid --before: %w", err)
					}
					before = n
				}
			}
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.Purge(context.Background(), core.PurgeInput{
				UID: uid, Before: before, Project: project,
			})
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&uid, "uid", "", "m_… or e_…")
	c.Flags().StringVar(&project, "project", "", "project scope_ref (numeric id)")
	c.Flags().StringVar(&beforeStr, "before", "", "unix seconds or Nd (e.g. 30d)")
	return c
}

func cmdExport() *cobra.Command {
	var format, outPath string
	c := &cobra.Command{
		Use:   "export",
		Short: "Export L1 methodology (no raw/evidence)",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			out, err := sess.API.ExportL1(context.Background(), format)
			if err != nil {
				return err
			}
			if outPath != "" {
				return os.WriteFile(outPath, []byte(out.Body), 0o600)
			}
			fmt.Print(out.Body)
			return nil
		},
	}
	c.Flags().StringVar(&format, "format", "md", "md|jsonl")
	c.Flags().StringVar(&outPath, "out", "", "write to file instead of stdout")
	return c
}

func cmdBackup() *cobra.Command {
	var dest, dir string
	var retain int
	c := &cobra.Command{
		Use:   "backup",
		Short: "Snapshot the store with VACUUM INTO",
		RunE: func(cmd *cobra.Command, args []string) error {
			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			ctx := context.Background()
			var out *core.BackupResult
			if dest != "" {
				out, err = sess.API.Backup(ctx, dest)
			} else {
				if dir == "" {
					home, err := config.Home()
					if err != nil {
						return err
					}
					dir = filepath.Join(home, "backups")
				}
				// Prefer BackupWithRetain when talking to a local Service.
				if svc, ok := sess.API.(*core.Service); ok {
					out, err = svc.BackupWithRetain(ctx, dir, retain)
				} else {
					path := core.DefaultBackupPath(dir)
					out, err = sess.API.Backup(ctx, path)
					if err == nil && retain > 0 {
						pruned, perr := core.PruneBackups(dir, retain)
						if perr != nil {
							return perr
						}
						out.Pruned = pruned
						out.RetainKeep = retain
					}
				}
			}
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	c.Flags().StringVar(&dest, "out", "", "exact destination .db path (skips --dir/--retain prune naming)")
	c.Flags().StringVar(&dir, "dir", "", "backup directory (default: $ROCKYMEM_HOME/backups)")
	c.Flags().IntVar(&retain, "retain", 0, "keep newest N rocky-mem-*.db in --dir (0 = keep all)")
	return c
}

func cmdRestore() *cobra.Command {
	c := &cobra.Command{
		Use:   "restore <snapshot.db>",
		Short: "Replace the store with a snapshot (serve must be stopped)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := config.ParseDSN(flagDSN)
			if err != nil {
				return err
			}
			if base, tok, ok := daemon.ResolveURL(path); ok {
				_ = base
				_ = tok
				return fmt.Errorf("refuse restore while serve appears running; stop serve first (discovery present)")
			}
			paths := daemon.PathsForDB(path)
			lock, err := daemon.Acquire(paths.Lock)
			if err != nil {
				return err
			}
			defer lock.Unlock()
			if err := core.RestoreFile(args[0], path); err != nil {
				return err
			}
			fmt.Printf("restored %s → %s\n", args[0], path)
			return nil
		},
	}
	return c
}

func cmdEval() *cobra.Command {
	var corpus string
	var minRecall float64
	c := &cobra.Command{
		Use:   "eval",
		Short: "Run retrieval eval corpus and print recall@10",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := corpus
			if path == "" {
				path = eval.DefaultCorpusPath()
			}
			rep, err := eval.RunFile(context.Background(), path)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(rep); err != nil {
				return err
			}
			if rep.RecallAt10 < minRecall {
				return fmt.Errorf("recall@10=%.3f below threshold %.3f", rep.RecallAt10, minRecall)
			}
			return nil
		},
	}
	c.Flags().StringVar(&corpus, "corpus", "", "path to eval corpus JSON")
	c.Flags().Float64Var(&minRecall, "min-recall", 0.8, "fail if recall@10 below this")
	return c
}
