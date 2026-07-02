package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestAPI() *API {
	return New(slog.New(slog.NewTextHandler(io.Discard, nil)), "test")
}

func TestHealthz(t *testing.T) {
	api := newTestAPI()
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReadyzTogglesWithState(t *testing.T) {
	api := newTestAPI()
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	// Not ready by default -> 503.
	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before ready, got %d", resp.StatusCode)
	}

	// After MarkReady -> 200.
	api.MarkReady()
	resp, err = http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after ready, got %d", resp.StatusCode)
	}
}

func TestHelloUsesQueryParam(t *testing.T) {
	api := newTestAPI()
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/hello?name=srikanth")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body["message"] != "hello, srikanth" {
		t.Fatalf("unexpected message: %q", body["message"])
	}
}

func TestWorkReturnsLatency(t *testing.T) {
	api := newTestAPI()
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/work")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if _, ok := body["latency_ms"]; !ok {
		t.Fatalf("expected latency_ms in response, got %v", body)
	}
}

func TestRootReturnsMetadata(t *testing.T) {
	api := newTestAPI()
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body["service"] != "devops-showcase" {
		t.Fatalf("expected service name in metadata, got %v", body["service"])
	}
	if _, ok := body["build"]; !ok {
		t.Fatalf("expected build info in metadata, got %v", body)
	}
}

// The /api/error endpoint fails ~30% of the time on purpose. Whatever it
// returns, it must be a valid response with a sensible status code and a
// well-formed JSON body — never a panic or malformed output.
func TestErrorEndpointIsAlwaysWellFormed(t *testing.T) {
	api := newTestAPI()
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	for i := 0; i < 50; i++ {
		resp, err := http.Get(srv.URL + "/api/error")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			resp.Body.Close()
			t.Fatalf("unexpected status code %d", resp.StatusCode)
		}
		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close()
			t.Fatalf("response was not valid JSON: %v", err)
		}
		resp.Body.Close()
	}
}
