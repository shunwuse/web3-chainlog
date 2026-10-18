-- Embedded into the binary via //go:embed in store.go.

-- Indexer cursor: one row, holding the last block that was indexed.
CREATE TABLE IF NOT EXISTS sync_state (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    last_block INTEGER NOT NULL DEFAULT 0
);

-- Raw decoded MessagePosted logs. UNIQUE(tx_hash, log_index) makes replayed
-- logs insert-free.
CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    block_number INTEGER NOT NULL,
    tx_hash      TEXT    NOT NULL,
    log_index    INTEGER NOT NULL,
    sender       TEXT    NOT NULL,
    content      TEXT    NOT NULL,
    created_at   INTEGER NOT NULL, -- block timestamp, in seconds
    UNIQUE (tx_hash, log_index)
);

-- Read model served by the API.
CREATE TABLE IF NOT EXISTS messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    sender       TEXT    NOT NULL,
    content      TEXT    NOT NULL,
    block_number INTEGER NOT NULL,
    created_at   INTEGER NOT NULL -- block timestamp, in seconds
);
