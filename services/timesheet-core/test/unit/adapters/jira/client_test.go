package jira_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"timesheet-bot/internal/timesheet/application"
	"timesheet-bot/internal/timesheet/infrastructure/jira"
)

func writeBody(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write response body: %v", err)
	}
}

// fixedIssue answers every request with status and body and counts the calls.
func fixedIssue(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(status)
		writeBody(t, w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestResolveReturnsIssueAndCachesIt(t *testing.T) {
	t.Parallel()
	type captured struct {
		path, fields, accept, user, pass string
		basicOK                          bool
	}
	seen := make(chan captured, 2)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		user, pass, ok := r.BasicAuth()
		seen <- captured{
			path: r.URL.Path, fields: r.URL.Query().Get("fields"), accept: r.Header.Get("Accept"),
			user: user, pass: pass, basicOK: ok,
		}
		writeBody(t, w, `{"id":"10042","key":"GCM-12","fields":{"summary":"Fix login"}}`)
	}))
	t.Cleanup(srv.Close)
	c := jira.NewClient(srv.URL+"/", "me@example.com", "tok", srv.Client())

	got, err := c.Resolve(t.Context(), "GCM-12")
	if err != nil {
		t.Fatalf("Resolve(GCM-12) error = %v, want nil", err)
	}
	want := application.Issue{ID: 10042, Key: "GCM-12", Summary: "Fix login"}
	if got != want {
		t.Errorf("Resolve(GCM-12) = %+v, want %+v", got, want)
	}
	req := <-seen
	wantReq := captured{
		path: "/rest/api/3/issue/GCM-12", fields: "summary", accept: "application/json",
		user: "me@example.com", pass: "tok", basicOK: true,
	}
	if req != wantReq {
		t.Errorf("request = %+v, want %+v", req, wantReq)
	}

	again, err := c.Resolve(t.Context(), "GCM-12")
	if err != nil || again != want {
		t.Errorf("cached Resolve(GCM-12) = %+v, %v, want %+v, nil", again, err, want)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("Jira called %d times for two lookups, want 1 (cached)", n)
	}
}

func TestResolveKeepsRequestedKeyWhenIssueMoved(t *testing.T) {
	t.Parallel()
	srv, _ := fixedIssue(t, http.StatusOK, `{"id":"10042","key":"NEW-7","fields":{"summary":"Moved"}}`)
	got, err := jira.NewClient(srv.URL, "me@example.com", "tok", srv.Client()).Resolve(t.Context(), "OLD-3")
	if err != nil {
		t.Fatalf("Resolve(OLD-3) error = %v, want nil", err)
	}
	if got.Key != "OLD-3" {
		t.Errorf("Resolve(OLD-3).Key = %q, want %q", got.Key, "OLD-3")
	}
}

func TestResolveMissingIssueIsNotFound(t *testing.T) {
	t.Parallel()
	srv, calls := fixedIssue(t, http.StatusNotFound, `{"errorMessages":["Issue does not exist"]}`)
	c := jira.NewClient(srv.URL, "me@example.com", "tok", srv.Client())

	for range 2 {
		if _, err := c.Resolve(t.Context(), "UTF-8"); !errors.Is(err, application.ErrIssueNotFound) {
			t.Errorf("Resolve(UTF-8) error = %v, want ErrIssueNotFound", err)
		}
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("Jira called %d times for two misses, want 2 (misses are not cached here)", n)
	}
}

// Anything other than a 404 must not look like ErrIssueNotFound: the
// application drops not-found keys but aborts the run on other failures.
func TestResolveFailuresAreNotNotFound(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "503", status: http.StatusServiceUnavailable, body: `busy`},
		{name: "401", status: http.StatusUnauthorized, body: ``},
		{name: "non-numeric id", status: http.StatusOK, body: `{"id":"abc","key":"GCM-12"}`},
		{name: "missing id", status: http.StatusOK, body: `{"key":"GCM-12"}`},
		{name: "malformed JSON", status: http.StatusOK, body: `{"id":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, _ := fixedIssue(t, tt.status, tt.body)
			c := jira.NewClient(srv.URL, "me@example.com", "tok-secret", srv.Client())

			got, err := c.Resolve(t.Context(), "GCM-12")
			if err == nil {
				t.Fatalf("Resolve(GCM-12) = %+v, nil, want an error", got)
			}
			if errors.Is(err, application.ErrIssueNotFound) {
				t.Errorf("Resolve(GCM-12) error = %v, want an error that is not ErrIssueNotFound", err)
			}
			if strings.Contains(err.Error(), "tok-secret") {
				t.Errorf("Resolve(GCM-12) error = %q, want it to omit the token", err)
			}
		})
	}
}
