// Package metrics implements a small, dependency-free Prometheus exposition
// layer. It records HTTP request counts and latency histograms and renders
// them in the Prometheus text exposition format (v0.0.4) so any standard
// Prometheus server can scrape /metrics.
//
// Keeping this dependency-free keeps the binary tiny and the build fully
// reproducible offline. Swapping in prometheus/client_golang later is a
// drop-in change behind the same Middleware/Handler interface.
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"devops-showcase/internal/version"
)

// defaultBuckets are the cumulative upper bounds (seconds) for the request
// duration histogram. They mirror Prometheus client defaults.
var defaultBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

type counterKey struct {
	method string
	path   string
	status string
}

type histKey struct {
	method string
	path   string
}

type histogram struct {
	// bucketCounts[i] holds the number of observations <= defaultBuckets[i]
	// (already cumulative per bucket).
	bucketCounts []uint64
	sum          float64
	count        uint64
}

// Registry is a thread-safe collector of application metrics.
type Registry struct {
	mu           sync.Mutex
	requestTotal map[counterKey]uint64
	durations    map[histKey]*histogram

	inFlight  int64
	startTime time.Time
	env       string
}

// New creates a Registry. env is emitted as a label on build_info.
func New(env string) *Registry {
	return &Registry{
		requestTotal: make(map[counterKey]uint64),
		durations:    make(map[histKey]*histogram),
		startTime:    time.Now(),
		env:          env,
	}
}

func (r *Registry) observe(method, path, status string, seconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestTotal[counterKey{method, path, status}]++

	hk := histKey{method, path}
	h := r.durations[hk]
	if h == nil {
		h = &histogram{bucketCounts: make([]uint64, len(defaultBuckets))}
		r.durations[hk] = h
	}
	h.sum += seconds
	h.count++
	for i, ub := range defaultBuckets {
		if seconds <= ub {
			h.bucketCounts[i]++
		}
	}
}

// statusRecorder captures the response status code written by downstream
// handlers so it can be used as a metric label.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.status = http.StatusOK
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}

// Middleware wraps an http.Handler, recording request counts and latency.
// It normalises the path label to the matched route pattern (Go 1.22+ mux)
// to keep metric cardinality bounded.
func (r *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt64(&r.inFlight, 1)
		defer atomic.AddInt64(&r.inFlight, -1)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, req)

		path := req.Pattern
		if path == "" {
			path = req.URL.Path
		}
		r.observe(req.Method, path, strconv.Itoa(rec.status), time.Since(start).Seconds())
	})
}

// Handler renders all metrics in the Prometheus text exposition format.
// It builds the full payload under lock into a buffer and performs a single
// write, so a slow scraper can never hold the metrics lock.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := r.render()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write(body)
	})
}

// render produces the exposition payload. Kept separate so it is easy to test
// and so the response write happens outside the critical section.
func (r *Registry) render() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	var b strings.Builder
	info := version.Get()

	b.WriteString("# HELP app_build_info Build metadata of the running binary.\n")
	b.WriteString("# TYPE app_build_info gauge\n")
	fmt.Fprintf(&b, "app_build_info{version=%q,commit=%q,build_date=%q,env=%q} 1\n",
		info.Version, info.Commit, info.BuildDate, r.env)

	b.WriteString("# HELP app_uptime_seconds Seconds since process start.\n")
	b.WriteString("# TYPE app_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "app_uptime_seconds %g\n", time.Since(r.startTime).Seconds())

	b.WriteString("# HELP http_requests_in_flight Current number of in-flight HTTP requests.\n")
	b.WriteString("# TYPE http_requests_in_flight gauge\n")
	fmt.Fprintf(&b, "http_requests_in_flight %d\n", atomic.LoadInt64(&r.inFlight))

	b.WriteString("# HELP http_requests_total Total number of HTTP requests processed.\n")
	b.WriteString("# TYPE http_requests_total counter\n")
	ckeys := make([]counterKey, 0, len(r.requestTotal))
	for k := range r.requestTotal {
		ckeys = append(ckeys, k)
	}
	sort.Slice(ckeys, func(i, j int) bool {
		if ckeys[i].path != ckeys[j].path {
			return ckeys[i].path < ckeys[j].path
		}
		if ckeys[i].method != ckeys[j].method {
			return ckeys[i].method < ckeys[j].method
		}
		return ckeys[i].status < ckeys[j].status
	})
	for _, k := range ckeys {
		fmt.Fprintf(&b, "http_requests_total{method=%q,path=%q,status=%q} %d\n",
			k.method, k.path, k.status, r.requestTotal[k])
	}

	b.WriteString("# HELP http_request_duration_seconds HTTP request latency in seconds.\n")
	b.WriteString("# TYPE http_request_duration_seconds histogram\n")
	hkeys := make([]histKey, 0, len(r.durations))
	for k := range r.durations {
		hkeys = append(hkeys, k)
	}
	sort.Slice(hkeys, func(i, j int) bool {
		if hkeys[i].path != hkeys[j].path {
			return hkeys[i].path < hkeys[j].path
		}
		return hkeys[i].method < hkeys[j].method
	})
	for _, k := range hkeys {
		h := r.durations[k]
		for i, ub := range defaultBuckets {
			fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%q,path=%q,le=%q} %d\n",
				k.method, k.path, formatFloat(ub), h.bucketCounts[i])
		}
		fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%q,path=%q,le=\"+Inf\"} %d\n",
			k.method, k.path, h.count)
		fmt.Fprintf(&b, "http_request_duration_seconds_sum{method=%q,path=%q} %g\n",
			k.method, k.path, h.sum)
		fmt.Fprintf(&b, "http_request_duration_seconds_count{method=%q,path=%q} %d\n",
			k.method, k.path, h.count)
	}

	return []byte(b.String())
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}
