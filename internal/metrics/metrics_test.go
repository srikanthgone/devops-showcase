package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareRecordsAndExposes(t *testing.T) {
	reg := New("test")

	handler := reg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// Drive a few requests through the middleware.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/hello", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Scrape /metrics.
	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mrec := httptest.NewRecorder()
	reg.Handler().ServeHTTP(mrec, mreq)

	body := mrec.Body.String()

	if !strings.Contains(body, `http_requests_total{method="GET",path="/api/hello",status="200"} 3`) {
		t.Fatalf("expected request counter of 3, got:\n%s", body)
	}
	if !strings.Contains(body, "http_request_duration_seconds_bucket") {
		t.Fatalf("expected latency histogram, got:\n%s", body)
	}
	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",path="/api/hello"} 3`) {
		t.Fatalf("expected histogram count of 3, got:\n%s", body)
	}
	if !strings.Contains(body, "app_build_info") {
		t.Fatalf("expected build_info metric, got:\n%s", body)
	}
}

func TestErrorStatusIsRecorded(t *testing.T) {
	reg := New("test")
	handler := reg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/error", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	body := string(reg.render())
	if !strings.Contains(body, `http_requests_total{method="GET",path="/api/error",status="500"} 1`) {
		t.Fatalf("expected a recorded 500, got:\n%s", body)
	}
}

func TestHistogramBucketsAreCumulative(t *testing.T) {
	reg := New("test")
	// Observations of 3ms, 30ms and 300ms for the same route.
	reg.observe("GET", "/x", "200", 0.003)
	reg.observe("GET", "/x", "200", 0.030)
	reg.observe("GET", "/x", "200", 0.300)

	body := string(reg.render())

	// le=0.005 should contain only the 3ms observation.
	if !strings.Contains(body, `http_request_duration_seconds_bucket{method="GET",path="/x",le="0.005"} 1`) {
		t.Fatalf("expected le=0.005 count of 1, got:\n%s", body)
	}
	// le=0.05 should contain the 3ms and 30ms observations.
	if !strings.Contains(body, `http_request_duration_seconds_bucket{method="GET",path="/x",le="0.05"} 2`) {
		t.Fatalf("expected le=0.05 count of 2, got:\n%s", body)
	}
	// +Inf and _count must equal the total number of observations.
	if !strings.Contains(body, `http_request_duration_seconds_bucket{method="GET",path="/x",le="+Inf"} 3`) {
		t.Fatalf("expected le=+Inf count of 3, got:\n%s", body)
	}
	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",path="/x"} 3`) {
		t.Fatalf("expected histogram count of 3, got:\n%s", body)
	}
}
