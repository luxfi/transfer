package voting

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/luxfi/transfer/pkg/store"
)

// Service provides proxy voting operations.
type Service struct {
	store store.TransferStore
}

// New returns a new voting service.
func New(s store.TransferStore) *Service {
	return &Service{store: s}
}

// CreateProposal saves a new proposal.
func (s *Service) CreateProposal(ctx context.Context, p *store.Proposal) error {
	if p.Title == "" {
		return fmt.Errorf("title required")
	}
	if len(p.ShareClassIDs) == 0 {
		return fmt.Errorf("at least one share_class_id required")
	}
	if p.QuorumPercent <= 0 || p.QuorumPercent > 100 {
		return fmt.Errorf("quorum_percent must be between 0 and 100")
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	if p.ID == "" {
		p.ID = generateID("prp")
	}
	return s.store.SaveProposal(ctx, p)
}

// ListProposals returns proposals matching the filter.
func (s *Service) ListProposals(ctx context.Context, filter store.ProposalFilter) ([]store.Proposal, error) {
	return s.store.ListProposals(ctx, filter)
}

// GetProposal returns a single proposal by ID.
func (s *Service) GetProposal(ctx context.Context, id string) (*store.Proposal, error) {
	return s.store.GetProposal(ctx, id)
}

// CastVote records a shareholder's vote on a proposal.
func (s *Service) CastVote(ctx context.Context, v *store.Vote) error {
	if v.ProposalID == "" {
		return fmt.Errorf("proposal_id required")
	}
	if v.ShareholderID == "" {
		return fmt.Errorf("shareholder_id required")
	}
	if v.Choice != "for" && v.Choice != "against" && v.Choice != "abstain" {
		return fmt.Errorf("choice must be 'for', 'against', or 'abstain'")
	}
	if v.SharesVoted <= 0 {
		return fmt.Errorf("shares_voted must be positive")
	}
	if v.ID == "" {
		v.ID = generateID("vot")
	}
	return s.store.CastVote(ctx, v)
}

// GetResults returns voting results for a proposal.
func (s *Service) GetResults(ctx context.Context, proposalID string) (*store.VoteResults, error) {
	return s.store.GetResults(ctx, proposalID)
}

func generateID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
