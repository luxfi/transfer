package filings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides regulatory filing operations.
type Service struct {
	store store.TransferStore
}

// New returns a new filings service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// Create saves a new filing.
func (s *Service) Create(ctx context.Context, f *store.Filing) error {
	if f.Type == "" {
		return fmt.Errorf("type required")
	}
	if f.Jurisdiction == "" {
		return fmt.Errorf("jurisdiction required")
	}
	if f.Status == "" {
		f.Status = "draft"
	}
	if f.ID == "" {
		f.ID = generateID("fil")
	}
	return s.store.SaveFiling(ctx, f)
}

// List returns filings matching the filter.
func (s *Service) List(ctx context.Context, filter store.FilingFilter) ([]store.Filing, error) {
	return s.store.ListFilings(ctx, filter)
}

// Get returns a single filing by ID.
func (s *Service) Get(ctx context.Context, id string) (*store.Filing, error) {
	return s.store.GetFiling(ctx, id)
}

// Update saves changes to an existing filing.
func (s *Service) Update(ctx context.Context, f *store.Filing) error {
	if f.ID == "" {
		return fmt.Errorf("filing ID required")
	}
	if _, err := s.store.GetFiling(ctx, f.ID); err != nil {
		return err
	}
	return s.store.SaveFiling(ctx, f)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
