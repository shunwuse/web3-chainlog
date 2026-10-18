"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** Shared with PostForm so a confirmed post can refresh exactly this list. */
export const MESSAGES_QUERY_KEY = ["messages"] as const;

/** The indexer's head. Both are read from the same database, so they are always
 *  refreshed together — a head behind its own entries would be a lie. */
export const SYNC_QUERY_KEY = ["sync"] as const;

type ApiMessage = {
  sender: string;
  content: string;
  blockNumber: number;
  timestamp: number;
};

type Message = {
  sender: string;
  content: string;
  blockNumber: number;
  timestamp: number;
};

function shorten(address: string): string {
  return address.length > 14 ? `${address.slice(0, 8)}…${address.slice(-6)}` : address;
}

function localTime(unixSeconds: number): string {
  const date = new Date(unixSeconds * 1000);
  if (Number.isNaN(date.getTime())) return "unknown time";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function isoTime(unixSeconds: number): string | undefined {
  const date = new Date(unixSeconds * 1000);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

/**
 * The API is another team's surface; treat a malformed row as a skipped row
 * rather than letting one bad field take the whole page down.
 */
function normalise(raw: unknown): Message | null {
  if (typeof raw !== "object" || raw === null) return null;
  const row = raw as Partial<ApiMessage>;
  if (typeof row.content !== "string") return null;

  return {
    sender: typeof row.sender === "string" ? row.sender : "unknown",
    content: row.content,
    blockNumber: typeof row.blockNumber === "number" ? row.blockNumber : 0,
    timestamp: typeof row.timestamp === "number" ? row.timestamp : 0,
  };
}

async function fetchMessages(signal: AbortSignal): Promise<Message[]> {
  const url = `${API_URL}/api/messages?limit=50`;

  let response: Response;
  try {
    response = await fetch(url, { signal });
  } catch (error) {
    // React Query cancels in-flight requests on refetch; that is not a failure.
    if (signal.aborted) throw error;
    throw new Error(`The indexer API at ${API_URL} did not answer. Is the backend running?`);
  }

  if (!response.ok) {
    throw new Error(`The indexer API replied ${response.status} ${response.statusText}.`);
  }

  const body: unknown = await response.json();
  const rows =
    typeof body === "object" && body !== null && Array.isArray((body as { messages?: unknown }).messages)
      ? ((body as { messages: unknown[] }).messages)
      : [];

  return rows.map(normalise).filter((row): row is Message => row !== null);
}

async function fetchLastBlock(signal: AbortSignal): Promise<number | null> {
  try {
    const response = await fetch(`${API_URL}/api/sync`, { signal });
    if (!response.ok) return null;
    const body: unknown = await response.json();
    const lastBlock = (body as { lastBlock?: unknown } | null)?.lastBlock;
    return typeof lastBlock === "number" ? lastBlock : null;
  } catch {
    return null;
  }
}

function RefreshIcon({ spinning }: { spinning: boolean }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 16 16"
      className={`size-3.5 ${spinning ? "animate-spin" : ""}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M14 8a6 6 0 1 1-1.8-4.3" />
      <path d="M14 2.5V6h-3.5" />
    </svg>
  );
}

export default function MessageList() {
  const queryClient = useQueryClient();

  const messages = useQuery({
    queryKey: MESSAGES_QUERY_KEY,
    queryFn: ({ signal }) => fetchMessages(signal),
    retry: 1,
  });

  const sync = useQuery({
    queryKey: SYNC_QUERY_KEY,
    queryFn: ({ signal }) => fetchLastBlock(signal),
    staleTime: 15_000,
  });

  const count = messages.data?.length ?? 0;

  return (
    <section aria-labelledby="log-heading" className="mt-12">
      <div className="flex items-end justify-between gap-4 border-b border-rule pb-2">
        <div>
          <h2
            id="log-heading"
            className="font-mono text-[11px] tracking-[0.2em] text-foreground uppercase"
          >
            The log
          </h2>
          <p className="mt-1 font-mono text-[11px] text-muted">
            {sync.data == null
              ? "read from the indexer"
              : `read from the indexer · indexed through block ${sync.data}`}
          </p>
        </div>

        <div className="flex items-center gap-3">
          <p aria-live="polite" className="font-mono text-[11px] text-muted">
            {messages.isError
              ? "unavailable"
              : messages.isFetching
                ? "syncing…"
                : `${count} ${count === 1 ? "entry" : "entries"}`}
          </p>
          <button
            type="button"
            onClick={() => void Promise.all([messages.refetch(), sync.refetch()])}
            disabled={messages.isFetching}
            className="flex items-center gap-1.5 rounded-sm border border-rule px-2 py-1 font-mono text-[11px] tracking-[0.1em] uppercase transition-colors hover:border-seal hover:text-seal disabled:cursor-not-allowed disabled:opacity-50"
          >
            <RefreshIcon spinning={messages.isFetching} />
            Refresh
          </button>
        </div>
      </div>

      <div aria-busy={messages.isPending}>
        {messages.isPending ? <SkeletonRows /> : null}

        {messages.isError ? (
          <div role="alert" className="mt-6 border-l-2 border-seal bg-seal-tint px-4 py-4">
            <p className="font-mono text-[11px] tracking-[0.2em] text-seal uppercase">
              Can&apos;t read the log
            </p>
            <p className="mt-2 text-sm text-foreground">
              {messages.error instanceof Error
                ? messages.error.message
                : "The guestbook API did not answer."}
            </p>
            <button
              type="button"
              onClick={() => void queryClient.invalidateQueries({ queryKey: MESSAGES_QUERY_KEY })}
              className="mt-3 rounded-sm border border-seal px-3 py-1.5 font-mono text-[11px] tracking-[0.1em] text-seal uppercase transition-colors hover:bg-seal hover:text-background"
            >
              Try again
            </button>
          </div>
        ) : null}

        {messages.isSuccess && count === 0 ? (
          <div className="mt-6 border border-dashed border-rule px-5 py-10 text-center">
            <p className="font-mono text-[11px] tracking-[0.2em] text-muted uppercase">
              Nothing signed yet
            </p>
            <p className="mx-auto mt-3 max-w-sm text-sm text-foreground">
              The guestbook is empty. Sign your first message above — it shows up
              here a moment after the chain confirms it.
            </p>
          </div>
        ) : null}

        {messages.isSuccess && count > 0 ? (
          <ol className="mt-6">
            {messages.data?.map((message, index) => (
              <li
                key={`${message.blockNumber}-${index}`}
                className="grid grid-cols-[2.5rem_minmax(0,1fr)] sm:grid-cols-[3.25rem_minmax(0,1fr)]"
              >
                {/* The margin: an entry number, then one continuous rule down
                    the page with a stamp dot on every entry. */}
                <div className="border-r border-rule pr-3 text-right font-mono text-[11px] leading-6 text-muted tabular-nums">
                  {String(index + 1).padStart(2, "0")}
                </div>

                <div className="relative min-w-0 py-4 pl-4 sm:pl-6">
                  <span
                    aria-hidden="true"
                    className="absolute top-[1.4rem] -left-[2.5px] size-[5px] bg-seal"
                  />
                  <p className="text-[0.9375rem] leading-relaxed break-words whitespace-pre-wrap">
                    {message.content}
                  </p>
                  <p className="mt-2 flex flex-wrap items-center gap-x-2 font-mono text-[11px] text-muted">
                    <span className="tabular-nums">{shorten(message.sender)}</span>
                    <span aria-hidden="true">·</span>
                    <span className="tabular-nums">block {message.blockNumber}</span>
                    <span aria-hidden="true">·</span>
                    <time dateTime={isoTime(message.timestamp)}>
                      {localTime(message.timestamp)}
                    </time>
                  </p>
                </div>
              </li>
            ))}
          </ol>
        ) : null}
      </div>
    </section>
  );
}

function SkeletonRows() {
  return (
    <div className="mt-6 animate-pulse" aria-hidden="true">
      {[0, 1, 2].map((row) => (
        <div key={row} className="grid grid-cols-[2.5rem_minmax(0,1fr)] sm:grid-cols-[3.25rem_minmax(0,1fr)]">
          <div className="border-r border-rule pr-3 text-right font-mono text-[11px] leading-6 text-muted/50 tabular-nums">
            0{row + 1}
          </div>
          <div className="space-y-2 py-4 pl-4 sm:pl-6">
            <div className="h-3 w-11/12 bg-rule" />
            <div className="h-3 w-2/5 bg-rule" />
          </div>
        </div>
      ))}
    </div>
  );
}