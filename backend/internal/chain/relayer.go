package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/yourname/web3-chainlog/backend/internal/config"
)

// receiptPollInterval is how often the relayer checks whether its transaction was mined.
const receiptPollInterval = time.Second

// Relayer signs and sends Guestbook.post transactions with the backend's own key.
// It exists to exercise the write path — nonce, gas, signing and receipt status —
// and is deliberately not wired to the HTTP API.
type Relayer struct {
	*contract
	key    *ecdsa.PrivateKey
	from   common.Address
	signer types.Signer
	logger *slog.Logger
}

// NewRelayer connects to the node and loads the key that pays for the transactions.
// The key is meant to be a throwaway testnet key.
func NewRelayer(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Relayer, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.PrivateKey == "" {
		return nil, errors.New("PRIVATE_KEY is required to relay transactions")
	}

	key, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse PRIVATE_KEY: %w", err)
	}

	c, err := dialContract(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if _, ok := c.abi.Methods["post"]; !ok {
		c.Close()
		return nil, errors.New("abi has no post method")
	}

	return &Relayer{
		contract: c,
		key:      key,
		from:     crypto.PubkeyToAddress(key.PublicKey),
		signer:   types.LatestSignerForChainID(c.chainID),
		logger:   logger,
	}, nil
}

// Post sends post(content) from the relayer key and waits until it is mined,
// returning the receipt so the caller can inspect the block and status.
//
// The nonce is read per call and no nonce manager is kept, so one relayer must
// post sequentially; concurrent calls would race on the same nonce.
func (r *Relayer) Post(ctx context.Context, content string) (*types.Receipt, error) {
	calldata, err := r.abi.Pack("post", content)
	if err != nil {
		return nil, fmt.Errorf("pack post(%q): %w", content, err)
	}

	nonce, err := r.client.PendingNonceAt(ctx, r.from)
	if err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}

	gasLimit, err := r.client.EstimateGas(ctx, ethereum.CallMsg{
		From: r.from,
		To:   &r.address,
		Data: calldata,
	})
	if err != nil {
		return nil, fmt.Errorf("estimate gas: %w", err)
	}

	gasFeeCap, gasTipCap, err := r.fees(ctx)
	if err != nil {
		return nil, err
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   r.chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &r.address,
		Value:     big.NewInt(0),
		Data:      calldata,
	})

	signed, err := types.SignTx(tx, r.signer, r.key)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	if err := r.client.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("send tx: %w", err)
	}
	r.logger.Info("relayed post", "tx", signed.Hash().Hex(), "from", r.from.Hex(), "nonce", nonce, "gas", gasLimit)

	receipt, err := r.waitMined(ctx, signed.Hash())
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("tx %s reverted in block %d", signed.Hash().Hex(), receipt.BlockNumber)
	}

	return receipt, nil
}

// fees returns the EIP-1559 caps: the node's suggested tip, and room for the base
// fee to double before the transaction becomes unminable.
func (r *Relayer) fees(ctx context.Context) (gasFeeCap, gasTipCap *big.Int, err error) {
	tip, err := r.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("suggest gas tip cap: %w", err)
	}

	head, err := r.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("read head: %w", err)
	}
	if head.BaseFee == nil {
		return nil, nil, errors.New("node reports no base fee; dynamic fee transactions are unavailable")
	}

	return new(big.Int).Add(new(big.Int).Lsh(head.BaseFee, 1), tip), tip, nil
}

// waitMined polls for the receipt until the transaction is mined or ctx ends.
func (r *Relayer) waitMined(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(receiptPollInterval)
	defer ticker.Stop()

	for {
		receipt, err := r.client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("read receipt %s: %w", hash.Hex(), err)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for receipt %s: %w", hash.Hex(), ctx.Err())
		case <-ticker.C:
		}
	}
}
