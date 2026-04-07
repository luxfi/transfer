package chain

import (
	"context"
	"errors"
	"fmt"
)

// SecurityToken manages ERC-3643 / T-REX compliant security tokens on-chain.
// Operations include deploy, mint, burn, transfer, and freeze/unfreeze.
// All mutating operations go through MPC signing -- no private keys are held locally.
type SecurityToken interface {
	Deploy(ctx context.Context, name, symbol string, totalSupply uint64) (contractAddr string, txHash string, err error)
	Mint(ctx context.Context, contractAddr, to string, amount uint64) (txHash string, err error)
	Burn(ctx context.Context, contractAddr, from string, amount uint64) (txHash string, err error)
	Transfer(ctx context.Context, contractAddr, from, to string, amount uint64) (txHash string, err error)
	Freeze(ctx context.Context, contractAddr, account string) (txHash string, err error)
	Unfreeze(ctx context.Context, contractAddr, account string) (txHash string, err error)
	BalanceOf(ctx context.Context, contractAddr, account string) (uint64, error)
}

// MPCSecurityToken implements SecurityToken via MPC signing service HTTP calls.
type MPCSecurityToken struct {
	ledger *MPCLedger // reuse the HTTP client and base URL
}

// NewMPCSecurityToken returns a SecurityToken backed by the MPC service at mpcURL.
func NewMPCSecurityToken(mpcURL string) *MPCSecurityToken {
	return &MPCSecurityToken{
		ledger: NewMPCLedger(mpcURL),
	}
}

func (t *MPCSecurityToken) Deploy(ctx context.Context, name, symbol string, totalSupply uint64) (string, string, error) {
	if name == "" || symbol == "" {
		return "", "", errors.New("chain: name and symbol required")
	}

	req := struct {
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		TotalSupply uint64 `json:"total_supply"`
	}{name, symbol, totalSupply}

	var resp struct {
		ContractAddr string `json:"contract_addr"`
		TxHash       string `json:"tx_hash"`
	}
	if err := t.ledger.post(ctx, "/v1/token/deploy", req, &resp); err != nil {
		return "", "", fmt.Errorf("chain: deploy: %w", err)
	}

	return resp.ContractAddr, resp.TxHash, nil
}

func (t *MPCSecurityToken) Mint(ctx context.Context, contractAddr, to string, amount uint64) (string, error) {
	if contractAddr == "" || to == "" {
		return "", errors.New("chain: contract_addr and to required")
	}

	req := struct {
		ContractAddr string `json:"contract_addr"`
		To           string `json:"to"`
		Amount       uint64 `json:"amount"`
	}{contractAddr, to, amount}

	var resp struct {
		TxHash string `json:"tx_hash"`
	}
	if err := t.ledger.post(ctx, "/v1/token/mint", req, &resp); err != nil {
		return "", fmt.Errorf("chain: mint: %w", err)
	}

	return resp.TxHash, nil
}

func (t *MPCSecurityToken) Burn(ctx context.Context, contractAddr, from string, amount uint64) (string, error) {
	if contractAddr == "" || from == "" {
		return "", errors.New("chain: contract_addr and from required")
	}

	req := struct {
		ContractAddr string `json:"contract_addr"`
		From         string `json:"from"`
		Amount       uint64 `json:"amount"`
	}{contractAddr, from, amount}

	var resp struct {
		TxHash string `json:"tx_hash"`
	}
	if err := t.ledger.post(ctx, "/v1/token/burn", req, &resp); err != nil {
		return "", fmt.Errorf("chain: burn: %w", err)
	}

	return resp.TxHash, nil
}

func (t *MPCSecurityToken) Transfer(ctx context.Context, contractAddr, from, to string, amount uint64) (string, error) {
	if contractAddr == "" || from == "" || to == "" {
		return "", errors.New("chain: contract_addr, from, and to required")
	}

	req := struct {
		ContractAddr string `json:"contract_addr"`
		From         string `json:"from"`
		To           string `json:"to"`
		Amount       uint64 `json:"amount"`
	}{contractAddr, from, to, amount}

	var resp struct {
		TxHash string `json:"tx_hash"`
	}
	if err := t.ledger.post(ctx, "/v1/token/transfer", req, &resp); err != nil {
		return "", fmt.Errorf("chain: transfer: %w", err)
	}

	return resp.TxHash, nil
}

func (t *MPCSecurityToken) Freeze(ctx context.Context, contractAddr, account string) (string, error) {
	if contractAddr == "" || account == "" {
		return "", errors.New("chain: contract_addr and account required")
	}

	req := struct {
		ContractAddr string `json:"contract_addr"`
		Account      string `json:"account"`
	}{contractAddr, account}

	var resp struct {
		TxHash string `json:"tx_hash"`
	}
	if err := t.ledger.post(ctx, "/v1/token/freeze", req, &resp); err != nil {
		return "", fmt.Errorf("chain: freeze: %w", err)
	}

	return resp.TxHash, nil
}

func (t *MPCSecurityToken) Unfreeze(ctx context.Context, contractAddr, account string) (string, error) {
	if contractAddr == "" || account == "" {
		return "", errors.New("chain: contract_addr and account required")
	}

	req := struct {
		ContractAddr string `json:"contract_addr"`
		Account      string `json:"account"`
	}{contractAddr, account}

	var resp struct {
		TxHash string `json:"tx_hash"`
	}
	if err := t.ledger.post(ctx, "/v1/token/unfreeze", req, &resp); err != nil {
		return "", fmt.Errorf("chain: unfreeze: %w", err)
	}

	return resp.TxHash, nil
}

func (t *MPCSecurityToken) BalanceOf(ctx context.Context, contractAddr, account string) (uint64, error) {
	if contractAddr == "" || account == "" {
		return 0, errors.New("chain: contract_addr and account required")
	}

	var resp struct {
		Balance uint64 `json:"balance"`
	}
	if err := t.ledger.get(ctx, "/v1/token/balance/"+contractAddr+"/"+account, &resp); err != nil {
		return 0, fmt.Errorf("chain: balance: %w", err)
	}

	return resp.Balance, nil
}
