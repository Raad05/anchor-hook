package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Raad05/anchor-hook/registry"
)

func newTestServer() *Server {
	return New(registry.New(), ":0")
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// ── POST /register-webhook ────────────────────────────────────────────────────

func TestRegisterWebhook_HappyPath(t *testing.T) {
	s := newTestServer()
	rr := doRequest(t, s.Handler(), http.MethodPost, "/register-webhook", map[string]string{
		"webhook_url": "https://example.com/hook",
		"event_type":  "transfer",
	})
	if rr.Code != http.StatusCreated {
		t.Errorf("status: got %d want %d — body: %s", rr.Code, http.StatusCreated, rr.Body.String())
	}
}

func TestRegisterWebhook_MissingURL(t *testing.T) {
	s := newTestServer()
	rr := doRequest(t, s.Handler(), http.MethodPost, "/register-webhook", map[string]string{
		"event_type": "transfer",
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestRegisterWebhook_MissingEventType(t *testing.T) {
	s := newTestServer()
	rr := doRequest(t, s.Handler(), http.MethodPost, "/register-webhook", map[string]string{
		"webhook_url": "https://example.com/hook",
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestRegisterWebhook_InvalidJSON(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/register-webhook", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestRegisterWebhook_WrongMethod(t *testing.T) {
	s := newTestServer()
	rr := doRequest(t, s.Handler(), http.MethodGet, "/register-webhook", nil)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

// ── GET /webhooks ─────────────────────────────────────────────────────────────

func TestListWebhooks_Empty(t *testing.T) {
	s := newTestServer()
	rr := doRequest(t, s.Handler(), http.MethodGet, "/webhooks", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestListWebhooks_AfterRegister(t *testing.T) {
	s := newTestServer()

	doRequest(t, s.Handler(), http.MethodPost, "/register-webhook", map[string]string{
		"webhook_url": "https://example.com/a",
		"event_type":  "stake",
	})

	rr := doRequest(t, s.Handler(), http.MethodGet, "/webhooks", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusOK)
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := result["stake"]; !ok {
		t.Errorf("expected 'stake' key in response, got: %v", result)
	}
}

// ── GET /health ───────────────────────────────────────────────────────────────

func TestHealth(t *testing.T) {
	s := newTestServer()
	rr := doRequest(t, s.Handler(), http.MethodGet, "/health", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d want %d", rr.Code, http.StatusOK)
	}
}

// ── CORS preflight ────────────────────────────────────────────────────────────

func TestCORSPreflight(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodOptions, "/register-webhook", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("preflight status: got %d want %d", rr.Code, http.StatusNoContent)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header missing")
	}
}
