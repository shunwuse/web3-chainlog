// Package chain talks to the Ethereum node: indexing MessagePosted logs and,
// optionally, relaying transactions.
package chain

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/yourname/web3-chainlog/backend/internal/config"
	"github.com/yourname/web3-chainlog/backend/internal/store"
)

//go:embed guestbook_abi.json
var guestbookABI string

// pollInterval is how often the indexer looks for new logs.
const pollInterval = 2 * time.Second

// Indexer watches MessagePosted logs and persists them through the store.
type Indexer struct {
	client   *ethclient.Client
	contract common.Address
	event    abi.Event
	store    *store.Store
	logger   *slog.Logger
}

// NewIndexer connects to the node and prepares to index the guestbook contract.
func NewIndexer(ctx context.Context, cfg *config.Config, st *store.Store, logger *slog.Logger) (*Indexer, error) {
	if logger == nil {
		logger = slog.Default()
	}

	client, err := ethclient.DialContext(ctx, cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.RPCURL, err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("read chain id: %w", err)
	}
	if chainID.Int64() != cfg.ChainID {
		client.Close()
		return nil, fmt.Errorf("chain id mismatch: node reports %d, config says %d", chainID.Int64(), cfg.ChainID)
	}

	if !common.IsHexAddress(cfg.ContractAddress) {
		client.Close()
		return nil, fmt.Errorf("invalid contract address %q", cfg.ContractAddress)
	}

	parsed, err := abi.JSON(strings.NewReader(guestbookABI))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("parse abi: %w", err)
	}
	event, ok := parsed.Events["MessagePosted"]
	if !ok {
		client.Close()
		return nil, errors.New("abi has no MessagePosted event")
	}

	return &Indexer{
		client:   client,
		contract: common.HexToAddress(cfg.ContractAddress),
		event:    event,
		store:    st,
		logger:   logger,
	}, nil
}

// Close releases the RPC connection.
func (i *Indexer) Close() {
	i.client.Close()
}

// Run indexes new logs until ctx is cancelled. Transient failures are logged and
// retried on the next tick rather than killing the loop.
func (i *Indexer) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if err := i.syncOnce(ctx); err != nil && ctx.Err() == nil {
			i.logger.Error("indexer sync failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// syncOnce indexes every log between the stored cursor and the chain head.
func (i *Indexer) syncOnce(ctx context.Context) error {
	lastIndexed, err := i.store.GetLastBlock(ctx)
	if err != nil {
		return err
	}

	latest, err := i.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("read latest block: %w", err)
	}
	if latest <= lastIndexed {
		return nil
	}

	logs, err := i.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(lastIndexed + 1),
		ToBlock:   new(big.Int).SetUint64(latest),
		Addresses: []common.Address{i.contract},
		Topics:    [][]common.Hash{{i.event.ID}},
	})
	if err != nil {
		return fmt.Errorf("filter logs: %w", err)
	}

	for _, log := range logs {
		event, err := i.decode(log)
		if err != nil {
			return err
		}
		if err := i.store.InsertEvent(ctx, event); err != nil {
			return err
		}
		i.logger.Info("indexed message", "block", event.BlockNumber, "tx", event.TxHash)
	}

	if err := i.store.SetLastBlock(ctx, latest); err != nil {
		return fmt.Errorf("advance cursor: %w", err)
	}
	return nil
}

// decode turns a MessagePosted log into a store.Event.
func (i *Indexer) decode(log types.Log) (store.Event, error) {
	if len(log.Topics) != 3 {
		return store.Event{}, fmt.Errorf("log %s: expected 3 topics, got %d", log.TxHash.Hex(), len(log.Topics))
	}

	data, err := i.event.Inputs.NonIndexed().UnpackValues(log.Data)
	if err != nil {
		return store.Event{}, fmt.Errorf("unpack log %s: %w", log.TxHash.Hex(), err)
	}

	content, ok := data[0].(string)
	if !ok {
		return store.Event{}, fmt.Errorf("log %s: content is %T, want string", log.TxHash.Hex(), data[0])
	}
	timestamp, ok := data[1].(*big.Int)
	if !ok {
		return store.Event{}, fmt.Errorf("log %s: timestamp is %T, want *big.Int", log.TxHash.Hex(), data[1])
	}

	return store.Event{
		BlockNumber: log.BlockNumber,
		TxHash:      log.TxHash.Hex(),
		LogIndex:    log.Index,
		Sender:      common.BytesToAddress(log.Topics[1].Bytes()).Hex(),
		Content:     content,
		Timestamp:   timestamp.Uint64(),
	}, nil
}
