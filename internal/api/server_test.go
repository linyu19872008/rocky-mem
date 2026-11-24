package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/api"
	"github.com/rocky-mem/rocky-mem/internal/client"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
)

func TestServeClientRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if _, err := st.EnsureLocalUser(ctx); err != nil {
		t.Fatal(err)
	}
	svc := &core.Service{Store: st}
	token := "test-token"
	h := (&api.Server{Svc: svc, Token: token}).Handler()
	ts := httptest.NewServer(h)
	defer ts.Close()

	c := client.New(ts.URL, token)
	if err := c.Health(ctx); err != nil {
		t.Fatal(err)
	}
	mem, err := c.Remember(ctx, core.RememberInput{
		Title: "pref", Body: "dark mode", MemKey: "theme", Scope: "global", MemType: "pref",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mem.UID == "" {
		t.Fatal("empty uid")
	}
	pack, err := c.Search(ctx, core.SearchInput{Query: "dark mode", IncludeProposed: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.UIDs) == 0 {
		t.Fatalf("empty search: %+v", pack)
	}
	ri, err := c.Reindex(ctx)
	if err != nil || ri.MemoryIndexed < 1 {
		t.Fatalf("%v %+v", err, ri)
	}

	// unauthorized
	bad := client.New(ts.URL, "wrong")
	if err := bad.Health(ctx); err == nil {
		t.Fatal("expected auth failure")
	}
}

func TestAuthRequired(t *testing.T) {
	h := (&api.Server{Svc: &core.Service{}, Token: "x"}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rr.Code)
	}
}
