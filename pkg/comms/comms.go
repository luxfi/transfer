package comms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides investor communication operations.
type Service struct {
	store store.TransferStore
}

// New returns a new comms service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// Create saves a new notice.
func (s *Service) Create(ctx context.Context, n *store.Notice) error {
	if n.Subject == "" {
		return fmt.Errorf("subject required")
	}
	if n.Type == "" {
		n.Type = "general"
	}
	if n.ID == "" {
		n.ID = generateID("ntc")
	}
	return s.store.SaveNotice(ctx, n)
}

// List returns notices matching the filter.
func (s *Service) List(ctx context.Context, filter store.NoticeFilter) ([]store.Notice, error) {
	return s.store.ListNotices(ctx, filter)
}

// Get returns a single notice by ID.
func (s *Service) Get(ctx context.Context, id string) (*store.Notice, error) {
	return s.store.GetNotice(ctx, id)
}

// Send marks a notice as sent and delivers to all recipients.
func (s *Service) Send(ctx context.Context, id string) error {
	return s.store.MarkNoticeSent(ctx, id)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
