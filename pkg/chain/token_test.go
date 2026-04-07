package chain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTokenServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/token/deploy":
			json.NewEncoder(w).Encode(map[string]string{
				"contract_addr": "0xcontract",
				"tx_hash":       "0xdeploy",
			})
		case "/v1/token/mint":
			json.NewEncoder(w).Encode(map[string]string{"tx_hash": "0xmint"})
		case "/v1/token/burn":
			json.NewEncoder(w).Encode(map[string]string{"tx_hash": "0xburn"})
		case "/v1/token/transfer":
			json.NewEncoder(w).Encode(map[string]string{"tx_hash": "0xtransfer"})
		case "/v1/token/freeze":
			json.NewEncoder(w).Encode(map[string]string{"tx_hash": "0xfreeze"})
		case "/v1/token/unfreeze":
			json.NewEncoder(w).Encode(map[string]string{"tx_hash": "0xunfreeze"})
		default:
			// BalanceOf: /v1/token/balance/{contractAddr}/{account}
			json.NewEncoder(w).Encode(map[string]uint64{"balance": 1000})
		}
	}))
}

func TestMPCSecurityTokenDeploy(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	addr, txHash, err := tok.Deploy(context.Background(), "TestSecurity", "TST", 1000000)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if addr != "0xcontract" {
		t.Errorf("addr = %q, want 0xcontract", addr)
	}
	if txHash != "0xdeploy" {
		t.Errorf("txHash = %q, want 0xdeploy", txHash)
	}
}

func TestMPCSecurityTokenMint(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	txHash, err := tok.Mint(context.Background(), "0xcontract", "0xrecipient", 500)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if txHash != "0xmint" {
		t.Errorf("txHash = %q, want 0xmint", txHash)
	}
}

func TestMPCSecurityTokenBurn(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	txHash, err := tok.Burn(context.Background(), "0xcontract", "0xholder", 100)
	if err != nil {
		t.Fatalf("burn: %v", err)
	}
	if txHash != "0xburn" {
		t.Errorf("txHash = %q, want 0xburn", txHash)
	}
}

func TestMPCSecurityTokenTransfer(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	txHash, err := tok.Transfer(context.Background(), "0xcontract", "0xfrom", "0xto", 250)
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if txHash != "0xtransfer" {
		t.Errorf("txHash = %q, want 0xtransfer", txHash)
	}
}

func TestMPCSecurityTokenFreeze(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	txHash, err := tok.Freeze(context.Background(), "0xcontract", "0xaccount")
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if txHash != "0xfreeze" {
		t.Errorf("txHash = %q, want 0xfreeze", txHash)
	}
}

func TestMPCSecurityTokenUnfreeze(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	txHash, err := tok.Unfreeze(context.Background(), "0xcontract", "0xaccount")
	if err != nil {
		t.Fatalf("unfreeze: %v", err)
	}
	if txHash != "0xunfreeze" {
		t.Errorf("txHash = %q, want 0xunfreeze", txHash)
	}
}

func TestMPCSecurityTokenBalanceOf(t *testing.T) {
	ts := newTokenServer(t)
	defer ts.Close()

	tok := NewMPCSecurityToken(ts.URL)
	bal, err := tok.BalanceOf(context.Background(), "0xcontract", "0xaccount")
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if bal != 1000 {
		t.Errorf("balance = %d, want 1000", bal)
	}
}

func TestMPCSecurityTokenValidation(t *testing.T) {
	tok := NewMPCSecurityToken("http://localhost:0")

	_, _, err := tok.Deploy(context.Background(), "", "TST", 100)
	if err == nil {
		t.Error("empty name should fail")
	}

	_, err = tok.Mint(context.Background(), "", "0xto", 100)
	if err == nil {
		t.Error("empty contract should fail")
	}

	_, err = tok.Burn(context.Background(), "0x1", "", 100)
	if err == nil {
		t.Error("empty from should fail")
	}

	_, err = tok.Transfer(context.Background(), "0x1", "0xfrom", "", 100)
	if err == nil {
		t.Error("empty to should fail")
	}

	_, err = tok.Freeze(context.Background(), "", "0xacc")
	if err == nil {
		t.Error("empty contract should fail")
	}

	_, err = tok.Unfreeze(context.Background(), "0x1", "")
	if err == nil {
		t.Error("empty account should fail")
	}

	_, err = tok.BalanceOf(context.Background(), "", "0xacc")
	if err == nil {
		t.Error("empty contract should fail")
	}
}
