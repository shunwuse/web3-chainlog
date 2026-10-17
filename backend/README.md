# backend

Go event indexer + HTTP API for the guestbook.

## Scope

Watches the contract's `MessagePosted` events, stores them in SQLite, and serves them over HTTP for the frontend. Can optionally post transactions itself (relayer). Scaffold stage — all packages are stubs.

## Layout

- `main.go` — wires config → store → indexer → api
- `internal/config` — config loading (`godotenv` + `os.Getenv`)
- `internal/store` — SQLite access (`database/sql` + `modernc.org/sqlite`); `schema.sql` holds the schema
- `internal/chain` — `indexer` (watch events) and `relayer` (send tx, testnet only)
- `internal/api` — `chi` router

## Config

`backend/.env` (copy from `.env.example`):

| Var | Meaning | Secret |
| --- | --- | --- |
| `RPC_URL` | Node endpoint (`http://127.0.0.1:8545` local, a Sepolia RPC otherwise) | no |
| `CHAIN_ID` | `31337` Anvil, `11155111` Sepolia | no |
| `CONTRACT_ADDRESS` | Deployed Guestbook address | no |
| `PRIVATE_KEY` | Relayer key — **testnet only, never a funded key** | **yes** |
| `DB_PATH` | SQLite file path (`./data.db`) | no |
| `API_PORT` | HTTP port (`8080`) | no |

`.env*` is gitignored; only `.env.example` is tracked.

## Database schema

```sql
sync_state(id, last_block)                                        -- indexer cursor
events(id, block_number, tx_hash, log_index, sender, content,     -- raw log, UNIQUE(tx_hash, log_index)
       created_at)
messages(id, sender, content, block_number, created_at)           -- read model for the API
```

## HTTP API

| Method | Path | Response |
| --- | --- | --- |
| GET | `/api/health` | `{"ok":true}` |
| GET | `/api/messages?limit=&offset=` | `{"messages":[{"sender","content","blockNumber","timestamp"}]}` |
| GET | `/api/sync` | `{"lastBlock":0}` |

## Indexer

Reads logs after `sync_state.last_block`, decodes `MessagePosted`, inserts into `events` (idempotent on `(tx_hash, log_index)`) and `messages`, then advances the cursor. Replays never double-write.

## Relayer (optional)

Lets the backend call `post` itself with a testnet key, to exercise nonce / gas / tx lifecycle. Not wired to the API by default.

## Docker

`Dockerfile` builds a static binary (`CGO_ENABLED=0`) and runs it on a distroless image. From the repo root, Compose starts the chain and the backend together:

```bash
cp backend/.env.example backend/.env    # fill CONTRACT_ADDRESS after deploy
docker compose up --build
```

Compose starts `anvil` (chain id 31337) and `backend`, overrides `RPC_URL` to the anvil service, and keeps the SQLite file in a named volume. Deploy the contract from the host (see [contracts/README.md](../contracts/README.md)), then restart the backend with the address set. The frontend is not containerized.

## Commands

```bash
cp .env.example .env
go mod tidy
go run .
go vet ./...
go test ./...
```

See the [root README](../README.md) for goals, architecture, and build order.
