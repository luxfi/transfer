package disclosures

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides investor disclosure operations.
type Service struct {
	store store.TransferStore
}

// New returns a new disclosures service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// Create saves a new disclosure document.
func (s *Service) Create(ctx context.Context, d *store.Disclosure) error {
	if d.Name == "" {
		return fmt.Errorf("name required")
	}
	if d.Type == "" {
		return fmt.Errorf("type required")
	}
	if d.ID == "" {
		d.ID = generateID("dsc")
	}
	return s.store.SaveDisclosure(ctx, d)
}

// List returns disclosures matching the filter.
func (s *Service) List(ctx context.Context, filter store.DisclosureFilter) ([]store.Disclosure, error) {
	return s.store.ListDisclosures(ctx, filter)
}

// Get returns a single disclosure by ID.
func (s *Service) Get(ctx context.Context, id string) (*store.Disclosure, error) {
	return s.store.GetDisclosure(ctx, id)
}

// Deliver marks a disclosure as delivered to a recipient.
func (s *Service) Deliver(ctx context.Context, id, recipientID string) error {
	return s.store.MarkDisclosureDelivered(ctx, id, recipientID)
}

// Acknowledge marks a disclosure as acknowledged by a recipient.
func (s *Service) Acknowledge(ctx context.Context, id, recipientID string) error {
	return s.store.AcknowledgeDisclosure(ctx, id, recipientID)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
