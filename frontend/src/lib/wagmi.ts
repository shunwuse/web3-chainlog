import { defineChain } from "viem";
import { createConfig, http } from "wagmi";
import { sepolia } from "wagmi/chains";
import { injected } from "wagmi/connectors/injected";

// Anvil, the local chain started by `anvil` or by docker compose (chain id 31337).
export const anvil = defineChain({
  id: 31337,
  name: "Anvil",
  nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
  rpcUrls: { default: { http: ["http://127.0.0.1:8545"] } },
});

// getConfig returns a fresh config per call, so the server and the browser never
// share one mutable instance. Only injected (browser-extension) wallets are set
// up: WalletConnect would need a cloud project id, and this project stays local.
export function getConfig() {
  return createConfig({
    chains: [sepolia, anvil],
    connectors: [injected()],
    ssr: true,
    transports: {
      [sepolia.id]: http(),
      [anvil.id]: http(),
    },
  });
}
