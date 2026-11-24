package core

import (
	"github.com/rocky-mem/rocky-mem/internal/store"
	"github.com/rocky-mem/rocky-mem/internal/termmap"
)

// NewService builds a Service with the embedded term map when available.
func NewService(st store.Store) *Service {
	svc := &Service{Store: st}
	if m, err := termmap.LoadDefault(); err == nil {
		svc.Terms = m
	}
	return svc
}
