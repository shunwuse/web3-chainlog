// Command backend runs the guestbook indexer and its HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourname/web3-chainlog/backend/internal/api"
	"github.com/yourname/web3-chainlog/backend/internal/chain"
	"github.com/yourname/web3-chainlog/backend/internal/config"
	"github.com/yourname/web3-chainlog/backend/internal/store"
)

// shutdownTimeout bounds how long in-flight requests may take to finish.
const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("backend exited", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	// SIGINT/SIGTERM cancels ctx, which stops the indexer and the server.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	indexer, err := chain.NewIndexer(ctx, cfg, st, logger)
	if err != nil {
		return err
	}
	defer indexer.Close()

	indexerDone := make(chan struct{})
	go func() {
		defer close(indexerDone)
		indexer.Run(ctx)
	}()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.APIPort),
		Handler: api.NewRouter(st, logger),
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	<-indexerDone
	return nil
}
