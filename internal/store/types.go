// Package store defines the storage interface shared by backends.
package store

import "context"

// Candidate is a retrieval hit. Score is comparable only within one call.
type Candidate struct {
	Kind    string  // memory | experience
	OwnerID int64
	Score   float64
}

// SearchDoc is the materialised retrieval document.
type SearchDoc struct {
	Kind     string
	OwnerID  int64
	Lang     string
	Title    string
	Body     string
	Updated  int64
}

// Memory is a state row.
type Memory struct {
	ID          int64
	UID         string
	Lineage     string
	Scope       string
	ScopeRef    string
	MemKey      *string
	MemType     string
	Title       string
	BodyMD      string
	Lang        string
	Pinned      bool
	Priority    int
	Status      string
	Sensitivity int
	ActorID     *int64
	Rev         int
	CreatedAt   int64
	TLastUsed   *int64
	UseCount    int
}

// Experience is a ledger row.
type Experience struct {
	ID            int64
	UID           string
	ExpType       string
	Scope         string
	ScopeRef      string
	Title         string
	Symptom       string
	SymptomKey    string
	RootCause     string
	FixMD         string
	FixHash       string
	AppliesWhen   string
	InvalidWhen   string
	Outcome       string
	Confidence    float64
	Trust         string
	ConflictGroup string
	Lang          string
	Sensitivity   int
	ActorID       *int64
	Rev           int
	TCreated      int64
	TLastUsed     *int64
	UseCount      int
}

// Raw is immutable evidence content.
type Raw struct {
	ID          int64
	UID         string
	Hash        string
	Channel     string
	SourceRef   string
	ActorID     *int64
	Agent       string
	SessionID   string
	Cwd         string
	ProjectID   *int64
	Lang        string
	Body        []byte
	Redaction   string
	RulePack    string
	Sensitivity int
	CreatedAt   int64
}

// Evidence links an owner to a raw row.
type Evidence struct {
	ID        int64
	OwnerType string
	OwnerID   int64
	RawID     int64
	Quote     string
	Role      string
	CreatedAt int64
}

// Project is a repository identity.
type Project struct {
	ID            int64
	Name          string
	RemoteURLNorm string
	RootHint      string
	CreatedBy     *int64
	CreatedAt     int64
}

// DoctorReport summarises integrity checks.
type DoctorReport struct {
	SchemaVersion   string   `json:"schema_version"`
	FTSAvailable    bool     `json:"fts_available"`
	FTSMode         string   `json:"fts_mode"`
	MemoryCount     int      `json:"memory_count"`
	ExperienceCount int      `json:"experience_count"`
	SearchDocOK     bool     `json:"search_doc_ok"`
	MemoryDocCount  int      `json:"memory_doc_count"`
	ActiveMemory    int      `json:"active_memory"`
	ExpDocCount     int      `json:"experience_doc_count"`
	IndexableExp    int      `json:"indexable_experience"`
	Notices         []string `json:"notices"`
}

// Store is the persistence boundary. Score from SearchCandidates is only
// meaningful within a single call — never compare across backends or sources.
type Store interface {
	Close() error
	Ping(ctx context.Context) error

	EnsureLocalUser(ctx context.Context) (userID int64, err error)
	UpsertProject(ctx context.Context, p Project) (Project, error)
	GetProject(ctx context.Context, id int64) (*Project, error)

	InsertRaw(ctx context.Context, r Raw) (Raw, error)
	GetRawByHash(ctx context.Context, hash string) (*Raw, error)
	GetRaw(ctx context.Context, id int64) (*Raw, error)

	InsertMemory(ctx context.Context, m Memory) (Memory, error)
	GetMemoryByUID(ctx context.Context, uid string) (*Memory, error)
	GetActiveMemoryByLineage(ctx context.Context, lineage string) (*Memory, error)
	ListMemoryVersions(ctx context.Context, lineage string) ([]Memory, error)
	GetActiveMemoryByKey(ctx context.Context, scope, scopeRef, key string) (*Memory, error)
	ListActiveMemory(ctx context.Context, scopes []ScopeFilter, limit int) ([]Memory, error)
	SupersedeMemory(ctx context.Context, oldID int64, expectedRev int, next Memory) (Memory, error)

	InsertExperience(ctx context.Context, e Experience) (Experience, error)
	GetExperienceByUID(ctx context.Context, uid string) (*Experience, error)
	GetExperience(ctx context.Context, id int64) (*Experience, error)
	ListExperiencesByIDs(ctx context.Context, ids []int64) ([]Experience, error)
	ListMemoriesByIDs(ctx context.Context, ids []int64) ([]Memory, error)
	FindExperiencesBySymptomKey(ctx context.Context, key string) ([]Experience, error)
	FindExperiencesByConflictGroup(ctx context.Context, group string) ([]Experience, error)
	MarkContested(ctx context.Context, ids []int64, group string) error
	UpdateExperienceTrust(ctx context.Context, id int64, trust string) error

	InsertEvidence(ctx context.Context, ev Evidence) (Evidence, error)
	ListEvidence(ctx context.Context, ownerType string, ownerID int64) ([]Evidence, error)

	SetTags(ctx context.Context, ownerType string, ownerID int64, tags []string) error
	ListTags(ctx context.Context, ownerType string, ownerID int64) ([]string, error)

	InsertFeedback(ctx context.Context, f Feedback) (Feedback, error)
	CountFeedback(ctx context.Context, ownerType string, ownerID int64, outcome string, sinceUnix int64) (int, error)

	WriteSearchDoc(ctx context.Context, doc SearchDoc) error
	DeleteSearchDoc(ctx context.Context, kind string, ownerID int64) error
	SearchCandidates(ctx context.Context, query string, kind string, limit int) ([]Candidate, error)

	BumpUse(ctx context.Context, kind string, id int64, now int64) error
	InsertAudit(ctx context.Context, action, targetType string, targetID *int64, targetUID, detail string) error

	Doctor(ctx context.Context) (DoctorReport, error)
	WithTx(ctx context.Context, fn func(Store) error) error
}

// Feedback is a use / falsification signal.
type Feedback struct {
	ID        int64
	OwnerType string
	OwnerID   int64
	OwnerUID  string
	Outcome   string // helpful | wrong | stale
	Note      string
	ActorID   *int64
	CreatedAt int64
}

// ScopeFilter selects rows visible to a reader.
type ScopeFilter struct {
	Scope    string
	ScopeRef string
}
