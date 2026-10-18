# contracts

Solidity guestbook contract + Foundry tests.

## Scope

The on-chain side: a minimal `Guestbook` where anyone can `post` a message and read it back. No access control, no fees. Covered by Foundry tests: posting, the getters, the `MessagePosted` event, the empty-content revert, and the `getMessages` slicing rules.

## Files

- `src/Guestbook.sol` — the contract
- `test/Guestbook.t.sol` — Foundry tests
- `foundry.toml` — solc `0.8.24`, optimizer on
- `remappings.txt` — `forge-std`

## Contract surface

```solidity
struct Message { address sender; string content; uint256 timestamp; }

function post(string calldata content) external;
event MessagePosted(address indexed sender, uint256 indexed index, string content, uint256 timestamp);
function messageCount() external view returns (uint256);
function getMessage(uint256 index) external view returns (Message);
function getMessages(uint256 offset, uint256 limit) external view returns (Message[]);
```

`MessagePosted` is the indexer's input; the backend decodes it (see [backend/README.md](../backend/README.md)).

## Commands

```bash
foundryup            # install / update Foundry
forge build
forge test
```

## Deploy

Local:

```bash
anvil                # chain id 31337, http://127.0.0.1:8545
forge create --rpc-url http://127.0.0.1:8545 \
  --private-key <anvil-private-key> \
  --broadcast \
  src/Guestbook.sol:Guestbook
```

Sepolia (testnet):

```bash
forge create --rpc-url $SEPOLIA_RPC_URL \
  --private-key $PRIVATE_KEY \
  --broadcast \
  src/Guestbook.sol:Guestbook
```

`--broadcast` is what actually sends the transaction; without it `forge create` only simulates and prints it.

Deploy-time env: `SEPOLIA_RPC_URL`, `PRIVATE_KEY` (**testnet only**). This key is only used to deploy the contract; the backend's relayer key is a separate one.

See the [root README](../README.md) for goals, architecture, and build order.
