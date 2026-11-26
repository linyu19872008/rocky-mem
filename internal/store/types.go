// Package store defines the storage interface shared by backends.
package store

import "context"

// Candidate is a retrieval hit. Score is comparable only within one call.
type Candidate struct {
	Kind    string // memory | experience
	OwnerID int64
	Score   float64
}

// SearchDoc is the materialised retrieval document.
type SearchDoc struct {
	Kind    string
	OwnerID int64
	Lang    string
	Title   string
	Body    string
	Updated int64
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

// Source is an ingested document: background material, not evidence.
type Source struct {
	ID           int64
	UID          string
	Kind         string // file | paste
	URI          string
	Title        string
	Format       string
	TextHash     string
	CharCount    int
	ChunkCount   int
	Lang         string
	Sensitivity  int
	Status       string // active | archived
	SupersededBy *int64
	ProjectID    *int64
	ActorID      *int64
	CreatedAt    int64
	TLastUsed    *int64
	UseCount     int
}

// Chunk is one slice of a source document.
type Chunk struct {
	ID          int64
	UID         string
	SourceID    int64
	Ord         int
	Heading     string
	Text        string
	CharCount   int
	Sensitivity int
}

// ConflictResolution records one human adjudication of a conflict group.
// Append-only: a group can be re-opened, and the history is the audit trail.
type ConflictResolution struct {
	ID          int64
	GroupID     string
	Action      string // adopt|split|reject_both|defer|reopen
	KeptID      *int64
	Note        string
	ActorID     *int64
	ReviewAfter *int64
	CreatedAt   int64
}

// Link relates two entities (complements / contradicts / merged_from / …).
type Link struct {
	ID        int64
	SrcType   string
	SrcID     int64
	DstType   string
	DstID     int64
	Relation  string
	Note      string
	CreatedAt int64
}

// CorpusStats summarises corpus state for doctor.
type CorpusStats struct {
	Sources       int
	ActiveSources int
	Chunks        int
	ActiveChunks  int // chunks whose source is active (i.e. should be indexed)
	ChunkDocs     int // search_doc rows with kind='chunk'
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
	SourceCount     int      `json:"source_count"`
	ActiveSources   int      `json:"active_sources"`
	ChunkCount      int      `json:"chunk_count"`
	ActiveChunks    int      `json:"active_chunks"`
	ChunkDocCount   int      `json:"chunk_doc_count"`
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
	ListExperiencesByTrust(ctx context.Context, trusts []string, limit int) ([]Experience, error)
	CountExperiencesByTrust(ctx context.Context, trusts []string) (map[string]int, error)
	ListAllActiveMemory(ctx context.Context) ([]Memory, error)
	ListIndexableExperiences(ctx context.Context) ([]Experience, error)
	FindExperiencesBySymptomKey(ctx context.Context, key string) ([]Experience, error)
	FindExperiencesByConflictGroup(ctx context.Context, group string) ([]Experience, error)
	FindExperienceDuplicate(ctx context.Context, title, symptom, fix string) (*Experience, error)
	MarkContested(ctx context.Context, ids []int64, group string) error
	UpdateExperienceTrust(ctx context.Context, id int64, trust string) error
	ClearSearchDocs(ctx context.Context) (int, error)
	Purge(ctx context.Context, f PurgeFilter) (PurgeResult, error)

	InsertEvidence(ctx context.Context, ev Evidence) (Evidence, error)
	ListEvidence(ctx context.Context, ownerType string, ownerID int64) ([]Evidence, error)

	SetTags(ctx context.Context, ownerType string, ownerID int64, tags []string) error
	ListTags(ctx context.Context, ownerType string, ownerID int64) ([]string, error)

	InsertFeedback(ctx context.Context, f Feedback) (Feedback, error)
	CountFeedback(ctx context.Context, ownerType string, ownerID int64, outcome string, sinceUnix int64) (int, error)

	// Conflict adjudication and cross-entity relations.
	UpdateExperienceConditions(ctx context.Context, id int64, appliesWhen, invalidWhen string) error
	InsertResolution(ctx context.Context, r ConflictResolution) (ConflictResolution, error)
	LatestResolution(ctx context.Context, group string) (*ConflictResolution, error)
	ListResolutions(ctx context.Context, group string) ([]ConflictResolution, error)
	InsertLink(ctx context.Context, l Link) error
	ListLinks(ctx context.Context, ownerType string, ownerID int64) ([]Link, error)

	// Corpus: ingested documents and their slices. Background material only —
	// no raw row, no evidence link (see design/adr/0008-corpus-layer.md).
	InsertSource(ctx context.Context, s Source) (Source, error)
	GetSourceByTextHash(ctx context.Context, hash string) (*Source, error)
	GetSourceByUID(ctx context.Context, uid string) (*Source, error)
	ArchiveSource(ctx context.Context, id, replacedBy, ts int64) error
	// ArchiveSourceByURI archives older active sources with the same uri in the
	// same project, pointing them at keepID. Returns how many were archived.
	ArchiveSourceByURI(ctx context.Context, projectID *int64, uri string, keepID int64) (int, error)
	ListSourcesByIDs(ctx context.Context, ids []int64) ([]Source, error)

	InsertChunks(ctx context.Context, cs []Chunk) ([]Chunk, error)
	GetChunkByUID(ctx context.Context, uid string) (*Chunk, error)
	ListChunksBySource(ctx context.Context, sourceID int64) ([]Chunk, error)
	// ListActiveChunksByIDs returns only chunks whose source is still active.
	ListActiveChunksByIDs(ctx context.Context, ids []int64) ([]Chunk, error)
	// ListAllActiveChunks returns every chunk of every active source (reindex).
	ListAllActiveChunks(ctx context.Context) ([]Chunk, error)

	DeleteSource(ctx context.Context, id int64) (deletedChunks int, err error)
	CorpusStats(ctx context.Context) (CorpusStats, error)

	WriteSearchDoc(ctx context.Context, doc SearchDoc) error
	DeleteSearchDoc(ctx context.Context, kind string, ownerID int64) error
	SearchCandidates(ctx context.Context, query string, kind string, limit int) ([]Candidate, error)

	BumpUse(ctx context.Context, kind string, id int64, now int64) error
	InsertAudit(ctx context.Context, action, targetType string, targetID *int64, targetUID, detail string) error
	ListAudit(ctx context.Context, sinceUnix int64, limit int) ([]AuditRow, error)

	Doctor(ctx context.Context) (DoctorReport, error)
	Backup(ctx context.Context, destPath string) error
	WithTx(ctx context.Context, fn func(Store) error) error
}

// AuditRow is one audit trail entry.
type AuditRow struct {
	ID         int64
	TS         int64
	Action     string
	TargetType string
	TargetID   *int64
	TargetUID  string
	Detail     string
}

// PurgeFilter selects rows to delete.
type PurgeFilter struct {
	UID             string
	Before          int64
	ProjectScopeRef string
}

// PurgeResult counts deleted rows.
type PurgeResult struct {
	Memories    int
	Experiences int
	SearchDocs  int
	Evidence    int
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
