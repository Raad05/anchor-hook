// Package listener provides a Solana WebSocket client that subscribes to
// program logs via the logsSubscribe RPC method and streams raw log bundles
// onto a channel for downstream processing.
package listener

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// RawLog carries the transaction signature and the full slice of log strings
// returned by a Solana logsSubscribe notification.
type RawLog struct {
	Signature string
	Logs      []string
}

// ─── JSON shapes for logsSubscribe ───────────────────────────────────────────

// subscribeRequest is the JSON-RPC message sent to the validator.
type subscribeRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// logsNotification is the shape of the incoming subscription update.
type logsNotification struct {
	Params struct {
		Result struct {
			Value struct {
				Signature string   `json:"signature"`
				Logs      []string `json:"logs"`
			} `json:"value"`
		} `json:"result"`
	} `json:"params"`
}

// subscriptionResponse is the initial ack from the validator.
type subscriptionResponse struct {
	Result int `json:"result"`
}

// Listener wraps a gorilla WebSocket connection and manages the RPC
// subscription lifecycle.
type Listener struct {
	conn  *websocket.Conn
	subID int
}

// Connect dials the Solana WebSocket endpoint and returns a Listener or an
// error.
func Connect(wsURL string) (*Listener, error) {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", wsURL, err)
	}
	log.Printf("[listener] connected to %s", wsURL)
	return &Listener{conn: conn}, nil
}

// Subscribe sends a logsSubscribe JSON-RPC request filtered to programID and
// records the subscription ID returned by the validator.
func (l *Listener) Subscribe(programID string) error {
	req := subscribeRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "logsSubscribe",
		Params: []interface{}{
			map[string]interface{}{"mentions": []string{programID}},
			map[string]string{"commitment": "confirmed"},
		},
	}

	if err := l.conn.WriteJSON(req); err != nil {
		return fmt.Errorf("send logsSubscribe: %w", err)
	}

	// Read the acknowledgement which carries the numeric subscription ID.
	_, msg, err := l.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read subscription ack: %w", err)
	}

	var ack subscriptionResponse
	if err := json.Unmarshal(msg, &ack); err != nil {
		return fmt.Errorf("parse subscription ack: %w", err)
	}
	l.subID = ack.Result
	log.Printf("[listener] subscribed: id=%d program=%s", l.subID, programID)
	return nil
}

// Listen reads from the WebSocket connection indefinitely, deserialising each
// notification and forwarding RawLog values to ch.  It reconnects automatically
// on transient read errors (up to maxRetries attempts with exponential backoff).
// The caller should close ch after cancelling the context / shutting down.
func (l *Listener) Listen(ch chan<- RawLog) {
	const maxRetries = 5
	backoff := 500 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[listener] reconnect attempt %d/%d (backoff=%s)", attempt, maxRetries, backoff)
			time.Sleep(backoff)
			backoff *= 2
		}

		for {
			_, msg, err := l.conn.ReadMessage()
			if err != nil {
				log.Printf("[listener] read error: %v", err)
				break // inner loop → trigger reconnect
			}

			var notif logsNotification
			if err := json.Unmarshal(msg, &notif); err != nil {
				// Could be a ping/pong or unrelated message; skip silently.
				continue
			}

			value := notif.Params.Result.Value
			if value.Signature == "" {
				// Not a log notification (e.g. subscription ack echoed).
				continue
			}

			ch <- RawLog{
				Signature: value.Signature,
				Logs:      value.Logs,
			}
		}
	}

	log.Printf("[listener] giving up after %d reconnect attempts", maxRetries)
	close(ch)
}

// Close tears down the underlying WebSocket connection.
func (l *Listener) Close() {
	_ = l.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	l.conn.Close()
}
