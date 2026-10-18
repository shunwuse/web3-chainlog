package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/yourname/web3-chainlog/backend/internal/store"
)

// newHandler returns a router backed by a throwaway store.
func newHandler(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	return NewRouter(st, nil), st
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealth(t *testing.T) {
	handler, _ := newHandler(t)

	rec := get(t, handler, "/api/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != `{"ok":true}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestSyncReportsCursor(t *testing.T) {
	handler, st := newHandler(t)

	if rec := get(t, handler, "/api/sync"); rec.Body.String() != `{"lastBlock":0}`+"\n" {
		t.Fatalf("empty body = %q", rec.Body.String())
	}

	if err := st.SetLastBlock(context.Background(), 12); err != nil {
		t.Fatalf("SetLastBlock: %v", err)
	}
	if rec := get(t, handler, "/api/sync"); rec.Body.String() != `{"lastBlock":12}`+"\n" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestMessagesEmptyIsJSONArray(t *testing.T) {
	handler, _ := newHandler(t)

	rec := get(t, handler, "/api/messages")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != `{"messages":[]}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestMessagesReturnsStoredEntries(t *testing.T) {
	handler, st := newHandler(t)

	event := store.Event{BlockNumber: 7, TxHash: "0xabc", LogIndex: 0, Sender: "0xalice", Content: "hi", Timestamp: 1234}
	if err := st.InsertEvent(context.Background(), event); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	rec := get(t, handler, "/api/messages")
	want := `{"messages":[{"sender":"0xalice","content":"hi","blockNumber":7,"timestamp":1234}]}` + "\n"
	if body := rec.Body.String(); body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestMessagesRejectsBadQuery(t *testing.T) {
	handler, _ := newHandler(t)

	tests := []struct {
		path string
		want int
	}{
		{"/api/messages", http.StatusOK},
		{"/api/messages?limit=100&offset=0", http.StatusOK},
		{"/api/messages?limit=abc", http.StatusBadRequest},
		{"/api/messages?limit=0", http.StatusBadRequest},
		{"/api/messages?limit=101", http.StatusBadRequest},
		{"/api/messages?offset=-1", http.StatusBadRequest},
		{"/api/messages?offset=abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		if rec := get(t, handler, tt.path); rec.Code != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.path, rec.Code, tt.want)
		}
	}
}

func TestCORSHeader(t *testing.T) {
	handler, _ := newHandler(t)

	rec := get(t, handler, "/api/health")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}
