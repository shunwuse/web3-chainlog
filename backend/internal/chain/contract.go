package chain

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/yourname/web3-chainlog/backend/internal/config"
)

//go:embed guestbook_abi.json
var guestbookABI string

// contract bundles the node connection with the guestbook ABI. The indexer reads
// MessagePosted events through it and the relayer sends post transactions, so the
// chain id check and the ABI parse happen exactly once for both.
type contract struct {
	client  *ethclient.Client
	address common.Address
	chainID *big.Int
	abi     abi.ABI
}

// dialContract connects to the configured node, refuses to continue unless it is
// the expected chain, and parses the ABI.
func dialContract(ctx context.Context, cfg *config.Config) (*contract, error) {
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

	return &contract{
		client:  client,
		address: common.HexToAddress(cfg.ContractAddress),
		chainID: chainID,
		abi:     parsed,
	}, nil
}

// Close releases the RPC connection.
func (c *contract) Close() {
	c.client.Close()
}
