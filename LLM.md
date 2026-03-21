# github.com/luxfi/transfer

OSS Transfer Agent library and binary. Base module customized by downstream
projects (liquidity/transfer, captable/transfer, etc.).

## Architecture

- `pkg/store/` — `TransferStore` interface + in-memory implementation
- `pkg/shareholder/` — Shareholder registry CRUD
- `pkg/ledger/` — Share transfer ledger with restriction enforcement
- `pkg/restrictions/` — Transfer restrictions (legends, lock-ups, ROFR, Rule 144)
- `pkg/disclosures/` — Investor disclosures (PPM, sub agreements, delivery tracking)
- `pkg/comms/` — Investor communications (notices, proxy, annual reports)
- `pkg/dividends/` — Dividend declaration, calculation, payment
- `pkg/filings/` — Regulatory filings (Form D, Blue Sky, state exemptions)
- `pkg/voting/` — Proxy voting (proposals, ballots, quorum, results)
- `pkg/router/` — Chi HTTP router mounting all domain handlers
- `cmd/transferd/` — Entry point binary

## Patterns

- All types live in `pkg/store/store.go` — domain packages reference them
- Domain services accept `store.TransferStore` — no direct DB coupling
- In-memory store for dev/test, swap for PostgreSQL store in prod
- Same chi/zerolog/cors stack as `github.com/luxfi/broker`
- Port: `TRANSFER_LISTEN` env, default `:8092`

## Extending

Downstream projects import this module and can:
1. Provide a different `TransferStore` implementation (e.g., PostgreSQL)
2. Add middleware (auth, tenant isolation) to the router
3. Register additional routes on the chi router
4. Wrap domain services with custom business logic
