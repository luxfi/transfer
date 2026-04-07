# Lux Transfer

Transfer agent domain library for securities record-keeping, shareholder management, and regulatory compliance.

```
go get github.com/luxfi/transfer
```

## Architecture

`luxfi/transfer` is a headless transfer agent engine. It provides the full SEC-regulated transfer agent function set as composable Go packages behind interface-driven storage. No database opinion -- consumers wire in PostgreSQL, SQLite, or anything that satisfies `store.TransferStore`.

```
cmd/transferd        Server binary (chi router, healthz, graceful shutdown)
pkg/store            Aggregate storage interface (8 sub-interfaces, 30+ methods)
pkg/shareholder      Shareholder CRUD, holdings, accreditation tracking
pkg/ledger           Immutable transfer ledger (issuance, transfer, cancellation, conversion, split, dividend)
pkg/restrictions     Transfer restrictions (legend, lockup, ROFR, Rule 144, affiliate)
pkg/disclosures      PPM/subscription agreement delivery, acknowledgment tracking
pkg/comms            Notices -- proxy, dividend, regulatory, K-1 distribution
pkg/dividends        Dividend declaration, record-date calculation, distribution, payment
pkg/filings          Form D, blue sky, state exemption filings
pkg/voting           Shareholder proposals, vote casting, quorum calculation, certification
pkg/crypto           PQ hybrid encryption envelopes + FHE field encryption
pkg/chain            On-chain ledger + ERC-3643 security token interface via MPC signing
pkg/router           REST API (chi) mounting all domain services
```

### Post-Quantum Cryptography

`pkg/crypto` provides two layers of protection:

**Envelope encryption** (ML-KEM-1024 + AES-256-GCM + ML-DSA-87):
- Key encapsulation via ML-KEM-1024 (FIPS 203, NIST Level 5)
- Symmetric payload encryption via AES-256-GCM
- Digital signatures via ML-DSA-87 (FIPS 204, NIST Level 5)
- Wraps `luxfi/crypto/mlkem` and `luxfi/crypto/mldsa` directly

**FHE field encryption** (via `luxfi/fhe`):
- Encrypt individual record fields as 64-bit ciphertexts
- Homomorphic comparison and summation on encrypted balances
- Evaluation keys allow computation without decryption keys

### On-Chain Ledger

`pkg/chain` commits transfer records on-chain and manages ERC-3643 security tokens. All signing is delegated to an external MPC service -- no private keys are held by the transfer agent.

- `OnChainLedger` -- record, verify, and retrieve ledger entries by tx hash
- `SecurityToken` -- deploy, mint, burn, transfer, freeze/unfreeze ERC-3643 tokens
- Both implemented via `MPCLedger`/`MPCSecurityToken` HTTP clients

### REST API

`pkg/router` exposes all domain services over HTTP via chi:

| Endpoint | Methods | Domain |
|----------|---------|--------|
| `/shareholders` | GET, POST | Shareholder management |
| `/shareholders/{id}/holdings` | GET | Position lookup |
| `/shareholders/{id}/restrictions` | GET | Per-holder restrictions |
| `/transfers` | GET, POST | Ledger entries |
| `/ledger/{shareClassId}` | GET | Per-class ledger view |
| `/disclosures` | GET, POST | Document delivery |
| `/disclosures/{id}/deliver` | POST | Mark delivered |
| `/disclosures/{id}/acknowledge` | POST | Record acknowledgment |
| `/notices` | GET, POST | Communications |
| `/notices/{id}/send` | POST | Dispatch notice |
| `/dividends` | GET, POST | Declarations |
| `/dividends/{id}/calculate` | POST | Compute distributions |
| `/dividends/{id}/pay` | POST | Execute payment |
| `/filings` | GET, POST, PATCH | Regulatory filings |
| `/proposals` | GET, POST | Shareholder voting |
| `/proposals/{id}/vote` | POST | Cast vote |
| `/proposals/{id}/results` | GET | Tally + quorum |
| `/restrictions` | GET, POST, DELETE | Transfer restrictions |
| `/restrictions/check` | POST | Pre-flight restriction check |
| `/healthz` | GET | Health probe |

## Quick Start

```go
import (
    "github.com/luxfi/transfer/pkg/store"
    "github.com/luxfi/transfer/pkg/shareholder"
    "github.com/luxfi/transfer/pkg/ledger"
    "github.com/luxfi/transfer/pkg/router"
)

// Implement store.TransferStore with your database.
db := myPostgresStore{}

// Create domain services.
shSvc := shareholder.NewService(db)
ldgSvc := ledger.NewService(db)
// ... remaining services

// Mount the router.
r := router.New(shSvc, ldgSvc, rstSvc, dscSvc, comSvc, divSvc, filSvc, votSvc)
http.ListenAndServe(":8080", r)
```

## Testing

```bash
go test ./...    # 26 test functions across all packages
```

## Papers

- [Lux PQ Crypto Suite](https://github.com/luxfi/papers/blob/main/lux-pq-crypto-suite.pdf) -- ML-KEM, ML-DSA, SLH-DSA parameter selection
- [Lux FHE Smart Contracts](https://github.com/luxfi/papers/blob/main/lux-fhe-smart-contracts.pdf) -- encrypted on-chain computation
- [Lux Threshold MPC](https://github.com/luxfi/papers/blob/main/lux-threshold-mpc.pdf) -- MPC signing architecture

## License

Lux Ecosystem License v1.2. See [LICENSE](https://github.com/luxfi/crypto/blob/main/LICENSE).
