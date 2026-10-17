# web3-chainlog

A practice repo for learning the blockchain / Web3 stack end to end: one Solidity contract, a Go event indexer + HTTP API, and a Next.js dApp, in one repo.

The event indexer is written from scratch with `go-ethereum` rather than an indexing framework. Runs locally with Anvil or on the Sepolia testnet, backed by a single SQLite file. No mainnet, no real funds.

## Status

Scaffold only. The layout and minimal files are in place; every implementation file is a stub (`package x`, `contract {}`, toy components). Nothing is wired yet. See **Build Order** for what to implement, in what sequence.

## Practice Goals

Touch the whole Web3 write/read path yourself, one layer at a time.

- **Contract** — write, test, and deploy a minimal guestbook: `post(string)`, the `MessagePosted` event, and read getters (`messageCount` / `getMessage` / `getMessages`).
- **Local chain** — run Anvil (chain id `31337`) and deploy the same contract to Sepolia (`11155111`).
- **Indexer, by hand** — watch `MessagePosted` with `go-ethereum`'s `ethclient`, resume from a `sync_state` cursor, and insert idempotently on `(tx_hash, log_index)` so replays never double-write.
- **API** — serve reads over `chi`: `GET /api/health`, `GET /api/messages`, `GET /api/sync`.
- **Storage** — persist indexed rows in SQLite (`modernc.org/sqlite`, no cgo), schema in `internal/store/schema.sql`.
- **dApp write path** — post from the browser with a wallet (`wagmi` / `viem`): the user signs, the contract emits, the indexer picks it up, the list refreshes.
- **Relayer (optional)** — let the backend send transactions itself (testnet only) to touch nonce / gas / tx lifecycle.
- **Testnet only** — no committed private keys.

## Skill / Tech

| Layer | Skill | Tech |
| --- | --- | --- |
| Contract | Solidity, ABIs, events, gas | Solidity `^0.8.24` + Foundry (`forge` / `anvil` / `cast`) |
| Backend | Go, HTTP APIs, SQL, log decoding, idempotency | Go 1.27, `chi`, `godotenv`, `database/sql` + `modernc.org/sqlite`, `go-ethereum` |
| Frontend | Wallet connect, signing, reading a contract | Next.js (App Router, TS) + Tailwind, `wagmi` + `viem` (+ RainbowKit / shadcn/ui) |
| Chain | Testnets, RPC, tx lifecycle | Anvil locally, Sepolia, RPC provider (Alchemy / Infura / public) |
| Storage | Schema, cursors, idempotency | SQLite (single file, no server) |

## Architecture

Write on-chain, read from the indexer. The chain is the source of truth; the API is a read cache built from its events.

```text
wallet --tx--> Guestbook (chain) --MessagePosted--> Go indexer --insert--> SQLite
                                                                            |
browser <--GET /api/messages-- chi API <------------------------------------+
```

- **Write** — the browser signs `post(content)`. The contract stores it and emits `MessagePosted(sender, index, content, timestamp)`.
- **Index** — the Go indexer reads logs after `sync_state.last_block`, decodes `MessagePosted`, inserts into `events` (idempotent) and `messages`, then advances the cursor.
- **Read** — the frontend fetches `GET /api/messages`; the API reads SQLite. The chain is never queried on the read path.
- **Relayer (optional)** — the backend can also call `post` itself with a testnet key.

## Project Layout

```text
.
├── docker-compose.yml        # anvil + backend
├── contracts/                # Solidity + Foundry
│   ├── foundry.toml
│   ├── remappings.txt        # forge-std
│   ├── src/Guestbook.sol     # guestbook contract (stub)
│   └── test/Guestbook.t.sol  # forge tests (stub)
├── backend/                  # Go indexer + HTTP API
│   ├── go.mod
│   ├── main.go               # wire config -> store -> indexer -> api (stub)
│   ├── Dockerfile
│   ├── .env.example
│   └── internal/
│       ├── config/config.go  # godotenv + os.Getenv (stub)
│       ├── store/store.go    # database/sql + SQLite (stub)
│       ├── store/schema.sql  # sync_state / events / messages
│       ├── chain/indexer.go  # ethclient: watch MessagePosted (stub)
│       ├── chain/relayer.go  # ethclient: send tx, testnet only (stub)
│       └── api/router.go     # chi routes (stub)
└── frontend/                 # Next.js dApp
    ├── package.json
    ├── .env.local.example
    └── src/
        ├── app/              # layout + page (stub)
        ├── components/       # ConnectButton / MessageList / PostForm (stubs)
        └── lib/              # contract address + ABI / wagmi config (stubs)
```

## Interfaces

Three seams connect the layers. Each is owned and documented by the layer that produces it.

- **Contract surface** — the ABI the other two layers talk to: [contracts/README.md](contracts/README.md)
- **Database schema** and **HTTP API** — what the indexer stores and the API serves: [backend/README.md](backend/README.md)
- **Consumer side** — the frontend reads the API and calls the contract: [frontend/README.md](frontend/README.md)

## Build Order

Each step is runnable before the next.

1. **Contract** — implement `Guestbook.sol` + `forge test` green.
2. **Local chain** — `anvil`, deploy, post a message with `cast` to see the event.
3. **Store** — config + SQLite schema; insert and read rows in a Go test.
4. **Indexer** — watch `MessagePosted`, advance the cursor, prove idempotency on replay.
5. **API** — `chi` routes reading `messages` / `sync_state`.
6. **Frontend read** — list messages from the API.
7. **Frontend write** — wallet connect + `post` via `wagmi`, then refresh.
8. **Relayer (optional)** — backend-signed `post`.

## Quick Start

Prerequisites: Go 1.27+, Foundry, Node 18+. Run the chain + backend either locally or with Docker Compose; the frontend runs on the host either way. Full steps live in each layer's README.

### Local

```bash
# contracts
cd contracts && forge build && forge test         # then: anvil + forge create

# backend
cd backend && cp .env.example .env && go run .    # fill RPC_URL, CHAIN_ID, CONTRACT_ADDRESS

# frontend
cd frontend && cp .env.local.example .env.local   # fill NEXT_PUBLIC_API_URL, CONTRACT_ADDRESS
npm install && npm run dev
```

### Docker Compose (anvil + backend)

```bash
cp backend/.env.example backend/.env              # fill CONTRACT_ADDRESS after deploy
docker compose up --build                         # anvil on :8545, backend on :8080
```

Then deploy the contract from the host (see [contracts/README.md](contracts/README.md)), put the address in `backend/.env`, and restart the backend. The frontend still runs on the host:

```bash
cd frontend && cp .env.local.example .env.local && npm install && npm run dev
```

Details: [contracts](contracts/README.md) · [backend](backend/README.md) · [frontend](frontend/README.md)

## Manual Test

1. **Deploy** — `anvil` in one terminal, `forge create` in another; note the address.
2. **Index** — point the indexer at it, post a message, and confirm a row lands in SQLite and shows in `GET /api/messages`.
3. **Idempotent** — restart the indexer mid-load; no message is duplicated.
4. **Wallet** — post from the browser, confirm the prompt, and watch the message appear after the receipt.
5. **Relayer (optional)** — post via the backend with a testnet key and confirm the tx lands.

## Development

Run the full suite across all three layers:

```bash
cd contracts && forge build && forge test
cd backend && go vet ./... && go test ./...
cd frontend && npm run lint && npm run build
```

## Out of scope

Mainnet and real funds, cloud deployment / hosting, tokenomics, auth, multi-chain, reorg rollback beyond a documented note, and indexer frameworks (Ponder / Subgraph).

## References

- Foundry — https://book.getfoundry.sh
- go-ethereum — https://geth.ethereum.org/docs
- wagmi — https://wagmi.sh
- Sepolia faucet — https://sepoliafaucet.com
- Alchemy / Infura RPC — https://www.alchemy.com · https://www.infura.io
