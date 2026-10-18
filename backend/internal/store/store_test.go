package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

// newStore returns a store backed by a throwaway database file.
func newStore(t *testing.T) *Store {
	t.Helper()

	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	return st
}

func TestListMessagesEmpty(t *testing.T) {
	st := newStore(t)

	messages, err := st.ListMessages(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("len = %d, want 0", len(messages))
	}
}

func TestInsertEventStoresMessage(t *testing.T) {
	st := newStore(t)
	event := Event{BlockNumber: 5, TxHash: "0xabc", LogIndex: 0, Sender: "0xalice", Content: "hello", Timestamp: 1000}

	if err := st.InsertEvent(context.Background(), event); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	messages, err := st.ListMessages(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len = %d, want 1", len(messages))
	}

	want := Message{Sender: "0xalice", Content: "hello", BlockNumber: 5, Timestamp: 1000}
	if messages[0] != want {
		t.Fatalf("message = %+v, want %+v", messages[0], want)
	}
}

func TestInsertEventIsIdempotentOnReplay(t *testing.T) {
	st := newStore(t)
	event := Event{BlockNumber: 5, TxHash: "0xabc", LogIndex: 0, Sender: "0xalice", Content: "hello", Timestamp: 1000}

	for i := 0; i < 3; i++ {
		if err := st.InsertEvent(context.Background(), event); err != nil {
			t.Fatalf("InsertEvent #%d: %v", i, err)
		}
	}

	messages, err := st.ListMessages(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len = %d after replay, want 1", len(messages))
	}
}

func TestInsertEventKeepsDistinctLogIndices(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	events := []Event{
		{BlockNumber: 1, TxHash: "0xsame", LogIndex: 0, Sender: "0xa", Content: "a", Timestamp: 1},
		{BlockNumber: 1, TxHash: "0xsame", LogIndex: 1, Sender: "0xa", Content: "b", Timestamp: 1},
	}
	for _, event := range events {
		if err := st.InsertEvent(ctx, event); err != nil {
			t.Fatalf("InsertEvent: %v", err)
		}
	}

	messages, err := st.ListMessages(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len = %d, want 2", len(messages))
	}
}

func TestListMessagesPaginates(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	for i, content := range []string{"one", "two", "three", "four"} {
		event := Event{
			BlockNumber: uint64(i + 1),
			TxHash:      fmt.Sprintf("0xtx%d", i),
			LogIndex:    0,
			Sender:      "0xbob",
			Content:     content,
			Timestamp:   uint64(1000 + i),
		}
		if err := st.InsertEvent(ctx, event); err != nil {
			t.Fatalf("InsertEvent(%q): %v", content, err)
		}
	}

	page, err := st.ListMessages(ctx, 2, 1)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("len = %d, want 2", len(page))
	}
	if page[0].Content != "two" || page[1].Content != "three" {
		t.Fatalf("page = %q, %q; want two, three", page[0].Content, page[1].Content)
	}

	if page, err := st.ListMessages(ctx, 2, 10); err != nil || len(page) != 0 {
		t.Fatalf("offset past end: len = %d, err = %v; want 0, nil", len(page), err)
	}
}

func TestLastBlockRoundTrip(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	last, err := st.GetLastBlock(ctx)
	if err != nil {
		t.Fatalf("GetLastBlock: %v", err)
	}
	if last != 0 {
		t.Fatalf("initial last block = %d, want 0", last)
	}

	if err := st.SetLastBlock(ctx, 42); err != nil {
		t.Fatalf("SetLastBlock: %v", err)
	}
	if last, err := st.GetLastBlock(ctx); err != nil || last != 42 {
		t.Fatalf("last block = %d, err = %v; want 42, nil", last, err)
	}

	// The cursor is a single row, so a later write overwrites it.
	if err := st.SetLastBlock(ctx, 7); err != nil {
		t.Fatalf("SetLastBlock overwrite: %v", err)
	}
	if last, err := st.GetLastBlock(ctx); err != nil || last != 7 {
		t.Fatalf("last block = %d, err = %v; want 7, nil", last, err)
	}
}
