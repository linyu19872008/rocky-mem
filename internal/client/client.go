// Package client talks to rockymem serve over loopback HTTP.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/linyu19872008/rocky-mem/internal/core"
	"github.com/linyu19872008/rocky-mem/internal/errs"
	"github.com/linyu19872008/rocky-mem/internal/store"
)

// DialTimeout is used when probing daemon health.
const DialTimeout = 2 * time.Second

// Client is a remote MemAPI.
type Client struct {
	Base  string
	Token string
	HTTP  *http.Client
}

// New builds a client for base URL (no trailing slash) and bearer token.
func New(base, token string) *Client {
	return &Client{
		Base:  base,
		Token: token,
		HTTP:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Health(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, "/health", nil, nil)
	return err
}

func (c *Client) Doctor(ctx context.Context) (store.DoctorReport, error) {
	var out store.DoctorReport
	_, err := c.do(ctx, http.MethodGet, "/api/v1/doctor", nil, &out)
	return out, err
}

func (c *Client) Stats(ctx context.Context) (*core.StatsResult, error) {
	var out core.StatsResult
	_, err := c.do(ctx, http.MethodGet, "/api/v1/stats", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Context(ctx context.Context, in core.ContextInput) (*core.PackResult, error) {
	var out core.PackResult
	if err := c.post(ctx, "/api/v1/context", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Search(ctx context.Context, in core.SearchInput) (*core.PackResult, error) {
	var out core.PackResult
	if err := c.post(ctx, "/api/v1/search", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Get(ctx context.Context, in core.GetInput) (*core.GetResult, error) {
	var out core.GetResult
	if err := c.post(ctx, "/api/v1/get", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Remember(ctx context.Context, in core.RememberInput) (*core.RememberResult, error) {
	var out core.RememberResult
	if err := c.post(ctx, "/api/v1/remember", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Record(ctx context.Context, in core.RecordInput) (*core.RecordResult, error) {
	var out core.RecordResult
	if err := c.post(ctx, "/api/v1/record", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Feedback(ctx context.Context, in core.FeedbackInput) (*core.FeedbackResult, error) {
	var out core.FeedbackResult
	if err := c.post(ctx, "/api/v1/feedback", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ReviewList(ctx context.Context, trusts []string, limit int) ([]core.ReviewItem, error) {
	body := map[string]any{"trusts": trusts, "limit": limit}
	var out []core.ReviewItem
	if err := c.post(ctx, "/api/v1/review/list", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ReviewApprove(ctx context.Context, uid string) (*core.ReviewResult, error) {
	var out core.ReviewResult
	if err := c.post(ctx, "/api/v1/review/approve", map[string]string{"uid": uid}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ReviewReject(ctx context.Context, uid, reason string) (*core.ReviewResult, error) {
	var out core.ReviewResult
	if err := c.post(ctx, "/api/v1/review/reject", map[string]string{"uid": uid, "reason": reason}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ReviewResolve(ctx context.Context, in core.ReviewResolveInput) (*core.ReviewResolveResult, error) {
	var out core.ReviewResolveResult
	if err := c.post(ctx, "/api/v1/review/resolve", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) IngestPreview(ctx context.Context, in core.IngestPreviewInput) (*core.IngestPreviewResult, error) {
	var out core.IngestPreviewResult
	if err := c.post(ctx, "/api/v1/ingest/preview", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) IngestConfirm(ctx context.Context, in core.IngestConfirmInput) (*core.IngestConfirmResult, error) {
	var out core.IngestConfirmResult
	if err := c.post(ctx, "/api/v1/ingest/confirm", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Reindex(ctx context.Context) (*core.ReindexResult, error) {
	var out core.ReindexResult
	if err := c.post(ctx, "/api/v1/reindex", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Purge(ctx context.Context, in core.PurgeInput) (*core.PurgeResult, error) {
	var out core.PurgeResult
	if err := c.post(ctx, "/api/v1/purge", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ExportL1(ctx context.Context, format string) (*core.ExportResult, error) {
	var out core.ExportResult
	if err := c.post(ctx, "/api/v1/export", map[string]string{"format": format}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Backup(ctx context.Context, dest string) (*core.BackupResult, error) {
	var out core.BackupResult
	if err := c.post(ctx, "/api/v1/backup", map[string]string{"path": dest}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) post(ctx context.Context, path string, in, out any) error {
	_, err := c.do(ctx, http.MethodPost, path, in, out)
	return err
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) (int, error) {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, body)
	if err != nil {
		return 0, err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode >= 400 {
		var wrap struct {
			Error *errs.E `json:"error"`
		}
		if json.Unmarshal(raw, &wrap) == nil && wrap.Error != nil {
			return res.StatusCode, wrap.Error
		}
		return res.StatusCode, fmt.Errorf("http %d: %s", res.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return res.StatusCode, err
		}
	}
	return res.StatusCode, nil
}
