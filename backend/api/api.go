// Package api exposes the webhook registration REST API.
//
// Endpoints:
//
//	POST /register-webhook  – register a URL for an event type
//	GET  /webhooks          – list all registered webhooks
//	GET  /health            – liveness probe
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Raad05/anchor-hook/registry"
)

// Server wraps the registry and the HTTP mux.
type Server struct {
	reg  *registry.Registry
	mux  *http.ServeMux
	addr string
}

// New creates a Server listening on addr (e.g. ":8080").
func New(reg *registry.Registry, addr string) *Server {
	s := &Server{reg: reg, mux: http.NewServeMux(), addr: addr}
	s.mux.HandleFunc("/register-webhook", s.handleRegister)
	s.mux.HandleFunc("/webhooks", s.handleList)
	s.mux.HandleFunc("/health", s.handleHealth)
	return s
}

// Start begins serving HTTP requests. It blocks until the server exits.
func (s *Server) Start() error {
	log.Printf("[api] listening on %s", s.addr)
	return http.ListenAndServe(s.addr, s.withCORS(s.mux))
}

// Handler returns the underlying http.Handler (useful for testing).
func (s *Server) Handler() http.Handler {
	return s.withCORS(s.mux)
}

// ── withCORS middleware ───────────────────────────────────────────────────────

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Handlers ─────────────────────────────────────────────────────────────────

type registerRequest struct {
	WebhookURL string `json:"webhook_url"`
	EventType  string `json:"event_type"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{"method not allowed"})
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid JSON body"})
		return
	}

	req.WebhookURL = strings.TrimSpace(req.WebhookURL)
	req.EventType = strings.TrimSpace(req.EventType)

	if req.WebhookURL == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{"webhook_url is required"})
		return
	}
	if req.EventType == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{"event_type is required"})
		return
	}

	s.reg.Add(req.EventType, req.WebhookURL)
	log.Printf("[api] registered webhook url=%s event_type=%s", req.WebhookURL, req.EventType)

	writeJSON(w, http.StatusCreated, map[string]string{
		"status":      "registered",
		"webhook_url": req.WebhookURL,
		"event_type":  req.EventType,
	})
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{"method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, s.reg.All())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
