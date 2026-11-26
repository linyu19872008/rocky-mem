package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linyu19872008/rocky-mem/internal/api"
	"github.com/linyu19872008/rocky-mem/internal/client"
	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/store/sqlite"
	"github.com/linyu19872008/rocky-mem/internal/web"
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

	stats, err := c.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ActiveMemories < 1 {
		t.Fatalf("stats memories: %+v", stats)
	}
	if !stats.DoctorOK {
		t.Fatalf("expected doctor_ok after reindex: %+v", stats)
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
	body3 := rr3.Body.String()
	if rr3.Code != http.StatusOK || !strings.Contains(body3, `id="root"`) || !strings.Contains(body3, "__ROCKY_BOOT__") {
		t.Fatalf("entry spa shell: %d", rr3.Code)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr4 := httptest.NewRecorder()
	h.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK || !strings.Contains(rr4.Body.String(), `id="root"`) {
		t.Fatalf("settings spa shell: %d", rr4.Code)
	}

	// Locale from Accept-Language is injected into boot JSON.
	req5 := httptest.NewRequest(http.MethodGet, "/", nil)
	req5.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	rr5 := httptest.NewRecorder()
	h.ServeHTTP(rr5, req5)
	if !strings.Contains(rr5.Body.String(), `"locale":"zh-CN"`) {
		t.Fatalf("expected zh-CN boot locale, got: %s", rr5.Body.String()[:min(200, rr5.Body.Len())])
	}
}

func TestDoctorPublishesContractAndFeatures(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := (&api.Server{Svc: &core.Service{Store: st}, Token: "t"}).Handler()
	ts := httptest.NewServer(h)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/doctor", nil)
	req.Header.Set("Authorization", "Bearer t")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body struct {
		Contract int      `json:"contract"`
		Features []string `json:"features"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Contract < 1 {
		t.Fatalf("contract = %d", body.Contract)
	}
	want := map[string]bool{"context": false, "search": false, "mcp.streamable-http": false}
	for _, f := range body.Features {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("feature %q missing from %v", name, body.Features)
		}
	}
}

func TestMCPStreamableHTTPHandshake(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlite.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := (&api.Server{Svc: &core.Service{Store: st}, Token: "t"}).Handler()
	ts := httptest.NewServer(h)
	defer ts.Close()

	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`

	// Without a token the transport must not answer.
	res, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(init))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", res.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(init))
	req.Header.Set("Authorization", "Bearer t")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d", res.StatusCode)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"serverInfo"`) || !strings.Contains(string(raw), "rocky-mem") {
		t.Fatalf("unexpected initialize body: %s", raw)
	}

	// Stateless mode: GET is not a stream.
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	req.Header.Set("Authorization", "Bearer t")
	req.Header.Set("Accept", "text/event-stream")
	res2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET /mcp status = %d", res2.StatusCode)
	}
}

func TestContractDocListsEveryFeature(t *testing.T) {
	doc, err := os.ReadFile("../../docs/contract.md")
	if err != nil {
		t.Fatalf("read contract doc: %v", err)
	}
	text := string(doc)
	for _, feature := range api.Features {
		if !strings.Contains(text, feature) {
			t.Errorf("docs/contract.md does not mention feature %q", feature)
		}
	}
	if !strings.Contains(text, "contract") {
		t.Error("docs/contract.md must explain the contract field")
	}
}
