# web3-chainlog

A practice repo for learning the blockchain / Web3 stack end to end: one Solidity contract, a Go event indexer + HTTP API, and a Next.js dApp, in one repo.

The event indexer is written from scratch with `go-ethereum` rather than an indexing framework. Runs locally with Anvil or on the Sepolia testnet, backed by a single SQLite file. No mainnet, no real funds.

## Status

Built and running locally end to end. The contract and its Foundry tests, the Go indexer + HTTP API, and the dApp (connect wallet, post, list) are all implemented, and the browser flow has been exercised against a local Anvil chain: the wallet writes, the indexer reads the log, and the list comes back from the API. The relayer exists as a package but has no entry point yet — see step 8 in **Build Order**.

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
| Frontend | Wallet connect, signing, reading a contract | Next.js 16 (App Router, TS) + Tailwind 4, `wagmi` 3 + `viem` 2 + TanStack Query |
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
│   ├── foundry.toml          # solc 0.8.24, optimizer on
│   ├── remappings.txt        # forge-std
│   ├── lib/forge-std         # test dependency, pinned submodule
│   ├── src/Guestbook.sol     # the guestbook contract
│   └── test/Guestbook.t.sol  # forge tests
├── backend/                  # Go indexer + HTTP API
│   ├── go.mod
│   ├── main.go               # wires config -> store -> indexer -> api
│   ├── Dockerfile
│   ├── .env.example
│   └── internal/
│       ├── config/config.go  # godotenv + os.Getenv
│       ├── store/store.go    # database/sql + SQLite
│       ├── store/schema.sql  # sync_state / events / messages
│       ├── chain/contract.go # shared client, address, embedded ABI
│       ├── chain/guestbook_abi.json  # the ABI the indexer decodes
│       ├── chain/indexer.go  # ethclient: watch MessagePosted
│       ├── chain/relayer.go  # ethclient: send tx, testnet only
│       └── api/router.go     # chi routes
└── frontend/                 # Next.js dApp
    ├── package.json
    ├── .env.local.example
    └── src/
        ├── app/              # layout, providers, page
        ├── components/       # ConnectButton / MessageList / PostForm
        └── lib/              # contract address + ABI / wagmi config
```

## Interfaces

Three seams connect the layers. Each is owned and documented by the layer that produces it.

- **Contract surface** — the ABI the other two layers talk to: [contracts/README.md](contracts/README.md)
- **Database schema** and **HTTP API** — what the indexer stores and the API serves: [backend/README.md](backend/README.md)
- **Consumer side** — the frontend reads the API and calls the contract: [frontend/README.md](frontend/README.md)

## Build Order

The order this repo was built in, one commit per step; each step runs before the next one is added.

1. **Contract** — `Guestbook.sol`, `forge test` green.
2. **Local chain** — `anvil`, deploy, post a message with `cast` to see the event.
3. **Store** — config + SQLite schema; insert and read rows in a Go test.
4. **Indexer** — watch `MessagePosted`, advance the cursor, prove idempotency on replay.
5. **API** — `chi` routes reading `messages` / `sync_state`.
6. **Frontend read** — list messages from the API.
7. **Frontend write** — wallet connect + `post` via `wagmi`, then refresh.
8. **Relayer (optional)** — backend-signed `post`. The package is here; nothing calls it yet.

## Quick Start

Prerequisites: Go 1.27+, Foundry, Node 20.9+ (Next.js 16's floor). Run the chain + backend either locally or with Docker Compose; the frontend runs on the host either way. Full steps live in each layer's README.

### Local

```bash
# contracts — deploy first: the backend needs the address
cd contracts && forge build && forge test         # then: anvil + forge create --broadcast

# backend
cd backend && cp .env.example .env && go run .    # fill RPC_URL + CONTRACT_ADDRESS (CHAIN_ID defaults to 31337)

# frontend
cd frontend && cp .env.local.example .env.local   # fill NEXT_PUBLIC_CONTRACT_ADDRESS
npm install && npm run dev                        # then open http://localhost:3000
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

## Wallet Setup

The dApp uses whatever injected wallet the browser exposes — MetaMask, Rabby,
Frame, or a hardware wallet's extension. There is no WalletConnect and no hosted
wallet modal, on purpose: both need a cloud project id, and this project stays
local and free. With no extension installed there is nothing to connect to, and
the page says so instead of offering a dead button.

### 1. Add the local chain

MetaMask → **Add network → Add a network manually**:

| Field | Value |
| --- | --- |
| Network name | Anvil |
| RPC URL | `http://127.0.0.1:8545` |
| Chain ID | `31337` |
| Currency symbol | ETH |

The chain ID is what keeps this a different network from mainnet, even though the
unit is also called ETH.

### 2. Fund the address with fake ETH

A new wallet address holds 0 ETH on a fresh Anvil chain, so the wallet cannot pay
gas and disables the confirm button — the usual reason a first post "does not
work". Give yourself test money; no private key needed:

```bash
# 100 ETH of invented money (the value is hex wei). Use your own address.
cast rpc anvil_setBalance 0xYOUR_ADDRESS 0x56BC75E2D63100000
```

The same command works whether Anvil runs on the host or in Compose, because the
port is mapped to `localhost`.

Alternatively import one of Anvil's ten pre-funded accounts: `anvil` prints their
addresses and private keys at startup, and each holds 10,000 ETH. Those keys are
public test keys — never send real funds to them.

### 3. Expect a fee prompt, and know it is fake

The confirmation shows a network fee, because a transaction really does have a
cost. On chain `31337` that cost is paid in Anvil's ETH, which exists only inside
your local chain: a post costs roughly `0.0001`–`0.0003` of money that is worth
nothing. Nothing in this repo can reach real funds.

On Sepolia the same prompt is paid in Sepolia test ETH from a faucet — also free.

### When the wallet will not connect

| Symptom | Cause |
| --- | --- |
| "No injected wallet detected in this browser" | No extension in this browser profile, or the dApp is open in a different browser than the one holding the wallet |
| Connects, but the badge reads `Unsupported · 1` | The wallet sits on mainnet; use the switch buttons the dApp offers |
| Confirm is greyed out, or the wallet reports insufficient funds | The address has 0 ETH on this chain — step 2 |
| "The contract call could not be made" | No contract at that address on the wallet's chain, or `NEXT_PUBLIC_CONTRACT_ADDRESS` is empty |
| Everything breaks after restarting the chain | Anvil keeps no state between runs: redeploy the contract, update the address, and clear the indexer's cursor (`backend/data.db`, or the Compose volume) so it does not resume from blocks that no longer exist |

## Manual Test

1. **Deploy** — `anvil` in one terminal, `forge create --broadcast` in another; note the address.
2. **Index** — point the indexer at it, post a message with `cast`, and confirm a row lands in SQLite and shows in `GET /api/messages`.
3. **Idempotent** — restart the indexer mid-load; no message is duplicated.
4. **Wallet** — post from the browser, confirm the prompt, and watch the message appear after the receipt.
5. **Relayer (optional)** — the `relayer` package signs and sends `post` with a testnet key, but nothing calls it yet: exercise it with a small `main` or a test of your own.

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
