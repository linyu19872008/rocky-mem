package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rocky-mem/rocky-mem/internal/api"
	"github.com/rocky-mem/rocky-mem/internal/client"
	"github.com/rocky-mem/rocky-mem/internal/core"
	"github.com/rocky-mem/rocky-mem/internal/store/sqlite"
	"github.com/rocky-mem/rocky-mem/internal/web"
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

func TestCookieAuthAndWebSession(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		t.Fatal(err)
	}
	token := "cookie-token"
	h := (&api.Server{Svc: core.NewService(st), Token: token, EnableWeb: true}).Handler()

	// HTML sets session cookie.
	req := httptest.NewRequest(http.MethodGet, "/review", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("review page %d", rr.Code)
	}
	cookies := rr.Result().Cookies()
	var session string
	for _, c := range cookies {
		if c.Name == web.SessionCookie {
			session = c.Value
		}
	}
	if session != token {
		t.Fatalf("cookie=%q want %q", session, token)
	}
	body := rr.Body.String()
	if strings.Contains(body, "const TOKEN") || strings.Contains(body, token) {
		snip := body
		if len(snip) > 200 {
			snip = snip[:200]
		}
		t.Fatalf("page must not embed raw token: %s", snip)
	}

	// API accepts cookie.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/review/list", strings.NewReader(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: token})
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("cookie api %d %s", rr2.Code, rr2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodGet, "/entry?uid=e_x", nil)
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK || !strings.Contains(rr3.Body.String(), "Loading") {
		t.Fatalf("entry page: %d", rr3.Code)
	}
}
