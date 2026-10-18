// Package config loads the backend settings from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every backend setting.
type Config struct {
	RPCURL          string // RPC_URL — node endpoint (anvil or a Sepolia RPC)
	ChainID         int64  // CHAIN_ID — 31337 (anvil) or 11155111 (Sepolia)
	ContractAddress string // CONTRACT_ADDRESS — deployed Guestbook address
	PrivateKey      string // PRIVATE_KEY — relayer key, optional, testnet only
	DBPath          string // DB_PATH — SQLite file path
	APIPort         int    // API_PORT — HTTP port
}

// Load reads .env (if present) plus the process environment, applies defaults,
// and validates required fields.
func Load() (*Config, error) {
	// .env is convenient locally but absent in Docker and CI, so a missing file is fine.
	_ = godotenv.Load()

	chainID, err := lookupInt64("CHAIN_ID", 31337)
	if err != nil {
		return nil, err
	}
	apiPort, err := lookupInt("API_PORT", 8080)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		RPCURL:          strings.TrimSpace(os.Getenv("RPC_URL")),
		ChainID:         chainID,
		ContractAddress: strings.TrimSpace(os.Getenv("CONTRACT_ADDRESS")),
		PrivateKey:      strings.TrimSpace(os.Getenv("PRIVATE_KEY")),
		DBPath:          lookupDefault("DB_PATH", "./data.db"),
		APIPort:         apiPort,
	}

	if cfg.RPCURL == "" {
		return nil, errors.New("RPC_URL is required")
	}
	if cfg.ContractAddress == "" {
		return nil, errors.New("CONTRACT_ADDRESS is required")
	}

	return cfg, nil
}

// lookupDefault returns the trimmed value of key, or fallback when it is unset or empty.
func lookupDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// lookupInt parses key as an int, or returns fallback when it is unset or empty.
func lookupInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return value, nil
}

// lookupInt64 parses key as an int64, or returns fallback when it is unset or empty.
func lookupInt64(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return value, nil
}
