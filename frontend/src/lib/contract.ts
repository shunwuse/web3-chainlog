import type { Address } from "viem";

export const CONTRACT_ADDRESS = (process.env.NEXT_PUBLIC_CONTRACT_ADDRESS ??
  "") as Address;

// Minimal ABI, mirroring the backend's: the write the UI makes, the on-chain
// count it shows, and the event the backend indexes. Reads go through the API.
export const CONTRACT_ABI = [
  {
    type: "function",
    name: "post",
    inputs: [{ name: "content", type: "string" }],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "messageCount",
    inputs: [],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
  },
  {
    type: "event",
    name: "MessagePosted",
    inputs: [
      { name: "sender", type: "address", indexed: true },
      { name: "index", type: "uint256", indexed: true },
      { name: "content", type: "string", indexed: false },
      { name: "timestamp", type: "uint256", indexed: false },
    ],
  },
] as const;
