package core

import "context"

// MemAPI is the application surface shared by local Service and HTTP client.
type MemAPI interface {
	Context(ctx context.Context, in ContextInput) (*PackResult, error)
	Search(ctx context.Context, in SearchInput) (*PackResult, error)
	Get(ctx context.Context, in GetInput) (*GetResult, error)
	Remember(ctx context.Context, in RememberInput) (*RememberResult, error)
	Record(ctx context.Context, in RecordInput) (*RecordResult, error)
	Feedback(ctx context.Context, in FeedbackInput) (*FeedbackResult, error)
	ReviewList(ctx context.Context, trusts []string, limit int) ([]ReviewItem, error)
	ReviewApprove(ctx context.Context, uid string) (*ReviewResult, error)
	ReviewReject(ctx context.Context, uid, reason string) (*ReviewResult, error)
	Reindex(ctx context.Context) (*ReindexResult, error)
	Purge(ctx context.Context, in PurgeInput) (*PurgeResult, error)
	ExportL1(ctx context.Context, format string) (*ExportResult, error)
}
