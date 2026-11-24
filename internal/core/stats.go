package core

import "context"

// StatsResult is a dashboard snapshot for the overview page.
type StatsResult struct {
	ActiveMemories int    `json:"active_memories"`
	Experiences    int    `json:"experiences"`
	PendingReview  int    `json:"pending_review"`
	Proposed       int    `json:"proposed"`
	Contested      int    `json:"contested"`
	DoctorOK       bool   `json:"doctor_ok"`
	SchemaVersion  string `json:"schema_version,omitempty"`
}

// Stats aggregates doctor counts and pending review tallies.
func (s *Service) Stats(ctx context.Context) (*StatsResult, error) {
	doc, err := s.Store.Doctor(ctx)
	if err != nil {
		return nil, err
	}
	byTrust, err := s.Store.CountExperiencesByTrust(ctx, []string{"proposed", "contested"})
	if err != nil {
		return nil, err
	}
	proposed := byTrust["proposed"]
	contested := byTrust["contested"]
	return &StatsResult{
		ActiveMemories: doc.ActiveMemory,
		Experiences:    doc.IndexableExp,
		PendingReview:  proposed + contested,
		Proposed:       proposed,
		Contested:      contested,
		DoctorOK:       doc.SearchDocOK,
		SchemaVersion:  doc.SchemaVersion,
	}, nil
}
