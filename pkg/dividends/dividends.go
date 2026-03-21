package dividends

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides dividend and distribution management.
type Service struct {
	store store.TransferStore
}

// New returns a new dividends service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// Create declares a new dividend.
func (s *Service) Create(ctx context.Context, d *store.Dividend) error {
	if d.ShareClassID == "" {
		return fmt.Errorf("share_class_id required")
	}
	if d.AmountPerShare <= 0 {
		return fmt.Errorf("amount_per_share must be positive")
	}
	if d.Type == "" {
		d.Type = "cash"
	}
	if d.Status == "" {
		d.Status = "declared"
	}
	if d.ID == "" {
		d.ID = generateID("div")
	}
	return s.store.SaveDividend(ctx, d)
}

// List returns dividends matching the filter.
func (s *Service) List(ctx context.Context, filter store.DividendFilter) ([]store.Dividend, error) {
	return s.store.ListDividends(ctx, filter)
}

// Get returns a single dividend by ID.
func (s *Service) Get(ctx context.Context, id string) (*store.Dividend, error) {
	return s.store.GetDividend(ctx, id)
}

// Calculate computes distributions for all shareholders holding the share class.
// In a real implementation this would query all balances as of the record date.
func (s *Service) Calculate(ctx context.Context, id string) (*store.Dividend, error) {
	d, err := s.store.GetDividend(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Status != "declared" && d.Status != "record_set" {
		return nil, fmt.Errorf("dividend %s cannot be calculated in status %s", id, d.Status)
	}

	// Get all shareholders with holdings in this share class
	shareholders, err := s.store.ListShareholders(ctx, store.ShareholderFilter{})
	if err != nil {
		return nil, err
	}

	d.Distributions = nil
	for _, sh := range shareholders {
		bal, err := s.store.GetShareBalance(ctx, sh.ID, d.ShareClassID)
		if err != nil || bal.Quantity <= 0 {
			continue
		}
		amount := float64(bal.Quantity) * d.AmountPerShare
		d.Distributions = append(d.Distributions, store.Distribution{
			ShareholderID: sh.ID,
			Shares:        bal.Quantity,
			Amount:        amount,
			Status:        "pending",
		})
	}

	d.Status = "calculated"
	if err := s.store.SaveDividend(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Pay marks a dividend as paid.
func (s *Service) Pay(ctx context.Context, id string) (*store.Dividend, error) {
	d, err := s.store.GetDividend(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Status != "calculated" {
		return nil, fmt.Errorf("dividend %s must be calculated before paying (status: %s)", id, d.Status)
	}

	for i := range d.Distributions {
		d.Distributions[i].Status = "paid"
	}
	d.Status = "paid"

	if err := s.store.SaveDividend(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
