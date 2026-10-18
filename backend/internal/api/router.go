// Package api exposes the indexed guestbook over HTTP.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yourname/web3-chainlog/backend/internal/store"
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

// server holds the handler dependencies.
type server struct {
	store  *store.Store
	logger *slog.Logger
}

// NewRouter returns the HTTP handler for the guestbook API.
func NewRouter(st *store.Store, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	s := &server{store: st, logger: logger}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(s.withCORS)

	r.Get("/api/health", s.health)
	r.Get("/api/messages", s.listMessages)
	r.Get("/api/sync", s.sync)

	return r
}

// withCORS lets the host-run frontend call the API from another origin.
func (s *server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) listMessages(w http.ResponseWriter, r *http.Request) {
	limit, err := intParam(r, "limit", defaultLimit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if limit < 1 || limit > maxLimit {
		http.Error(w, fmt.Sprintf("limit must be between 1 and %d", maxLimit), http.StatusBadRequest)
		return
	}

	offset, err := intParam(r, "offset", 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if offset < 0 {
		http.Error(w, "offset must not be negative", http.StatusBadRequest)
		return
	}

	messages, err := s.store.ListMessages(r.Context(), limit, offset)
	if err != nil {
		s.logger.Error("list messages", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// A non-nil slice keeps the empty case as `[]` instead of `null`.
	response := messagesResponse{Messages: make([]messageResponse, 0, len(messages))}
	for _, message := range messages {
		response.Messages = append(response.Messages, messageResponse{
			Sender:      message.Sender,
			Content:     message.Content,
			BlockNumber: message.BlockNumber,
			Timestamp:   message.Timestamp,
		})
	}
	writeJSON(w, response)
}

func (s *server) sync(w http.ResponseWriter, r *http.Request) {
	lastBlock, err := s.store.GetLastBlock(r.Context())
	if err != nil {
		s.logger.Error("get last block", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, syncResponse{LastBlock: lastBlock})
}

type messageResponse struct {
	Sender      string `json:"sender"`
	Content     string `json:"content"`
	BlockNumber uint64 `json:"blockNumber"`
	Timestamp   uint64 `json:"timestamp"`
}

type messagesResponse struct {
	Messages []messageResponse `json:"messages"`
}

type syncResponse struct {
	LastBlock uint64 `json:"lastBlock"`
}

// intParam reads an integer query parameter, falling back to fallback when absent.
func intParam(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
