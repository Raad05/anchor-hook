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
	"strings"
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

// webhookPayload is the generic JSON body sent to non-Discord targets.
type webhookPayload struct {
	EventType string `json:"event_type"`
	User      string `json:"user"`
	Amount    uint64 `json:"amount"`
	Timestamp string `json:"timestamp"`
}

// ── Discord embed types ───────────────────────────────────────────────────────

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"` // decimal RGB
	Fields      []discordField `json:"fields"`
	Footer      struct {
		Text string `json:"text"`
	} `json:"footer"`
	Timestamp string `json:"timestamp"` // ISO-8601
}

type discordPayload struct {
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url"`
	Embeds    []discordEmbed `json:"embeds"`
}

// actionColor returns a Discord embed colour for a known action type.
func actionColor(action string) int {
	switch strings.ToLower(action) {
	case "transfer":
		return 0x5865F2 // blurple
	case "vote":
		return 0x57F287 // green
	case "stake":
		return 0xFEE75C // yellow
	case "unstake":
		return 0xED4245 // red
	case "withdraw":
		return 0xEB459E // fuchsia
	default:
		return 0x99AAB5 // grey
	}
}

// buildDiscordBody formats a Discord embed payload from ua.
func buildDiscordBody(p webhookPayload) ([]byte, error) {
	emoji := map[string]string{
		"transfer": "💸",
		"vote":     "🗳️",
		"stake":    "🔒",
		"unstake":  "🔓",
		"withdraw": "📤",
	}
	icon := emoji[strings.ToLower(p.EventType)]
	if icon == "" {
		icon = "⚡"
	}

	dp := discordPayload{
		Username:  "Anchor Hook",
		AvatarURL: "https://raw.githubusercontent.com/solana-labs/token-list/main/assets/mainnet/So11111111111111111111111111111111111111112/logo.png",
		Embeds: []discordEmbed{
			{
				Title:       fmt.Sprintf("%s On-Chain Event Detected", icon),
				Description: fmt.Sprintf("A **%s** action was emitted from your Anchor program.", p.EventType),
				Color:       actionColor(p.EventType),
				Fields: []discordField{
					{Name: "Action", Value: fmt.Sprintf("`%s`", p.EventType), Inline: true},
					{Name: "Amount", Value: fmt.Sprintf("`%d`", p.Amount), Inline: true},
					{Name: "Wallet", Value: fmt.Sprintf("`%s`", p.User), Inline: false},
				},
				Timestamp: p.Timestamp,
			},
		},
	}
	dp.Embeds[0].Footer.Text = "Anchor Hook • Colosseum"
	return json.Marshal(dp)
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
	var body []byte
	var err error

	switch {
	case strings.Contains(url, "discord.com/api/webhooks"):
		body, err = buildDiscordBody(payload)
	case isTeamsURL(url):
		body, err = buildTeamsBody(payload)
	default:
		body, err = json.Marshal(payload)
	}
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

// ── Microsoft Teams helpers ──────────────────────────────────────────────────

// isTeamsURL reports whether url is a Microsoft Teams / Power Platform
// incoming webhook endpoint.
func isTeamsURL(url string) bool {
	return strings.Contains(url, "webhook.office.com") ||
		strings.Contains(url, "office.com") ||
		strings.Contains(url, "powerplatform.com")
}

// teamsMessageCard is the legacy Teams MessageCard payload (universally
// supported by all Teams incoming webhook connectors).
type teamsMessageCard struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor string         `json:"themeColor"`
	Summary    string         `json:"summary"`
	Sections   []teamsSection `json:"sections"`
}

type teamsSection struct {
	ActivityTitle    string       `json:"activityTitle"`
	ActivitySubtitle string       `json:"activitySubtitle"`
	Facts            []teamsFact  `json:"facts"`
	Markdown         bool         `json:"markdown"`
}

type teamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// teamsThemeColor returns a hex colour string for a known action type.
func teamsThemeColor(action string) string {
	switch strings.ToLower(action) {
	case "transfer":
		return "5865F2"
	case "vote":
		return "57F287"
	case "stake":
		return "FEE75C"
	case "unstake":
		return "ED4245"
	case "withdraw":
		return "EB459E"
	default:
		return "0076D7"
	}
}

// buildTeamsBody formats a Teams MessageCard payload.
func buildTeamsBody(p webhookPayload) ([]byte, error) {
	emoji := map[string]string{
		"transfer": "💸",
		"vote":     "🗳️",
		"stake":    "🔒",
		"unstake":  "🔓",
		"withdraw": "📤",
	}
	icon := emoji[strings.ToLower(p.EventType)]
	if icon == "" {
		icon = "⚡"
	}

	card := teamsMessageCard{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: teamsThemeColor(p.EventType),
		Summary:    fmt.Sprintf("%s On-Chain Event: %s", icon, p.EventType),
		Sections: []teamsSection{
			{
				ActivityTitle:    fmt.Sprintf("%s On-Chain Event Detected", icon),
				ActivitySubtitle: "Anchor Hook • Colosseum",
				Facts: []teamsFact{
					{Name: "Action", Value: p.EventType},
					{Name: "Amount", Value: fmt.Sprintf("%d", p.Amount)},
					{Name: "Wallet", Value: p.User},
					{Name: "Timestamp", Value: p.Timestamp},
				},
				Markdown: true,
			},
		},
	}
	return json.Marshal(card)
}
