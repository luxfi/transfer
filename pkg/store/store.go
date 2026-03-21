package store

import (
	"context"
	"time"
)

// --- Shareholder types ---

type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type Shareholder struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	Type                string     `json:"type"` // individual, entity, trust
	TaxID               string     `json:"tax_id,omitempty"`
	Address             Address    `json:"address"`
	Accredited          bool       `json:"accredited"`
	AccreditationExpiry *time.Time `json:"accreditation_expiry,omitempty"`
	Holdings            []Balance  `json:"holdings,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type ShareholderFilter struct {
	Type       string `json:"type,omitempty"`
	Accredited *bool  `json:"accredited,omitempty"`
	Query      string `json:"query,omitempty"`
}

// --- Ledger types ---

type Transfer struct {
	ID                  string    `json:"id"`
	FromShareholderID   string    `json:"from_shareholder_id,omitempty"`
	ToShareholderID     string    `json:"to_shareholder_id"`
	ShareClassID        string    `json:"share_class_id"`
	Quantity            int64     `json:"quantity"`
	PricePerShare       float64   `json:"price_per_share"`
	Type                string    `json:"type"` // issuance, transfer, cancellation, conversion, split, dividend
	Reason              string    `json:"reason,omitempty"`
	RestrictionsChecked bool      `json:"restrictions_checked"`
	CreatedAt           time.Time `json:"created_at"`
}

type TransferFilter struct {
	ShareholderID string `json:"shareholder_id,omitempty"`
	ShareClassID  string `json:"share_class_id,omitempty"`
	Type          string `json:"type,omitempty"`
}

type Balance struct {
	ShareholderID string `json:"shareholder_id"`
	ShareClassID  string `json:"share_class_id"`
	Quantity      int64  `json:"quantity"`
	Restricted    int64  `json:"restricted"`
	Available     int64  `json:"available"`
}

// --- Restriction types ---

type Restriction struct {
	ID            string     `json:"id"`
	ShareholderID string     `json:"shareholder_id"`
	ShareClassID  string     `json:"share_class_id"`
	Type          string     `json:"type"` // legend, lockup, rofr, rule144, affiliate
	Description   string     `json:"description"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
}

type RestrictionCheck struct {
	Allowed    bool     `json:"allowed"`
	Violations []string `json:"violations,omitempty"`
}

// --- Disclosure types ---

type Disclosure struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Type        string                `json:"type"` // ppm, subscription_agreement, supplement, annual_report
	DocumentURL string                `json:"document_url"`
	Recipients  []DisclosureRecipient `json:"recipients,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
}

type DisclosureRecipient struct {
	ShareholderID  string     `json:"shareholder_id"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ViewedAt       *time.Time `json:"viewed_at,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

type DisclosureFilter struct {
	Type          string `json:"type,omitempty"`
	ShareholderID string `json:"shareholder_id,omitempty"`
}

// --- Communication types ---

type Notice struct {
	ID         string            `json:"id"`
	Subject    string            `json:"subject"`
	Body       string            `json:"body"`
	Type       string            `json:"type"` // general, proxy, dividend, regulatory, k1
	Recipients []NoticeRecipient `json:"recipients,omitempty"`
	SentAt     *time.Time        `json:"sent_at,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

type NoticeRecipient struct {
	ShareholderID string     `json:"shareholder_id"`
	Email         string     `json:"email"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
}

type NoticeFilter struct {
	Type string `json:"type,omitempty"`
}

// --- Dividend types ---

type Dividend struct {
	ID              string         `json:"id"`
	ShareClassID    string         `json:"share_class_id"`
	Type            string         `json:"type"` // cash, stock, return_of_capital
	AmountPerShare  float64        `json:"amount_per_share"`
	RecordDate      time.Time      `json:"record_date"`
	PaymentDate     time.Time      `json:"payment_date"`
	Status          string         `json:"status"` // declared, record_set, calculated, paid
	Distributions   []Distribution `json:"distributions,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

type Distribution struct {
	ShareholderID string  `json:"shareholder_id"`
	Shares        int64   `json:"shares"`
	Amount        float64 `json:"amount"`
	TaxWithheld   float64 `json:"tax_withheld"`
	Status        string  `json:"status"`
}

type DividendFilter struct {
	ShareClassID string `json:"share_class_id,omitempty"`
	Status       string `json:"status,omitempty"`
}

// --- Filing types ---

type Filing struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // form_d, form_d_amendment, blue_sky, state_exemption
	Jurisdiction    string                 `json:"jurisdiction"`
	Status          string                 `json:"status"` // draft, filed, accepted, rejected
	FiledAt         *time.Time             `json:"filed_at,omitempty"`
	ReferenceNumber string                 `json:"reference_number,omitempty"`
	Data            map[string]interface{} `json:"data,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

type FilingFilter struct {
	Type         string `json:"type,omitempty"`
	Jurisdiction string `json:"jurisdiction,omitempty"`
	Status       string `json:"status,omitempty"`
}

// --- Voting types ---

type Proposal struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Type          string    `json:"type"` // board_election, amendment, merger, general
	ShareClassIDs []string  `json:"share_class_ids"`
	RecordDate    time.Time `json:"record_date"`
	Deadline      time.Time `json:"deadline"`
	QuorumPercent float64   `json:"quorum_percent"`
	Status        string    `json:"status"` // draft, open, closed, certified
	CreatedAt     time.Time `json:"created_at"`
}

type ProposalFilter struct {
	Status string `json:"status,omitempty"`
}

type Vote struct {
	ID            string    `json:"id"`
	ProposalID    string    `json:"proposal_id"`
	ShareholderID string    `json:"shareholder_id"`
	Choice        string    `json:"choice"` // for, against, abstain
	SharesVoted   int64     `json:"shares_voted"`
	CastAt        time.Time `json:"cast_at"`
}

type VoteResults struct {
	ProposalID          string `json:"proposal_id"`
	TotalEligibleShares int64  `json:"total_eligible_shares"`
	TotalVotedShares    int64  `json:"total_voted_shares"`
	QuorumMet           bool   `json:"quorum_met"`
	For                 int64  `json:"for"`
	Against             int64  `json:"against"`
	Abstain             int64  `json:"abstain"`
	Status              string `json:"status"`
}

// TransferStore is the storage interface for the transfer agent.
// Implementations: in-memory (dev), PostgreSQL (prod).
type TransferStore interface {
	// Shareholders
	ListShareholders(ctx context.Context, filter ShareholderFilter) ([]Shareholder, error)
	GetShareholder(ctx context.Context, id string) (*Shareholder, error)
	SaveShareholder(ctx context.Context, s *Shareholder) error

	// Ledger entries
	RecordTransfer(ctx context.Context, t *Transfer) error
	ListTransfers(ctx context.Context, filter TransferFilter) ([]Transfer, error)
	GetShareBalance(ctx context.Context, shareholderID, shareClassID string) (*Balance, error)

	// Disclosures
	SaveDisclosure(ctx context.Context, d *Disclosure) error
	ListDisclosures(ctx context.Context, filter DisclosureFilter) ([]Disclosure, error)
	GetDisclosure(ctx context.Context, id string) (*Disclosure, error)
	MarkDisclosureDelivered(ctx context.Context, id, recipientID string) error
	AcknowledgeDisclosure(ctx context.Context, id, recipientID string) error

	// Communications
	SaveNotice(ctx context.Context, n *Notice) error
	ListNotices(ctx context.Context, filter NoticeFilter) ([]Notice, error)
	GetNotice(ctx context.Context, id string) (*Notice, error)
	MarkNoticeSent(ctx context.Context, id string) error

	// Dividends
	SaveDividend(ctx context.Context, d *Dividend) error
	ListDividends(ctx context.Context, filter DividendFilter) ([]Dividend, error)
	GetDividend(ctx context.Context, id string) (*Dividend, error)

	// Filings
	SaveFiling(ctx context.Context, f *Filing) error
	ListFilings(ctx context.Context, filter FilingFilter) ([]Filing, error)
	GetFiling(ctx context.Context, id string) (*Filing, error)

	// Voting
	SaveProposal(ctx context.Context, p *Proposal) error
	ListProposals(ctx context.Context, filter ProposalFilter) ([]Proposal, error)
	GetProposal(ctx context.Context, id string) (*Proposal, error)
	CastVote(ctx context.Context, v *Vote) error
	GetResults(ctx context.Context, proposalID string) (*VoteResults, error)

	// Restrictions
	SaveRestriction(ctx context.Context, r *Restriction) error
	ListRestrictions(ctx context.Context, shareholderID string) ([]Restriction, error)
	DeleteRestriction(ctx context.Context, id string) error
	CheckTransferAllowed(ctx context.Context, fromID, toID, shareClassID string, qty int64) (*RestrictionCheck, error)
}
