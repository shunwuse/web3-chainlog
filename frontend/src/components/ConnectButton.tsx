"use client";

import { useSyncExternalStore } from "react";
import {
  useChains,
  useConnect,
  useConnection,
  useConnectors,
  useDisconnect,
  useSwitchChain,
} from "wagmi";

import { BaseError } from "viem";

/** 0xf39F…a48E — enough to recognise an account, short enough for a header. */
function shorten(address: string): string {
  return `${address.slice(0, 6)}…${address.slice(-4)}`;
}

/** `walk` hands the predicate unknown values, so narrow before reading `.name`. */
function hasName(name: string): (candidate: unknown) => boolean {
  return (candidate) =>
    typeof candidate === "object" &&
    candidate !== null &&
    (candidate as { name?: unknown }).name === name;
}

/** Wallet errors are for people, not for logs. */
function describeWalletError(error: unknown): string {
  if (!(error instanceof BaseError)) return "Something went wrong. Try again.";

  if (error.walk(hasName("UserRejectedRequestError"))) {
    return "Cancelled — the wallet declined the request.";
  }
  if (error.walk(hasName("ConnectorNotFoundError"))) {
    return "No browser wallet found. Install an injected wallet and reload.";
  }
  if (error.walk(hasName("ProviderNotFoundError"))) {
    return "That wallet did not expose an injected provider. Try another one.";
  }

  return (error.shortMessage || error.message).split("\n")[0];
}

export default function ConnectButton() {
  const { status, address, chainId } = useConnection();
  const connectors = useConnectors();
  const connect = useConnect();
  const disconnect = useDisconnect();
  const switchChain = useSwitchChain();
  const chains = useChains();

  // Browser-only fact, read without an effect: `null` on the server, then the
  // real answer after hydration.
  const hasInjectedWallet = useSyncExternalStore(
    () => () => {},
    () => typeof window !== "undefined" && "ethereum" in window,
    () => null,
  );

  // False on the server *and* during hydration, true only once mounted — so the
  // first client render matches the server's instead of wagmi's reconnect state.
  const mounted = useSyncExternalStore(
    () => () => {},
    () => true,
    () => false,
  );

  const connector = connectors[0];
  const connecting = mounted && (status === "connecting" || connect.isPending);
  const connectError = connect.error ?? switchChain.error;

  if (!mounted || status !== "connected") {
    return (
      <div className="flex flex-col items-end gap-2">
        <button
          type="button"
          onClick={() => connect.mutate({ connector })}
          disabled={!connector || connecting}
          className="rounded-sm bg-seal px-4 py-2 font-mono text-xs tracking-[0.12em] text-background uppercase transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-45"
        >
          {connecting ? "Connecting…" : "Connect wallet"}
        </button>

        {connect.error ? (
          <p role="alert" className="max-w-64 text-right font-mono text-xs text-seal">
            {describeWalletError(connect.error)}
          </p>
        ) : hasInjectedWallet === false ? (
          <p className="max-w-64 text-right font-mono text-xs text-muted">
            No injected wallet detected in this browser.
          </p>
        ) : null}
      </div>
    );
  }

  const network = chains.find((chain) => chain.id === chainId);

  return (
    <div className="flex flex-col items-end gap-2">
      {/* An identity plate: the network is the loud fact, the address is the
          quiet one, and disconnect is deliberately unbuttoned. */}
      <div className="flex items-center gap-3 rounded-sm border border-rule px-3 py-1.5">
        <span
          className={`font-mono text-xs tracking-[0.12em] uppercase ${
            network ? "text-foreground" : "text-seal"
          }`}
        >
          {network ? network.name : `Unsupported · ${chainId}`}
        </span>
        <span aria-hidden="true" className="h-3 w-px bg-rule" />
        <span className="font-mono text-xs text-muted" title={address}>
          <span className="sr-only">Wallet address </span>
          {shorten(address)}
        </span>
        <button
          type="button"
          onClick={() => disconnect.mutate()}
          className="rounded-sm font-mono text-xs text-muted underline decoration-rule underline-offset-4 transition-colors hover:text-seal"
        >
          Disconnect
        </button>
      </div>

      {connectError ? (
        <p role="alert" className="max-w-64 text-right font-mono text-xs text-seal">
          {describeWalletError(connectError)}
        </p>
      ) : null}

      {!network ? (
        <div className="max-w-72 border-l-2 border-seal bg-seal-tint px-3 py-2">
          <p className="text-xs text-foreground">
            This wallet sits on a network the contract is not deployed on, so
            posting will fail until you switch.
          </p>
          <div className="mt-2 flex flex-wrap justify-end gap-2">
            {chains.map((chain) => (
              <button
                key={chain.id}
                type="button"
                onClick={() => switchChain.mutate({ chainId: chain.id })}
                disabled={switchChain.isPending}
                className="rounded-sm border border-rule px-2 py-1 font-mono text-[11px] tracking-[0.1em] uppercase transition-colors hover:border-seal hover:text-seal disabled:cursor-not-allowed disabled:opacity-50"
              >
                {chain.name}
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}