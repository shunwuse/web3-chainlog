import ConnectButton from "@/components/ConnectButton";
import MessageList from "@/components/MessageList";
import PostForm from "@/components/PostForm";

export default function Home() {
  return (
    <div className="mx-auto flex w-full max-w-2xl grow flex-col px-5 pt-10 pb-20 sm:px-8 sm:pt-14">
      <header className="flex flex-col gap-6 border-b border-foreground pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="font-mono text-[11px] tracking-[0.28em] text-muted uppercase">
            Solidity → indexer → SQLite
          </p>
          <h1 className="mt-3 text-[clamp(2rem,7vw,2.75rem)] leading-[1.05] font-semibold tracking-tight">
            The Chainlog
          </h1>
          <p className="mt-3 max-w-md text-sm leading-relaxed text-muted">
            A guestbook that lives on chain. Every entry below was signed by a
            wallet and read back through an indexer that never trusts the page.
          </p>
        </div>

        <div className="shrink-0">
          <ConnectButton />
        </div>
      </header>

      <PostForm />
      <MessageList />

      <footer className="mt-16 border-t border-rule pt-4">
        <p className="font-mono text-[11px] leading-relaxed text-muted">
          Practice project. Local chain only — testnet ETH, never mainnet funds.
        </p>
      </footer>
    </div>
  );
}