// Package chain provides on-chain ledger and security token interfaces
// for the transfer agent, backed by an MPC signing service.
package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LedgerEntry is a transfer agent ledger record committed on-chain.
type LedgerEntry struct {
	TenantID    string    `json:"tenant_id"`
	SecurityID  string    `json:"security_id"`
	FromID      string    `json:"from_id,omitempty"`
	ToID        string    `json:"to_id"`
	Quantity    int64     `json:"quantity"`
	Type        string    `json:"type"` // issuance, transfer, cancellation, conversion
	Timestamp   time.Time `json:"timestamp"`
	PQSignature []byte    `json:"pq_signature,omitempty"`
}

// OnChainLedger records and verifies transfer entries on-chain.
type OnChainLedger interface {
	RecordEntry(ctx context.Context, entry *LedgerEntry) (txHash string, err error)
	VerifyEntry(ctx context.Context, txHash string) (*LedgerEntry, bool, error)
	GetEntryByHash(ctx context.Context, txHash string) (*LedgerEntry, error)
}

// MPCLedger implements OnChainLedger via an external MPC signing service.
type MPCLedger struct {
	mpcURL string
	client *http.Client
}

// NewMPCLedger returns a ledger that delegates signing to the MPC service at mpcURL.
func NewMPCLedger(mpcURL string) *MPCLedger {
	return &MPCLedger{
		mpcURL: mpcURL,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (l *MPCLedger) RecordEntry(ctx context.Context, entry *LedgerEntry) (string, error) {
	if entry == nil {
		return "", errors.New("chain: nil entry")
	}
	if entry.TenantID == "" {
		return "", errors.New("chain: tenant_id required")
	}
	if entry.ToID == "" {
		return "", errors.New("chain: to_id required")
	}

	var resp struct {
		TxHash string `json:"tx_hash"`
	}
	if err := l.post(ctx, "/v1/ledger/record", entry, &resp); err != nil {
		return "", fmt.Errorf("chain: record entry: %w", err)
	}

	return resp.TxHash, nil
}

func (l *MPCLedger) VerifyEntry(ctx context.Context, txHash string) (*LedgerEntry, bool, error) {
	if txHash == "" {
		return nil, false, errors.New("chain: empty tx hash")
	}

	var resp struct {
		Entry    *LedgerEntry `json:"entry"`
		Verified bool         `json:"verified"`
	}
	if err := l.get(ctx, "/v1/ledger/verify/"+txHash, &resp); err != nil {
		return nil, false, fmt.Errorf("chain: verify entry: %w", err)
	}

	return resp.Entry, resp.Verified, nil
}

func (l *MPCLedger) GetEntryByHash(ctx context.Context, txHash string) (*LedgerEntry, error) {
	if txHash == "" {
		return nil, errors.New("chain: empty tx hash")
	}

	var resp LedgerEntry
	if err := l.get(ctx, "/v1/ledger/entry/"+txHash, &resp); err != nil {
		return nil, fmt.Errorf("chain: get entry: %w", err)
	}

	return &resp, nil
}

// post sends a JSON POST to the MPC service.
func (l *MPCLedger) post(ctx context.Context, path string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.mpcURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return l.do(req, result)
}

// get sends a JSON GET to the MPC service.
func (l *MPCLedger) get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.mpcURL+path, nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	return l.do(req, result)
}

func (l *MPCLedger) do(req *http.Request, result any) error {
	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mpc service %d: %s", resp.StatusCode, body)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}

	return nil
}
