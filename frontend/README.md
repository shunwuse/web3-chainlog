# frontend

Next.js dApp for the guestbook.

## Scope

Connects the wallet, reads messages from the backend API, and posts new messages to the contract. Scaffold stage — pages and components are stubs; wallets and contract calls are not wired yet.

## Layout

- `src/app` — App Router pages (`layout.tsx`, `page.tsx`)
- `src/components` — `ConnectButton`, `MessageList`, `PostForm`
- `src/lib` — `contract.ts` (address + ABI) and `wagmi.ts` (chain / wallet config)

## Config

`frontend/.env.local` (copy from `.env.local.example`):

| Var | Meaning |
| --- | --- |
| `NEXT_PUBLIC_API_URL` | Backend base URL (`http://localhost:8080`) |
| `NEXT_PUBLIC_CHAIN_ID` | Target chain (`11155111`) |
| `NEXT_PUBLIC_CONTRACT_ADDRESS` | Deployed Guestbook address |
| `NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID` | Optional; injected wallets work without it |

## Components

- `ConnectButton` — connect / disconnect the wallet (RainbowKit)
- `MessageList` — fetch and render `GET /api/messages`
- `PostForm` — sign `post(content)` via `wagmi`, then refresh

## Integration

- **Read** — `GET /api/messages` from the backend ([backend/README.md](../backend/README.md)).
- **Write** — the Guestbook contract's `post` via `wagmi`; ABI and address in `lib/contract.ts` ([contracts/README.md](../contracts/README.md)).

## Commands

The frontend runs on the host (not containerized).

```bash
cp .env.local.example .env.local
npm install
npm run dev
npm run lint
npm run build
```

See the [root README](../README.md) for goals, architecture, and build order.
