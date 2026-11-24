package redact_test

import (
	"strings"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/redact"
)

func TestApplyUnchanged(t *testing.T) {
	r := redact.Apply("hello overtime sheet 林宇")
	if !r.Unchanged || len(r.Spans) != 0 {
		t.Fatalf("%+v", r)
	}
	if len(r.Segments) != 1 || r.Segments[0].Kind != "eq" {
		t.Fatalf("segments %+v", r.Segments)
	}
}

func TestApplySpansAndSegments(t *testing.T) {
	in := "note api_key=supersecrettoken123 end"
	r := redact.Apply(in)
	if r.Unchanged || r.Hits["generic_secret"] < 1 {
		t.Fatalf("%+v", r)
	}
	if strings.Contains(r.Text, "supersecrettoken123") {
		t.Fatalf("leaked: %s", r.Text)
	}
	if len(r.Spans) < 1 || r.Spans[0].Original == "" || r.Spans[0].Replacement == "" {
		t.Fatalf("spans %+v", r.Spans)
	}
	var kinds []string
	for _, s := range r.Segments {
		kinds = append(kinds, s.Kind)
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, "del") || !strings.Contains(joined, "ins") {
		t.Fatalf("kinds %s", joined)
	}
}
