package tempo_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"timesheet-bot/internal/shared/httpx"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/infrastructure/tempo"
)

// createBody is the Tempo v4 POST /worklogs body this adapter sends.
type createBody struct {
	AuthorAccountID  string `json:"authorAccountId"`
	IssueID          int64  `json:"issueId"`
	StartDate        string `json:"startDate"`
	StartTime        string `json:"startTime"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	Description      string `json:"description"`
}

// capturedRequest is what the fake Tempo server saw, handed to the test
// goroutine over a channel.
type capturedRequest struct {
	method, path, auth, contentType string
	body                            createBody
	decodeErr                       error
}

func writeBody(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write response body: %v", err)
	}
}

func monday(t *testing.T) kernel.WorkDay {
	t.Helper()
	day, err := kernel.ParseWorkDay("2026-10-05")
	if err != nil {
		t.Fatalf("ParseWorkDay: %v", err)
	}
	return day
}

func TestCreatePostsWorklog(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			seen := make(chan capturedRequest, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c := capturedRequest{
					method: r.Method, path: r.URL.Path,
					auth: r.Header.Get("Authorization"), contentType: r.Header.Get("Content-Type"),
				}
				dec := json.NewDecoder(r.Body)
				dec.DisallowUnknownFields()
				c.decodeErr = dec.Decode(&c.body)
				seen <- c
				w.WriteHeader(status)
				writeBody(t, w, `{"tempoWorklogId":126,"issue":{"id":10042}}`)
			}))
			t.Cleanup(srv.Close)

			w := application.Worklog{
				IssueID: 10042, Day: monday(t), StartTime: "09:45:00", Seconds: 2700,
				Description: "Work on GCM-12", AuthorAccountID: "acc-1",
			}
			id, err := tempo.NewClient(srv.URL+"/4/", "tok", srv.Client()).Create(t.Context(), w)
			if err != nil {
				t.Fatalf("Create(%+v) error = %v, want nil", w, err)
			}
			if id != "126" {
				t.Errorf("Create(%+v) = %q, want %q", w, id, "126")
			}

			got := <-seen
			if got.method != http.MethodPost || got.path != "/4/worklogs" {
				t.Errorf("request = %s %s, want POST /4/worklogs", got.method, got.path)
			}
			if got.auth != "Bearer tok" {
				t.Errorf("Authorization = %q, want %q", got.auth, "Bearer tok")
			}
			if got.contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got.contentType)
			}
			if got.decodeErr != nil {
				t.Fatalf("decode request body: %v", got.decodeErr)
			}
			want := createBody{
				AuthorAccountID: "acc-1", IssueID: 10042, StartDate: "2026-10-05",
				StartTime: "09:45:00", TimeSpentSeconds: 2700, Description: "Work on GCM-12",
			}
			if got.body != want {
				t.Errorf("request body = %+v, want %+v", got.body, want)
			}
		})
	}
}

func TestCreateFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "400 with a body", status: http.StatusBadRequest, body: `{"errors":[{"message":"period closed"}]}`},
		{name: "500", status: http.StatusInternalServerError, body: `oops`},
		{name: "missing tempoWorklogId", status: http.StatusOK, body: `{"issue":{"id":10042}}`},
		{name: "malformed JSON", status: http.StatusOK, body: `{"tempoWorklogId":`},
		{name: "non-numeric tempoWorklogId", status: http.StatusOK, body: `{"tempoWorklogId":"126"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				writeBody(t, w, tt.body)
			}))
			t.Cleanup(srv.Close)

			id, err := tempo.NewClient(srv.URL, "tok", srv.Client()).Create(t.Context(), application.Worklog{Day: monday(t)})
			if err == nil || id != "" {
				t.Errorf("Create() = %q, %v, want \"\" and an error", id, err)
			}
		})
	}
}

// The adapter has no error sentinels, so the reason Tempo gave can only be
// checked in the message. The token must never appear there.
func TestCreateErrorNamesTempoReasonNotToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeBody(t, w, `{"errors":[{"message":"period closed"}]}`)
	}))
	t.Cleanup(srv.Close)

	_, err := tempo.NewClient(srv.URL, "tok-secret", srv.Client()).Create(t.Context(), application.Worklog{Day: monday(t)})
	if err == nil {
		t.Fatal("Create() error = nil, want a 400 error")
	}
	if !strings.Contains(err.Error(), "period closed") {
		t.Errorf("Create() error = %q, want it to include Tempo's reason", err)
	}
	if strings.Contains(err.Error(), "tok-secret") {
		t.Errorf("Create() error = %q, want it to omit the token", err)
	}
}

// AC-005.3: Tempo stores the worklog but answers 502. The production client
// (httpx.NewClient) must not post it again.
func TestCreateDoesNotResendWorklogOn5xx(t *testing.T) {
	t.Skip("BUG H2: POST re-sent on 5xx; un-skip with US-005-T2")
	t.Parallel()
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	// Same transport as httpx.NewClient, with the backoff sleep removed.
	hc := &http.Client{Transport: &httpx.RetryTransport{Sleep: func(time.Duration) {}}}

	_, err := tempo.NewClient(srv.URL, "tok", hc).Create(t.Context(), application.Worklog{IssueID: 10042, Day: monday(t), Seconds: 2700})
	if err == nil {
		t.Errorf("Create() error = nil, want the 502 surfaced")
	}
	if got := posts.Load(); got != 1 {
		t.Errorf("POST /worklogs sent %d times, want 1 (never blindly re-sent)", got)
	}
}

func TestCreateHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Errorf("request reached Tempo, want none after cancel")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := tempo.NewClient(srv.URL, "tok", srv.Client()).Create(ctx, application.Worklog{Day: monday(t)})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Create(canceled ctx) error = %v, want context.Canceled", err)
	}
}
