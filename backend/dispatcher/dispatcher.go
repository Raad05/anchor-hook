// Package dispatcher fans out decoded UserAction events to all registered
// webhook URLs concurrently, with exponential-backoff retry on failure.
package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Raad05/anchor-hook/decoder"
	"github.com/Raad05/anchor-hook/registry"
)

const (
	defaultWorkers    = 4
	defaultMaxRetries = 3
	initialBackoff    = 500 * time.Millisecond
)

// job is a single "deliver this payload to this URL" unit of work.
type job struct {
	url     string
	payload webhookPayload
}

// webhookPayload is the JSON body sent to the target webhook URL.
type webhookPayload struct {
	EventType string `json:"event_type"`
	User      string `json:"user"`
	Amount    uint64 `json:"amount"`
	Timestamp string `json:"timestamp"`
}

// Dispatcher routes decoded events to registered webhook URLs via a
// fixed-size pool of goroutine workers.
type Dispatcher struct {
	reg        *registry.Registry
	jobChan    chan job
	httpClient *http.Client
	workers    int
	maxRetries int
}

// New creates a Dispatcher backed by reg.  Call Start before Dispatch.
func New(reg *registry.Registry) *Dispatcher {
	return &Dispatcher{
		reg:        reg,
		jobChan:    make(chan job, 256),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		workers:    defaultWorkers,
		maxRetries: defaultMaxRetries,
	}
}

// Start spawns the worker goroutines.  They run until ctx is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.workers; i++ {
		go d.worker(ctx, i)
	}
	log.Printf("[dispatcher] started %d workers", d.workers)
}

// Dispatch looks up all webhooks registered for ua.ActionType (and the "*"
// wildcard) and enqueues a delivery job for each registered URL.
func (d *Dispatcher) Dispatch(ua *decoder.UserAction) {
	targets := d.reg.Get(ua.ActionType)
	if len(targets) == 0 {
		return
	}

	payload := webhookPayload{
		EventType: ua.ActionType,
		User:      ua.UserBase58(),
		Amount:    ua.Amount,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	for _, t := range targets {
		select {
		case d.jobChan <- job{url: t.URL, payload: payload}:
		default:
			log.Printf("[dispatcher] job queue full, dropping delivery to %s", t.URL)
		}
	}
}

// worker reads jobs from jobChan and calls postWithRetry for each one.
func (d *Dispatcher) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[dispatcher] worker %d shutting down", id)
			return
		case j, ok := <-d.jobChan:
			if !ok {
				return
			}
			if err := d.postWithRetry(j.url, j.payload); err != nil {
				log.Printf("[dispatcher] worker %d: failed to deliver to %s: %v", id, j.url, err)
			}
		}
	}
}

// postWithRetry attempts to POST payload to url up to maxRetries times with
// exponential backoff (500 ms → 1 s → 2 s).
func (d *Dispatcher) postWithRetry(url string, payload webhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	backoff := initialBackoff
	for attempt := 1; attempt <= d.maxRetries; attempt++ {
		err = d.doPost(url, body)
		if err == nil {
			log.Printf("[dispatcher] delivered to %s (attempt %d)", url, attempt)
			return nil
		}
		log.Printf("[dispatcher] attempt %d/%d to %s failed: %v", attempt, d.maxRetries, url, err)
		if attempt < d.maxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return fmt.Errorf("all %d attempts failed for %s: %w", d.maxRetries, url, err)
}

// doPost performs a single HTTP POST.
func (d *Dispatcher) doPost(url string, body []byte) error {
	resp, err := d.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx response: %s", resp.Status)
	}
	return nil
}
