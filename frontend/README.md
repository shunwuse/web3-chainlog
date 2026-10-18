# frontend

Next.js dApp for the guestbook.

## Scope

Connects an injected wallet, reads messages from the backend API, and posts new messages to the contract. The list is read from the API only — the chain is never queried for it — and it refreshes after a post confirms.

## Layout

- `src/app` — App Router pages (`layout.tsx`, `page.tsx`) and `providers.tsx` (wagmi + TanStack Query)
- `src/components` — `ConnectButton`, `MessageList`, `PostForm`
- `src/lib` — `contract.ts` (address + minimal ABI) and `wagmi.ts` (chains / connector config)

## Config

`frontend/.env.local` (copy from `.env.local.example`):

| Var | Meaning |
| --- | --- |
| `NEXT_PUBLIC_API_URL` | Backend base URL (`http://localhost:8080`) |
| `NEXT_PUBLIC_CONTRACT_ADDRESS` | Deployed Guestbook address |

Those two are the whole config. The chain is not set here: `src/lib/wagmi.ts` targets Sepolia (`11155111`) and Anvil (`31337`) with the injected connector — no WalletConnect project id, no cloud service.

## Components

- `ConnectButton` — connect / disconnect the injected wallet, show the active network, and offer a switch when it is not one of the two configured chains
- `MessageList` — fetch `GET /api/messages` and render the log, with loading / empty / error states and a manual refresh
- `PostForm` — sign `post(content)` via `wagmi`, wait for the receipt, then refresh the list

## Integration

- **Read** — `GET /api/messages` from the backend ([backend/README.md](../backend/README.md)).
- **Write** — the Guestbook contract's `post` via `wagmi`; ABI and address in `lib/contract.ts` ([contracts/README.md](../contracts/README.md)).

## Commands

The frontend runs on the host (not containerized). It needs the backend running for the list; connect a wallet to post.

```bash
cp .env.local.example .env.local   # set NEXT_PUBLIC_CONTRACT_ADDRESS
npm install
npm run dev                        # http://localhost:3000
npm run lint
npm run build
```

See the [root README](../README.md) for goals, architecture, and build order.
