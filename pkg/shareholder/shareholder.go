package shareholder

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides shareholder registry operations.
type Service struct {
	store store.TransferStore
}

// New returns a new shareholder service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// List returns shareholders matching the filter.
func (s *Service) List(ctx context.Context, filter store.ShareholderFilter) ([]store.Shareholder, error) {
	return s.store.ListShareholders(ctx, filter)
}

// Get returns a single shareholder by ID.
func (s *Service) Get(ctx context.Context, id string) (*store.Shareholder, error) {
	return s.store.GetShareholder(ctx, id)
}

// Create validates and saves a new shareholder.
func (s *Service) Create(ctx context.Context, sh *store.Shareholder) error {
	if sh.Name == "" {
		return fmt.Errorf("name required")
	}
	if sh.Email == "" {
		return fmt.Errorf("email required")
	}
	if sh.Type == "" {
		sh.Type = "individual"
	}
	if sh.ID == "" {
		sh.ID = generateID("sh")
	}
	return s.store.SaveShareholder(ctx, sh)
}

// Update saves changes to an existing shareholder.
func (s *Service) Update(ctx context.Context, sh *store.Shareholder) error {
	if sh.ID == "" {
		return fmt.Errorf("shareholder ID required")
	}
	// Verify exists
	if _, err := s.store.GetShareholder(ctx, sh.ID); err != nil {
		return err
	}
	return s.store.SaveShareholder(ctx, sh)
}

// GetHoldings returns all balances for a shareholder by querying the ledger.
func (s *Service) GetHoldings(ctx context.Context, id string) ([]store.Balance, error) {
	sh, err := s.store.GetShareholder(ctx, id)
	if err != nil {
		return nil, err
	}
	return sh.Holdings, nil
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
