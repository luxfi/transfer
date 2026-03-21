package restrictions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides transfer restriction operations.
type Service struct {
	store store.TransferStore
}

// New returns a new restrictions service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// Create saves a new restriction.
func (s *Service) Create(ctx context.Context, r *store.Restriction) error {
	if r.ShareholderID == "" {
		return fmt.Errorf("shareholder_id required")
	}
	if r.ShareClassID == "" {
		return fmt.Errorf("share_class_id required")
	}
	if r.Type == "" {
		return fmt.Errorf("type required")
	}
	if r.ID == "" {
		r.ID = generateID("rst")
	}
	r.Active = true
	return s.store.SaveRestriction(ctx, r)
}

// List returns restrictions for a shareholder.
func (s *Service) List(ctx context.Context, shareholderID string) ([]store.Restriction, error) {
	return s.store.ListRestrictions(ctx, shareholderID)
}

// Delete removes a restriction.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteRestriction(ctx, id)
}

// Check tests whether a transfer is allowed.
func (s *Service) Check(ctx context.Context, fromID, toID, shareClassID string, qty int64) (*store.RestrictionCheck, error) {
	return s.store.CheckTransferAllowed(ctx, fromID, toID, shareClassID, qty)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
