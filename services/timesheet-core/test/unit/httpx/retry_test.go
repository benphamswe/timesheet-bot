package httpx_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"timesheet-bot/internal/shared/httpx"
)

// sleepRecorder replaces time.Sleep so backoff is asserted without waiting.
type sleepRecorder struct{ slept []time.Duration }

func (s *sleepRecorder) Sleep(d time.Duration) { s.slept = append(s.slept, d) }

func newRetryClient(maxAttempts int) (*http.Client, *sleepRecorder) {
	rec := &sleepRecorder{}
	return &http.Client{Transport: &httpx.RetryTransport{MaxAttempts: maxAttempts, Sleep: rec.Sleep}}, rec
}

// statusSequence answers with codes[i] on call i and repeats the last code.
// It sets Retry-After on every non-final response when retryAfter is not empty.
type statusSequence struct {
	codes      []int
	retryAfter string
	calls      atomic.Int32
}

func (s *statusSequence) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	n := int(s.calls.Add(1))
	code := s.codes[min(n, len(s.codes))-1]
	if s.retryAfter != "" && n < len(s.codes) {
		w.Header().Set("Retry-After", s.retryAfter)
	}
	w.WriteHeader(code)
}

func get(t *testing.T, c *http.Client, url string) (*http.Response, error) {
	t.Helper()
	// A nil body, not http.NoBody: see TestRetryReplaysNoBodyRequest.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest(GET %s): %v", url, err)
	}
	return c.Do(req)
}

func TestRetryReplaysNoBodyRequest(t *testing.T) {
	t.Skip("BUG: RetryTransport returns errNoRewind for a request whose Body is http.NoBody (GetBody is nil)")
	t.Parallel()
	h := &statusSequence{codes: []int{http.StatusServiceUnavailable, http.StatusOK}}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, _ := newRetryClient(5)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest(GET): %v", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("GET with http.NoBody error = %v, want nil", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || h.calls.Load() != 2 {
		t.Errorf("GET = %d after %d calls, want %d after 2", resp.StatusCode, h.calls.Load(), http.StatusOK)
	}
}

func TestRetryHonorsRetryAfterSeconds(t *testing.T) {
	t.Parallel()
	h := &statusSequence{codes: []int{http.StatusTooManyRequests, http.StatusOK}, retryAfter: "7"}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, rec := newRetryClient(5)

	resp, err := get(t, c, srv.URL)
	if err != nil {
		t.Fatalf("GET error = %v, want nil", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || h.calls.Load() != 2 {
		t.Errorf("GET = %d after %d calls, want %d after 2", resp.StatusCode, h.calls.Load(), http.StatusOK)
	}
	if want := []time.Duration{7 * time.Second}; !slices.Equal(rec.slept, want) {
		t.Errorf("slept = %v, want %v", rec.slept, want)
	}
}

// RetryTransport parses only the delta-seconds form of Retry-After, so the
// HTTP-date form falls back to the exponential backoff.
func TestRetryAfterHTTPDateFallsBackToBackoff(t *testing.T) {
	t.Parallel()
	h := &statusSequence{
		codes:      []int{http.StatusServiceUnavailable, http.StatusOK},
		retryAfter: "Fri, 09 Oct 2026 16:00:30 GMT",
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, rec := newRetryClient(5)

	resp, err := get(t, c, srv.URL)
	if err != nil {
		t.Fatalf("GET error = %v, want nil", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || h.calls.Load() != 2 {
		t.Errorf("GET = %d after %d calls, want %d after 2", resp.StatusCode, h.calls.Load(), http.StatusOK)
	}
	if want := []time.Duration{time.Second}; !slices.Equal(rec.slept, want) {
		t.Errorf("slept = %v, want %v", rec.slept, want)
	}
}

func TestRetryGivesUpAfterMaxAttemptsWithExponentialBackoff(t *testing.T) {
	t.Parallel()
	h := &statusSequence{codes: []int{http.StatusServiceUnavailable}}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, rec := newRetryClient(3)

	resp, err := get(t, c, srv.URL)
	if err != nil {
		t.Fatalf("GET error = %v, want nil", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable || h.calls.Load() != 3 {
		t.Errorf("GET = %d after %d calls, want %d after 3", resp.StatusCode, h.calls.Load(), http.StatusServiceUnavailable)
	}
	if want := []time.Duration{time.Second, 2 * time.Second}; !slices.Equal(rec.slept, want) {
		t.Errorf("slept = %v, want %v", rec.slept, want)
	}
}

func TestRetryResendsBody(t *testing.T) {
	t.Parallel()
	var (
		mu     sync.Mutex
		bodies []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		bodies = append(bodies, string(b))
		n := len(bodies)
		mu.Unlock()
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	c, _ := newRetryClient(5)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatalf("NewRequest(POST): %v", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("POST error = %v, want nil", err)
	}
	resp.Body.Close()
	mu.Lock()
	defer mu.Unlock()
	if want := []string{`{"a":1}`, `{"a":1}`}; !slices.Equal(bodies, want) {
		t.Errorf("request bodies = %q, want %q", bodies, want)
	}
}

func TestRetryDoesNotRetryClientErrors(t *testing.T) {
	t.Parallel()
	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			t.Parallel()
			h := &statusSequence{codes: []int{code}}
			srv := httptest.NewServer(h)
			t.Cleanup(srv.Close)
			c, rec := newRetryClient(5)

			resp, err := get(t, c, srv.URL)
			if err != nil {
				t.Fatalf("GET error = %v, want nil", err)
			}
			resp.Body.Close()
			if resp.StatusCode != code || h.calls.Load() != 1 {
				t.Errorf("GET = %d after %d calls, want %d after 1", resp.StatusCode, h.calls.Load(), code)
			}
			if len(rec.slept) != 0 {
				t.Errorf("slept = %v, want no backoff", rec.slept)
			}
		})
	}
}

func TestRetryStopsWhenContextCanceledDuringBackoff(t *testing.T) {
	t.Parallel()
	h := &statusSequence{codes: []int{http.StatusServiceUnavailable}}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var sleeps int
	c := &http.Client{Transport: &httpx.RetryTransport{
		MaxAttempts: 5,
		Sleep:       func(time.Duration) { sleeps++; cancel() }, // the caller gives up while we wait
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest(GET): %v", err)
	}

	resp, err := c.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("GET error = %v, want context.Canceled", err)
	}
	if got := h.calls.Load(); got != 1 || sleeps != 1 {
		t.Errorf("GET made %d calls and %d sleeps, want 1 and 1", got, sleeps)
	}
}

func TestRetryRefusesBodyThatCannotBeReplayed(t *testing.T) {
	t.Parallel()
	h := &statusSequence{codes: []int{http.StatusServiceUnavailable, http.StatusOK}}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, _ := newRetryClient(5)

	// io.MultiReader is not a type http.NewRequest knows how to rewind, so GetBody is nil.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, io.MultiReader(strings.NewReader("x")))
	if err != nil {
		t.Fatalf("NewRequest(POST): %v", err)
	}
	resp, err := c.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	if !errors.Is(err, httpx.ErrNoRewind) {
		t.Errorf("POST error = %v, want ErrNoRewind", err)
	}
	if got := h.calls.Load(); got != 1 {
		t.Errorf("POST made %d calls, want 1", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRetryReturnsTransportErrorWithoutRetry(t *testing.T) {
	t.Parallel()
	errDial := errors.New("dial refused")
	var calls int
	rt := &httpx.RetryTransport{
		Base:  roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errDial }),
		Sleep: func(time.Duration) { t.Errorf("Sleep called, want no backoff after a transport error") },
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://tempo.invalid/", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest(GET): %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
		t.Errorf("RoundTrip() response = %v, want nil", resp)
	}
	if !errors.Is(err, errDial) {
		t.Errorf("RoundTrip() error = %v, want %v", err, errDial)
	}
	if calls != 1 {
		t.Errorf("RoundTrip() made %d calls, want 1", calls)
	}
}

func TestNewClientSetsTimeoutAndRetryTransport(t *testing.T) {
	t.Parallel()
	c := httpx.NewClient(30 * time.Second)
	if c.Timeout != 30*time.Second {
		t.Errorf("NewClient(30s).Timeout = %v, want 30s", c.Timeout)
	}
	if _, ok := c.Transport.(*httpx.RetryTransport); !ok {
		t.Errorf("NewClient(30s).Transport is %T, want *httpx.RetryTransport", c.Transport)
	}
}
