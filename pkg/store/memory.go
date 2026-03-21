package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryStore is an in-memory TransferStore for development and testing.
type MemoryStore struct {
	mu            sync.RWMutex
	shareholders  map[string]*Shareholder
	transfers     []Transfer
	balances      map[string]*Balance // key: shareholderID:shareClassID
	restrictions  map[string]*Restriction
	disclosures   map[string]*Disclosure
	notices       map[string]*Notice
	dividends     map[string]*Dividend
	filings       map[string]*Filing
	proposals     map[string]*Proposal
	votes         map[string][]Vote // key: proposalID
}

// NewMemoryStore returns a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		shareholders: make(map[string]*Shareholder),
		balances:     make(map[string]*Balance),
		restrictions: make(map[string]*Restriction),
		disclosures:  make(map[string]*Disclosure),
		notices:      make(map[string]*Notice),
		dividends:    make(map[string]*Dividend),
		filings:      make(map[string]*Filing),
		proposals:    make(map[string]*Proposal),
		votes:        make(map[string][]Vote),
	}
}

func balanceKey(shareholderID, shareClassID string) string {
	return shareholderID + ":" + shareClassID
}

// --- Shareholders ---

func (m *MemoryStore) ListShareholders(_ context.Context, filter ShareholderFilter) ([]Shareholder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Shareholder
	for _, s := range m.shareholders {
		if filter.Type != "" && s.Type != filter.Type {
			continue
		}
		if filter.Accredited != nil && s.Accredited != *filter.Accredited {
			continue
		}
		if filter.Query != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(filter.Query)) &&
			!strings.Contains(strings.ToLower(s.Email), strings.ToLower(filter.Query)) {
			continue
		}
		out = append(out, *s)
	}
	return out, nil
}

func (m *MemoryStore) GetShareholder(_ context.Context, id string) (*Shareholder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.shareholders[id]
	if !ok {
		return nil, fmt.Errorf("shareholder %s not found", id)
	}
	cp := *s
	return &cp, nil
}

func (m *MemoryStore) SaveShareholder(_ context.Context, s *Shareholder) error {
	if s.ID == "" {
		return fmt.Errorf("shareholder ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.shareholders[s.ID]; !exists {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()
	cp := *s
	m.shareholders[s.ID] = &cp
	return nil
}

// --- Ledger ---

func (m *MemoryStore) RecordTransfer(_ context.Context, t *Transfer) error {
	if t.ID == "" {
		return fmt.Errorf("transfer ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	t.CreatedAt = time.Now()
	m.transfers = append(m.transfers, *t)

	// Update balances
	if t.FromShareholderID != "" {
		key := balanceKey(t.FromShareholderID, t.ShareClassID)
		bal, ok := m.balances[key]
		if !ok {
			bal = &Balance{ShareholderID: t.FromShareholderID, ShareClassID: t.ShareClassID}
			m.balances[key] = bal
		}
		bal.Quantity -= t.Quantity
		bal.Available -= t.Quantity
	}

	key := balanceKey(t.ToShareholderID, t.ShareClassID)
	bal, ok := m.balances[key]
	if !ok {
		bal = &Balance{ShareholderID: t.ToShareholderID, ShareClassID: t.ShareClassID}
		m.balances[key] = bal
	}
	bal.Quantity += t.Quantity
	bal.Available += t.Quantity

	return nil
}

func (m *MemoryStore) ListTransfers(_ context.Context, filter TransferFilter) ([]Transfer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Transfer
	for _, t := range m.transfers {
		if filter.ShareholderID != "" && t.FromShareholderID != filter.ShareholderID && t.ToShareholderID != filter.ShareholderID {
			continue
		}
		if filter.ShareClassID != "" && t.ShareClassID != filter.ShareClassID {
			continue
		}
		if filter.Type != "" && t.Type != filter.Type {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (m *MemoryStore) GetShareBalance(_ context.Context, shareholderID, shareClassID string) (*Balance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := balanceKey(shareholderID, shareClassID)
	bal, ok := m.balances[key]
	if !ok {
		return &Balance{ShareholderID: shareholderID, ShareClassID: shareClassID}, nil
	}
	cp := *bal
	return &cp, nil
}

// --- Restrictions ---

func (m *MemoryStore) SaveRestriction(_ context.Context, r *Restriction) error {
	if r.ID == "" {
		return fmt.Errorf("restriction ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r.CreatedAt = time.Now()
	cp := *r
	m.restrictions[r.ID] = &cp
	return nil
}

func (m *MemoryStore) ListRestrictions(_ context.Context, shareholderID string) ([]Restriction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Restriction
	for _, r := range m.restrictions {
		if shareholderID != "" && r.ShareholderID != shareholderID {
			continue
		}
		out = append(out, *r)
	}
	return out, nil
}

func (m *MemoryStore) DeleteRestriction(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.restrictions[id]; !ok {
		return fmt.Errorf("restriction %s not found", id)
	}
	delete(m.restrictions, id)
	return nil
}

func (m *MemoryStore) CheckTransferAllowed(_ context.Context, fromID, _ string, shareClassID string, qty int64) (*RestrictionCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	check := &RestrictionCheck{Allowed: true}
	now := time.Now()

	for _, r := range m.restrictions {
		if r.ShareholderID != fromID || r.ShareClassID != shareClassID || !r.Active {
			continue
		}
		if r.ExpiresAt != nil && r.ExpiresAt.Before(now) {
			continue
		}
		switch r.Type {
		case "lockup":
			check.Allowed = false
			check.Violations = append(check.Violations, fmt.Sprintf("lockup restriction %s: %s", r.ID, r.Description))
		case "legend":
			check.Violations = append(check.Violations, fmt.Sprintf("legend on shares: %s", r.Description))
		case "rofr":
			check.Violations = append(check.Violations, fmt.Sprintf("right of first refusal applies: %s", r.Description))
		case "rule144":
			check.Allowed = false
			check.Violations = append(check.Violations, fmt.Sprintf("Rule 144 holding period: %s", r.Description))
		case "affiliate":
			check.Violations = append(check.Violations, fmt.Sprintf("affiliate restriction: %s", r.Description))
		}
	}

	// Check available balance
	key := balanceKey(fromID, shareClassID)
	if bal, ok := m.balances[key]; ok {
		if bal.Available < qty {
			check.Allowed = false
			check.Violations = append(check.Violations, fmt.Sprintf("insufficient available shares: have %d, need %d", bal.Available, qty))
		}
	} else {
		check.Allowed = false
		check.Violations = append(check.Violations, "no shares held")
	}

	return check, nil
}

// --- Disclosures ---

func (m *MemoryStore) SaveDisclosure(_ context.Context, d *Disclosure) error {
	if d.ID == "" {
		return fmt.Errorf("disclosure ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	d.CreatedAt = time.Now()
	cp := *d
	cp.Recipients = make([]DisclosureRecipient, len(d.Recipients))
	copy(cp.Recipients, d.Recipients)
	m.disclosures[d.ID] = &cp
	return nil
}

func (m *MemoryStore) ListDisclosures(_ context.Context, filter DisclosureFilter) ([]Disclosure, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Disclosure
	for _, d := range m.disclosures {
		if filter.Type != "" && d.Type != filter.Type {
			continue
		}
		if filter.ShareholderID != "" {
			found := false
			for _, r := range d.Recipients {
				if r.ShareholderID == filter.ShareholderID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, *d)
	}
	return out, nil
}

func (m *MemoryStore) GetDisclosure(_ context.Context, id string) (*Disclosure, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.disclosures[id]
	if !ok {
		return nil, fmt.Errorf("disclosure %s not found", id)
	}
	cp := *d
	return &cp, nil
}

func (m *MemoryStore) MarkDisclosureDelivered(_ context.Context, id, recipientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.disclosures[id]
	if !ok {
		return fmt.Errorf("disclosure %s not found", id)
	}
	now := time.Now()
	for i := range d.Recipients {
		if d.Recipients[i].ShareholderID == recipientID {
			d.Recipients[i].DeliveredAt = &now
			return nil
		}
	}
	return fmt.Errorf("recipient %s not found in disclosure %s", recipientID, id)
}

func (m *MemoryStore) AcknowledgeDisclosure(_ context.Context, id, recipientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.disclosures[id]
	if !ok {
		return fmt.Errorf("disclosure %s not found", id)
	}
	now := time.Now()
	for i := range d.Recipients {
		if d.Recipients[i].ShareholderID == recipientID {
			d.Recipients[i].AcknowledgedAt = &now
			return nil
		}
	}
	return fmt.Errorf("recipient %s not found in disclosure %s", recipientID, id)
}

// --- Notices ---

func (m *MemoryStore) SaveNotice(_ context.Context, n *Notice) error {
	if n.ID == "" {
		return fmt.Errorf("notice ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n.CreatedAt = time.Now()
	cp := *n
	cp.Recipients = make([]NoticeRecipient, len(n.Recipients))
	copy(cp.Recipients, n.Recipients)
	m.notices[n.ID] = &cp
	return nil
}

func (m *MemoryStore) ListNotices(_ context.Context, filter NoticeFilter) ([]Notice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Notice
	for _, n := range m.notices {
		if filter.Type != "" && n.Type != filter.Type {
			continue
		}
		out = append(out, *n)
	}
	return out, nil
}

func (m *MemoryStore) GetNotice(_ context.Context, id string) (*Notice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.notices[id]
	if !ok {
		return nil, fmt.Errorf("notice %s not found", id)
	}
	cp := *n
	return &cp, nil
}

func (m *MemoryStore) MarkNoticeSent(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.notices[id]
	if !ok {
		return fmt.Errorf("notice %s not found", id)
	}
	now := time.Now()
	n.SentAt = &now
	for i := range n.Recipients {
		n.Recipients[i].DeliveredAt = &now
	}
	return nil
}

// --- Dividends ---

func (m *MemoryStore) SaveDividend(_ context.Context, d *Dividend) error {
	if d.ID == "" {
		return fmt.Errorf("dividend ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	d.CreatedAt = time.Now()
	cp := *d
	cp.Distributions = make([]Distribution, len(d.Distributions))
	copy(cp.Distributions, d.Distributions)
	m.dividends[d.ID] = &cp
	return nil
}

func (m *MemoryStore) ListDividends(_ context.Context, filter DividendFilter) ([]Dividend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Dividend
	for _, d := range m.dividends {
		if filter.ShareClassID != "" && d.ShareClassID != filter.ShareClassID {
			continue
		}
		if filter.Status != "" && d.Status != filter.Status {
			continue
		}
		out = append(out, *d)
	}
	return out, nil
}

func (m *MemoryStore) GetDividend(_ context.Context, id string) (*Dividend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.dividends[id]
	if !ok {
		return nil, fmt.Errorf("dividend %s not found", id)
	}
	cp := *d
	return &cp, nil
}

// --- Filings ---

func (m *MemoryStore) SaveFiling(_ context.Context, f *Filing) error {
	if f.ID == "" {
		return fmt.Errorf("filing ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f.CreatedAt = time.Now()
	cp := *f
	m.filings[f.ID] = &cp
	return nil
}

func (m *MemoryStore) ListFilings(_ context.Context, filter FilingFilter) ([]Filing, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Filing
	for _, f := range m.filings {
		if filter.Type != "" && f.Type != filter.Type {
			continue
		}
		if filter.Jurisdiction != "" && f.Jurisdiction != filter.Jurisdiction {
			continue
		}
		if filter.Status != "" && f.Status != filter.Status {
			continue
		}
		out = append(out, *f)
	}
	return out, nil
}

func (m *MemoryStore) GetFiling(_ context.Context, id string) (*Filing, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.filings[id]
	if !ok {
		return nil, fmt.Errorf("filing %s not found", id)
	}
	cp := *f
	return &cp, nil
}

// --- Voting ---

func (m *MemoryStore) SaveProposal(_ context.Context, p *Proposal) error {
	if p.ID == "" {
		return fmt.Errorf("proposal ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p.CreatedAt = time.Now()
	cp := *p
	cp.ShareClassIDs = make([]string, len(p.ShareClassIDs))
	copy(cp.ShareClassIDs, p.ShareClassIDs)
	m.proposals[p.ID] = &cp
	return nil
}

func (m *MemoryStore) ListProposals(_ context.Context, filter ProposalFilter) ([]Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Proposal
	for _, p := range m.proposals {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		out = append(out, *p)
	}
	return out, nil
}

func (m *MemoryStore) GetProposal(_ context.Context, id string) (*Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.proposals[id]
	if !ok {
		return nil, fmt.Errorf("proposal %s not found", id)
	}
	cp := *p
	return &cp, nil
}

func (m *MemoryStore) CastVote(_ context.Context, v *Vote) error {
	if v.ID == "" {
		return fmt.Errorf("vote ID required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.proposals[v.ProposalID]
	if !ok {
		return fmt.Errorf("proposal %s not found", v.ProposalID)
	}
	if p.Status != "open" {
		return fmt.Errorf("proposal %s is not open for voting (status: %s)", v.ProposalID, p.Status)
	}

	// Check for duplicate vote
	for _, existing := range m.votes[v.ProposalID] {
		if existing.ShareholderID == v.ShareholderID {
			return fmt.Errorf("shareholder %s has already voted on proposal %s", v.ShareholderID, v.ProposalID)
		}
	}

	v.CastAt = time.Now()
	m.votes[v.ProposalID] = append(m.votes[v.ProposalID], *v)
	return nil
}

func (m *MemoryStore) GetResults(_ context.Context, proposalID string) (*VoteResults, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.proposals[proposalID]
	if !ok {
		return nil, fmt.Errorf("proposal %s not found", proposalID)
	}

	results := &VoteResults{
		ProposalID: proposalID,
		Status:     p.Status,
	}

	// Calculate total eligible shares from balances for the relevant share classes
	classSet := make(map[string]bool)
	for _, cls := range p.ShareClassIDs {
		classSet[cls] = true
	}
	for key, bal := range m.balances {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 && classSet[parts[1]] {
			results.TotalEligibleShares += bal.Quantity
		}
	}

	for _, v := range m.votes[proposalID] {
		results.TotalVotedShares += v.SharesVoted
		switch v.Choice {
		case "for":
			results.For += v.SharesVoted
		case "against":
			results.Against += v.SharesVoted
		case "abstain":
			results.Abstain += v.SharesVoted
		}
	}

	if results.TotalEligibleShares > 0 {
		votedPct := float64(results.TotalVotedShares) / float64(results.TotalEligibleShares) * 100
		results.QuorumMet = votedPct >= p.QuorumPercent
	}

	return results, nil
}
