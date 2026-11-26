package eval_test

import (
	"context"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/eval"
)

func TestRecallAt10(t *testing.T) {
	rep, err := eval.RunFile(context.Background(), eval.DefaultCorpusPath())
	if err != nil {
		t.Fatal(err)
	}
	if rep.RecallAt10 < 0.8 {
		for _, d := range rep.Details {
			if !d.Hit {
				t.Logf("miss: q=%q want=%v got=%v", d.Query, d.Want, d.GotIDs)
			}
		}
		t.Fatalf("recall@10=%.3f want >= 0.8 (hits=%d/%d)", rep.RecallAt10, rep.Hits, rep.Queries)
	}
}
