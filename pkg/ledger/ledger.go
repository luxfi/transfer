package ledger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides share transfer ledger operations.
type Service struct {
	store store.TransferStore
}

// New returns a new ledger service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// RecordTransfer validates and records a share transfer.
// For transfers (not issuances), it checks restrictions first.
func (s *Service) RecordTransfer(ctx context.Context, t *store.Transfer) error {
	if t.ToShareholderID == "" {
		return fmt.Errorf("to_shareholder_id required")
	}
	if t.ShareClassID == "" {
		return fmt.Errorf("share_class_id required")
	}
	if t.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if t.ID == "" {
		t.ID = generateID("txn")
	}

	// For non-issuance transfers, check restrictions
	if t.Type == "transfer" && t.FromShareholderID != "" {
		check, err := s.store.CheckTransferAllowed(ctx, t.FromShareholderID, t.ToShareholderID, t.ShareClassID, t.Quantity)
		if err != nil {
			return fmt.Errorf("restriction check failed: %w", err)
		}
		if !check.Allowed {
			return fmt.Errorf("transfer blocked: %v", check.Violations)
		}
		t.RestrictionsChecked = true
	}

	return s.store.RecordTransfer(ctx, t)
}

// ListTransfers returns transfers matching the filter.
func (s *Service) ListTransfers(ctx context.Context, filter store.TransferFilter) ([]store.Transfer, error) {
	return s.store.ListTransfers(ctx, filter)
}

// GetBalance returns the share balance for a shareholder in a share class.
func (s *Service) GetBalance(ctx context.Context, shareholderID, shareClassID string) (*store.Balance, error) {
	return s.store.GetShareBalance(ctx, shareholderID, shareClassID)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
