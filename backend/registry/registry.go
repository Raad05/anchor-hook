// Package registry provides a thread-safe, in-memory store for webhook
// registrations keyed by event type.
//
// A wildcard event type "*" can be used to receive all event types.
package registry

import "sync"

// WebhookEntry holds a registered webhook URL for a given event type.
type WebhookEntry struct {
	URL       string `json:"url"`
	EventType string `json:"event_type"`
}

// Registry is a mutex-protected map of event_type → []WebhookEntry.
type Registry struct {
	mu       sync.RWMutex
	webhooks map[string][]WebhookEntry
}

// New returns an initialised, empty Registry.
func New() *Registry {
	return &Registry{
		webhooks: make(map[string][]WebhookEntry),
	}
}

// Add registers a webhook URL for the given event type.
// Duplicate entries (same URL + event_type) are silently ignored.
func (r *Registry) Add(eventType, webhookURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.webhooks[eventType] {
		if entry.URL == webhookURL {
			return // already registered
		}
	}
	r.webhooks[eventType] = append(r.webhooks[eventType], WebhookEntry{
		URL:       webhookURL,
		EventType: eventType,
	})
}

// Get returns all webhooks registered for the given event type plus any
// registered under the wildcard "*".
func (r *Registry) Get(eventType string) []WebhookEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	targets := make([]WebhookEntry, 0)
	targets = append(targets, r.webhooks[eventType]...)

	// Add wildcard entries if the event type is not already "*".
	if eventType != "*" {
		targets = append(targets, r.webhooks["*"]...)
	}
	return targets
}

// All returns a snapshot of the full registry (safe to serialise to JSON).
func (r *Registry) All() map[string][]WebhookEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := make(map[string][]WebhookEntry, len(r.webhooks))
	for k, v := range r.webhooks {
		cp := make([]WebhookEntry, len(v))
		copy(cp, v)
		snapshot[k] = cp
	}
	return snapshot
}
