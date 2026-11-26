package termmap_test

import (
	"strings"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/termmap"
)

func TestExpandConfigMap(t *testing.T) {
	m, err := termmap.LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	got := m.Expand("滚动更新后配置丢失")
	var joined []string
	for _, e := range got {
		joined = append(joined, e.Query)
	}
	s := strings.Join(joined, " | ")
	if !strings.Contains(s, "ConfigMap") && !strings.Contains(strings.ToLower(s), "rolling") {
		t.Fatalf("expected expansion toward ConfigMap/rolling, got %v", joined)
	}
	if got[0].Weight != 1.0 {
		t.Fatalf("original weight: %v", got[0])
	}
}

func TestExpandNoMatch(t *testing.T) {
	m, err := termmap.LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	got := m.Expand("xyzzy unique nonsense")
	if len(got) != 1 || got[0].Query != "xyzzy unique nonsense" {
		t.Fatalf("%+v", got)
	}
}
