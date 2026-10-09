// Package httpx holds HTTP plumbing shared by the external API adapters.
package httpx

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

// RetryTransport retries 429 and 5xx responses with exponential backoff,
// honoring a Retry-After header given in seconds.
type RetryTransport struct {
	Base        http.RoundTripper   // defaults to http.DefaultTransport
	MaxAttempts int                 // total attempts, defaults to 5
	Sleep       func(time.Duration) // defaults to time.Sleep; replaced in tests
}

// RoundTrip implements [http.RoundTripper]. It sends req and, while the
// response is 429 or 5xx and attempts remain, drains and closes that response,
// waits, and sends req again with a fresh body from req.GetBody. It returns the
// last response or transport error. If a retry is due but req has a body and
// no GetBody, it returns [ErrNoRewind] instead; if req's context ends between
// attempts, it returns the context's error.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base, sleep, maxAttempts := t.Base, t.Sleep, t.MaxAttempts
	if base == nil {
		base = http.DefaultTransport
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	backoff := time.Second
	for attempt := 1; ; attempt++ {
		r := req
		if attempt > 1 && req.Body != nil {
			if req.GetBody == nil {
				return nil, ErrNoRewind
			}
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			r = req.Clone(req.Context())
			r.Body = body
		}

		resp, err := base.RoundTrip(r)
		if err != nil || attempt == maxAttempts || !retryable(resp.StatusCode) {
			return resp, err
		}

		wait := backoff
		if s, convErr := strconv.Atoi(resp.Header.Get("Retry-After")); convErr == nil && s >= 0 {
			wait = time.Duration(s) * time.Second
		}
		// This response is discarded in favor of the next attempt. Draining it
		// lets the connection be reused; a drain or close error only costs that
		// reuse and does not affect the retry, so both errors are ignored.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		default:
		}
		sleep(wait)
		backoff *= 2
	}
}

func retryable(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

type rewindError struct{}

func (rewindError) Error() string { return "httpx: request body cannot be replayed for retry" }

// ErrNoRewind is returned by [RetryTransport.RoundTrip] when a retry is due but
// the request has a body and no GetBody, so the body cannot be sent again.
var ErrNoRewind error = rewindError{}

// NewClient returns an http.Client with retries and a timeout.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &RetryTransport{}}
}
