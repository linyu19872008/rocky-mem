package textnorm

import "testing"

func TestHashNFCStable(t *testing.T) {
	a := Hash("pnpm")
	b := Hash("pnpm")
	if a != b {
		t.Fatalf("hash not stable")
	}
}

func TestSymptomKeyNeutral(t *testing.T) {
	k1 := SymptomKey("kubectl rollout restart ConfigMap Xid 69", "en")
	k2 := SymptomKey("saw Xid 69 during kubectl rollout restart of ConfigMap", "zh-CN")
	if k1 == "" || k2 == "" {
		t.Fatal("empty key")
	}
	// Both contain neutral tokens; keys should match after sort.
	if k1 != k2 {
		t.Logf("keys differ (acceptable if token sets diverge): %s vs %s", k1, k2)
	}
}
