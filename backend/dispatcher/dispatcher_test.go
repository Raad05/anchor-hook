package dispatcher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Raad05/anchor-hook/decoder"
	"github.com/Raad05/anchor-hook/registry"
)

// makeUA returns a minimal UserAction for testing.
func makeUA(actionType string, amount uint64) *decoder.UserAction {
	ua := &decoder.UserAction{ActionType: actionType, Amount: amount}
	ua.User[0] = 0x01
	return ua
}

// ── Happy-path delivery ───────────────────────────────────────────────────────

func TestDispatch_DeliveryHappyPath(t *testing.T) {
	received := make(chan webhookPayload, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p webhookPayload
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &p)
		received <- p
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reg := registry.New()
	reg.Add("transfer", srv.URL)

	d := New(reg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	ua := makeUA("transfer", 500_000_000)
	d.Dispatch(ua)

	select {
	case p := <-received:
		if p.EventType != "transfer" {
			t.Errorf("event_type: got %q want %q", p.EventType, "transfer")
		}
		if p.Amount != 500_000_000 {
			t.Errorf("amount: got %d want %d", p.Amount, 500_000_000)
		}
		if p.User == "" {
			t.Error("user should not be empty")
		}
		if p.Timestamp == "" {
			t.Error("timestamp should not be empty")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}
}

// ── Wildcard delivery ─────────────────────────────────────────────────────────

func TestDispatch_WildcardDelivery(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reg := registry.New()
	reg.Add("*", srv.URL) // catch-all

	d := New(reg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	d.Dispatch(makeUA("stake", 1))
	d.Dispatch(makeUA("vote", 2))

	time.Sleep(500 * time.Millisecond)
	if got := callCount.Load(); got != 2 {
		t.Errorf("expected 2 wildcard deliveries, got %d", got)
	}
}

// ── No matching webhooks ──────────────────────────────────────────────────────

func TestDispatch_NoWebhooks(t *testing.T) {
	reg := registry.New()
	d := New(reg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	// Should not panic or block.
	d.Dispatch(makeUA("unknown", 0))
}

// ── Retry mechanism ───────────────────────────────────────────────────────────

func TestDispatch_RetryOnFailure(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError) // fail first 2 attempts
			return
		}
		w.WriteHeader(http.StatusOK) // succeed on attempt 3
	}))
	defer srv.Close()

	reg := registry.New()
	reg.Add("transfer", srv.URL)

	d := New(reg)
	// Override backoff to be nearly instant for the test.
	d.maxRetries = 3

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	d.Dispatch(makeUA("transfer", 1))

	// Retry backoff: 500ms + 1s = 1.5s, allow 5s total.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out; call count = %d", callCount.Load())
		case <-time.After(100 * time.Millisecond):
			if callCount.Load() >= 3 {
				return // success — retried until it worked
			}
		}
	}
}
