package chain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMPCLedgerRecordEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ledger/record" {
			t.Errorf("path = %q, want /v1/ledger/record", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}

		var entry LedgerEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if entry.TenantID != "tenant-1" {
			t.Errorf("tenant_id = %q, want tenant-1", entry.TenantID)
		}
		if entry.Quantity != 100 {
			t.Errorf("quantity = %d, want 100", entry.Quantity)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tx_hash": "0xabc123"})
	}))
	defer ts.Close()

	ledger := NewMPCLedger(ts.URL)
	txHash, err := ledger.RecordEntry(context.Background(), &LedgerEntry{
		TenantID:   "tenant-1",
		SecurityID: "sec-1",
		ToID:       "holder-1",
		Quantity:   100,
		Type:       "issuance",
		Timestamp:  time.Now(),
	})
	if err != nil {
		t.Fatalf("record entry: %v", err)
	}
	if txHash != "0xabc123" {
		t.Errorf("tx_hash = %q, want 0xabc123", txHash)
	}
}

func TestMPCLedgerVerifyEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ledger/verify/0xabc123" {
			t.Errorf("path = %q, want /v1/ledger/verify/0xabc123", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"entry":    LedgerEntry{TenantID: "t1", ToID: "h1", Quantity: 50},
			"verified": true,
		})
	}))
	defer ts.Close()

	ledger := NewMPCLedger(ts.URL)
	entry, verified, err := ledger.VerifyEntry(context.Background(), "0xabc123")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !verified {
		t.Error("verified = false, want true")
	}
	if entry.Quantity != 50 {
		t.Errorf("quantity = %d, want 50", entry.Quantity)
	}
}

func TestMPCLedgerGetEntryByHash(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ledger/entry/0xdef456" {
			t.Errorf("path = %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LedgerEntry{
			TenantID: "t1",
			ToID:     "h1",
			Quantity: 200,
		})
	}))
	defer ts.Close()

	ledger := NewMPCLedger(ts.URL)
	entry, err := ledger.GetEntryByHash(context.Background(), "0xdef456")
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if entry.Quantity != 200 {
		t.Errorf("quantity = %d, want 200", entry.Quantity)
	}
}

func TestMPCLedgerValidation(t *testing.T) {
	ledger := NewMPCLedger("http://localhost:0")

	_, err := ledger.RecordEntry(context.Background(), nil)
	if err == nil {
		t.Error("nil entry should fail")
	}

	_, err = ledger.RecordEntry(context.Background(), &LedgerEntry{})
	if err == nil {
		t.Error("empty tenant_id should fail")
	}

	_, _, err = ledger.VerifyEntry(context.Background(), "")
	if err == nil {
		t.Error("empty tx hash should fail")
	}

	_, err = ledger.GetEntryByHash(context.Background(), "")
	if err == nil {
		t.Error("empty tx hash should fail")
	}
}

func TestMPCLedgerServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	ledger := NewMPCLedger(ts.URL)
	_, err := ledger.RecordEntry(context.Background(), &LedgerEntry{
		TenantID: "t1",
		ToID:     "h1",
		Quantity: 1,
	})
	if err == nil {
		t.Error("server error should propagate")
	}
}
