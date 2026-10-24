"use client";

import { useEffect, useId, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { BaseError, formatUnits } from "viem";
import {
  useBalance,
  useChains,
  useConnection,
  useWaitForTransactionReceipt,
  useWriteContract,
} from "wagmi";

import { MESSAGES_QUERY_KEY, SYNC_QUERY_KEY } from "@/components/MessageList";
import { CONTRACT_ABI, CONTRACT_ADDRESS } from "@/lib/contract";

/** Long enough to say something, short enough to keep the log readable. */
const MAX_LENGTH = 280;

/** `walk` hands the predicate unknown values, so narrow before reading `.name`. */
function hasName(name: string): (candidate: unknown) => boolean {
  return (candidate) =>
    typeof candidate === "object" &&
    candidate !== null &&
    (candidate as { name?: unknown }).name === name;
}

/**
 * Turn a wallet / viem error into one sentence a non-engineer can act on.
 * Everything nested (rejection, revert reason, gas) lives on the cause chain,
 * so walk it rather than matching on the outermost wrapper.
 */
function describeError(error: unknown): string {
  if (!(error instanceof BaseError)) {
    return "Something went wrong while talking to your wallet. Try again.";
  }

  const find = (name: string): BaseError | null =>
    error.walk(hasName(name)) as BaseError | null;

  if (find("UserRejectedRequestError")) {
    return "Cancelled — you declined the signature, so nothing was sent.";
  }

  const revert = find("ContractFunctionRevertedError");
  if (revert) {
    const reason =
      "reason" in revert && typeof revert.reason === "string"
        ? revert.reason
        : revert.details;
    return reason
      ? `The contract refused the message: ${reason}`
      : "The contract refused the message and the transaction reverted.";
  }

  if (find("ContractFunctionExecutionError")) {
    return "The contract call could not be made. Check that the contract address is set and deployed on this network.";
  }
  if (find("ChainMismatchError")) {
    return "Your wallet is on the wrong network for this contract. Switch networks and try again.";
  }
  if (find("InsufficientFundsError")) {
    return "This account does not have enough ETH to pay for gas.";
  }
  if (find("NonceTooLowError")) {
    return "A transaction from this account is still pending. Wait a moment, then try again.";
  }
  if (find("TransactionReceiptNotFoundError")) {
    return "The transaction was sent but never produced a receipt. It may still be in the mempool — check your wallet.";
  }
  if (find("ExecutionRevertedError")) {
    return "The transaction reverted on chain.";
  }

  const firstLine = (error.shortMessage || error.message).split("\n")[0];
  return firstLine.length > 160 ? `${firstLine.slice(0, 157)}…` : firstLine;
}

export default function PostForm() {
  const { status, chainId, address } = useConnection();
  const chains = useChains();
  const queryClient = useQueryClient();

  // Every chain that carries a contract here is a test chain, but the two behave
  // differently: Anvil invents its own ETH, Sepolia hands it out from a faucet.
  const onConfiguredChain = chains.some((chain) => chain.id === chainId);
  const activeChain = chains.find((candidate) => candidate.id === chainId);

  // Disabled (address undefined) until connected to a chain the config knows.
  const balance = useBalance({ address: onConfiguredChain ? address : undefined, chainId });

  const [content, setContent] = useState("");
  const fieldId = useId();
  const errorId = useId();
  const statusId = useId();

  const write = useWriteContract();
  const receipt = useWaitForTransactionReceipt({ hash: write.data });

  const contractMissing = !CONTRACT_ADDRESS;
  const tooLong = content.length > MAX_LENGTH;
  const empty = content.trim().length === 0;

  // The wallet will happily open a confirmation for a transaction it cannot pay
  // for, so say it plainly here instead of letting the prompt fail.
  const symbol = balance.data?.symbol ?? activeChain?.nativeCurrency.symbol ?? "ETH";
  const wei = balance.data?.value;
  const broke = status === "connected" && onConfiguredChain && wei === BigInt(0);
  const balanceText =
    wei === undefined
      ? null
      : Number(formatUnits(wei, balance.data?.decimals ?? 18)).toLocaleString("en-US", {
          maximumFractionDigits: 4,
        });

  // A fee prompt on a test chain is real UI around fake money. Say which kind of
  // fake it is, or a first-time user reads it as a bill.
  const gasMoney =
    chainId === 31337
      ? "Gas is paid in Anvil's test ETH — it exists only on your machine and is worth nothing, so the fee your wallet shows is not real money."
      : chainId === 11155111
        ? "Gas is paid in Sepolia test ETH, which comes free from a faucet. No real money."
        : null;

  // react-query reports a *disabled* query as pending, so receipt.isPending is
  // true before a hash exists; isLoading only when a receipt is actually awaited.
  const receiptPending = write.data !== undefined && receipt.isLoading;

  const submitting = write.isPending || receiptPending;
  const lockedReason = contractMissing
    ? "No contract address configured. Set NEXT_PUBLIC_CONTRACT_ADDRESS to post."
    : status !== "connected"
      ? "Connect a wallet to post."
      : !onConfiguredChain
        ? "Switch to a configured network to post."
        : null;

  const canSubmit = lockedReason === null && !empty && !tooLong && !submitting;

  // The chain has the message; the indexer needs a beat to catch up. Refetch
  // now, then once more after it has had time to poll the new block.
  const confirmedHash = receipt.data?.transactionHash ?? null;

  // Clear the field when a *new* transaction confirms. This is React's
  // "adjust state while rendering" pattern: the render that observes the new
  // hash also empties the box, instead of an effect causing a second render.
  const [clearedHash, setClearedHash] = useState<string | null>(null);
  if (confirmedHash !== null && confirmedHash !== clearedHash) {
    setClearedHash(confirmedHash);
    setContent("");
  }

  useEffect(() => {
    if (confirmedHash === null) return;

    void queryClient.invalidateQueries({ queryKey: MESSAGES_QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: SYNC_QUERY_KEY });
    const followUp = setTimeout(() => {
      void queryClient.invalidateQueries({ queryKey: MESSAGES_QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: SYNC_QUERY_KEY });
    }, 2500);
    return () => clearTimeout(followUp);
  }, [confirmedHash, queryClient]);

  const failure = write.error ?? receipt.error ?? undefined;

  const stage: "idle" | "signature" | "pending" | "confirmed" =
    write.isPending
      ? "signature"
      : receiptPending
        ? "pending"
        : receipt.isSuccess
          ? "confirmed"
          : "idle";

  return (
    <section aria-labelledby="compose-heading" className="mt-10">
      <div className="flex items-end justify-between gap-4 border-b border-rule pb-2">
        <h2
          id="compose-heading"
          className="font-mono text-[11px] tracking-[0.2em] text-foreground uppercase"
        >
          Sign the book
        </h2>
        <p className="font-mono text-[11px] text-muted">posts a transaction on-chain</p>
      </div>

      {status === "connected" && onConfiguredChain ? (
        <div className="mt-3">
          {broke ? (
            <div className="border-l-2 border-seal bg-seal-tint px-3 py-2.5">
              <p className="text-xs leading-relaxed text-foreground">
                <span className="font-mono">
                  0 {symbol}
                </span>{" "}
                on this network, so there is nothing to pay gas with and the wallet
                will refuse to sign.
                {chainId === 31337
                  ? " Anvil money is free — fund this address from a terminal:"
                  : " Get free test ETH from a Sepolia faucet, then try again."}
              </p>
              {chainId === 31337 && address ? (
                <code className="mt-2 block overflow-x-auto rounded-sm bg-background px-2 py-1.5 font-mono text-[11px] whitespace-nowrap">
                  cast rpc anvil_setBalance {address} 0x56BC75E2D63100000
                </code>
              ) : null}
            </div>
          ) : gasMoney ? (
            <p className="text-xs leading-relaxed text-muted">
              {balanceText ? (
                <span className="font-mono text-foreground">
                  {balanceText} {symbol}
                </span>
              ) : null}
              {balanceText ? " · " : null}
              {gasMoney}
            </p>
          ) : null}
        </div>
      ) : null}

      <form
        className="mt-5"
        onSubmit={(event) => {
          event.preventDefault();
          if (!canSubmit) return;
          write.reset();
          write.mutate({
            abi: CONTRACT_ABI,
            address: CONTRACT_ADDRESS,
            functionName: "post",
            args: [content.trim()],
          });
        }}
      >
        <label htmlFor={fieldId} className="block text-sm text-foreground">
          Your message
        </label>
        <p id={`${fieldId}-hint`} className="mt-1 text-xs text-muted">
          Stored on chain forever. Everyone who reads this book will see it.
        </p>

        <textarea
          id={fieldId}
          name="content"
          rows={3}
          value={content}
          disabled={lockedReason !== null || submitting}
          aria-describedby={`${fieldId}-hint ${statusId}${failure ? ` ${errorId}` : ""}`}
          aria-invalid={tooLong || failure !== undefined}
          onChange={(event) =>
            // Hard cap one character past the limit so the counter can still
            // show the over-length state instead of silently truncating.
            setContent(event.target.value.slice(0, MAX_LENGTH + 1))
          }
          placeholder="It was a dark and stormy night…"
          className={`mt-2 block w-full resize-y rounded-sm border bg-background px-3 py-2.5 font-mono text-sm leading-relaxed placeholder:text-muted/60 disabled:opacity-60 ${
            tooLong || failure ? "border-seal" : "border-rule"
          }`}
        />

        <div className="mt-2 flex items-center justify-between gap-4">
          <p className="font-mono text-[11px] text-muted">
            {lockedReason ??
              (tooLong
                ? "Too long — trim a little."
                : `${content.length} / ${MAX_LENGTH}`)}
          </p>

          <button
            type="submit"
            disabled={!canSubmit}
            className="rounded-sm bg-seal px-4 py-2 font-mono text-xs tracking-[0.12em] text-background uppercase transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-45"
          >
            {stage === "signature"
              ? "Confirm in wallet…"
              : stage === "pending"
                ? "Posting…"
                : "Post message"}
          </button>
        </div>
      </form>

      {/* Transaction lifecycle. One polite live region so a screen reader hears
          the progress once, not three times. */}
      <div className="mt-5 border border-rule px-3 py-2.5">
        <ol className="flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[11px] tracking-[0.1em] uppercase">
          {(
            [
              ["signature", "Awaiting signature"],
              ["pending", "Pending"],
              ["confirmed", "Confirmed"],
            ] as const
          ).map(([key, label], index) => {
            const order = ["signature", "pending", "confirmed"];
            const done = stage === "idle" ? -1 : order.indexOf(stage);
            const active = done === index;
            return (
              <li key={key} className="flex items-center gap-2">
                {index > 0 ? <span aria-hidden="true" className="text-rule">/</span> : null}
                <span
                  className={
                    active
                      ? "text-seal"
                      : done > index
                        ? "text-foreground"
                        : "text-muted"
                  }
                >
                  {label}
                </span>
              </li>
            );
          })}
        </ol>

        <p id={statusId} aria-live="polite" className="mt-2 text-xs text-muted">
          {stage === "signature" ? "Open your wallet and approve the transaction." : null}
          {stage === "pending"
            ? "Signed and broadcast. Waiting for the chain to include it in a block."
            : null}
          {stage === "confirmed"
            ? `Confirmed in block ${receipt.data?.blockNumber}. The indexer will surface it in a moment.`
            : null}
          {stage === "idle" ? "No transaction in flight." : null}
        </p>

        {write.data ? (
          <p className="mt-2 font-mono text-[11px] break-all text-muted">
            <span className="text-foreground">tx</span> {write.data}
          </p>
        ) : null}
      </div>

      {failure ? (
        <p
          id={errorId}
          role="alert"
          className="mt-3 border-l-2 border-seal bg-seal-tint px-3 py-2.5 text-sm"
        >
          <span className="font-mono text-[11px] tracking-[0.2em] text-seal uppercase">
            Post failed
          </span>
          <span className="mt-1.5 block text-foreground">{describeError(failure)}</span>
        </p>
      ) : null}
    </section>
  );
}