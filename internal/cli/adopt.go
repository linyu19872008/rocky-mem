package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rocky-mem/rocky-mem/internal/adoption"
	"github.com/rocky-mem/rocky-mem/internal/config"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store"
	_ "github.com/rocky-mem/rocky-mem/internal/store/postgres"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
	"github.com/spf13/cobra"
)

func cmdAdoptVerify() *cobra.Command {
	var writeVerdict bool
	var asJSON bool
	c := &cobra.Command{
		Use:   "adopt-verify",
		Short: "Tier-0 adoption gate: protocol files + workflow + audit verdict",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := adoption.FindRepoRoot(wd)
			if err != nil {
				return err
			}

			protocolErr := adoption.CheckProtocolFiles(root)
			protocolOK := protocolErr == nil

			driver, path, err := config.ParseDSN(flagDSN)
			if err != nil {
				return err
			}
			// Prefer project .local DB when --dsn unset and it exists (sqlite only).
			if flagDSN == "" {
				candidate := filepath.Join(root, ".local", "rocky-mem.db")
				if _, err := os.Stat(candidate); err == nil {
					driver, path = "sqlite", candidate
				}
			}
			var st store.Store
			if driver == "postgres" {
				st, _, _, err = store.OpenDSN(flagDSN)
			} else {
				st, err = sqlite.Open(path)
			}
			if err != nil {
				return err
			}
			defer st.Close()
			ctx := context.Background()
			if _, err := st.EnsureLocalUser(ctx); err != nil {
				return err
			}
			svc := &core.Service{Store: st}

			since := time.Now().Unix()
			wf, wfErr := adoption.RunWorkflow(ctx, svc, root, fmt.Sprintf("adopt-verify-%d", since))
			workflowOK := wfErr == nil

			v, err := adoption.LoadAndAnalyze(ctx, st, since-1, protocolOK, workflowOK)
			if err != nil {
				return err
			}
			if protocolErr != nil {
				v.Notes = append(v.Notes, "protocol: "+protocolErr.Error())
			}
			if wfErr != nil {
				v.Notes = append(v.Notes, "workflow: "+wfErr.Error())
			}
			if wf != nil {
				v.Notes = append(v.Notes, fmt.Sprintf("workflow steps: %v", wf.Steps))
				v.Notes = append(v.Notes, fmt.Sprintf("record_uid=%s remember_uid=%s", wf.RecordUID, wf.RememberUID))
			}

			if writeVerdict {
				outPath := filepath.Join(root, "docs", "adoption-verdict.md")
				body := adoption.RenderVerdictMarkdown(v)
				body += "\n## Last automated run\n\n"
				body += fmt.Sprintf("```\n%s\n```\n", mustJSON(v))
				if err := os.WriteFile(outPath, []byte(body), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "wrote %s\n", outPath)
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(v); err != nil {
					return err
				}
			} else {
				fmt.Print(adoption.RenderVerdictMarkdown(v))
			}

			if v.Status != "pass" {
				return fmt.Errorf("tier-0 adoption verify status=%s", v.Status)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&writeVerdict, "write-verdict", false, "write docs/adoption-verdict.md")
	c.Flags().BoolVar(&asJSON, "json", false, "print machine-readable verdict")
	return c
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
