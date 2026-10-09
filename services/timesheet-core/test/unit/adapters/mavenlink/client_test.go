package mavenlink_test

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
	"timesheet-bot/internal/timesheet/infrastructure/mavenlink"
)

var week = kernel.WeekID{Year: 2026, Week: 41}

// createBody is the POST /time_entries.json body this adapter sends.
type createBody struct {
	TimeEntry struct {
		WorkspaceID       int64  `json:"workspace_id"`
		DatePerformed     string `json:"date_performed"`
		TimeInMinutes     int    `json:"time_in_minutes"`
		StoryID           int64  `json:"story_id"`
		RoleID            int64  `json:"role_id"`
		Notes             string `json:"notes"`
		Billable          *bool  `json:"billable"`
		ExternalReference struct {
			ServiceName     string `json:"service_name"`
			ServiceModel    string `json:"service_model"`
			ServiceModelRef string `json:"service_model_ref"`
		} `json:"external_reference"`
	} `json:"time_entry"`
}

type capturedRequest struct {
	method, path, auth, contentType string
	body                            []byte
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

func bearerClient(srv *httptest.Server, token string) *mavenlink.Client {
	return mavenlink.NewClient(srv.URL+"/api/v1/", mavenlink.NewBearer(token), srv.Client())
}

func TestCreatePostsTimeEntry(t *testing.T) {
	t.Parallel()
	seen := make(chan capturedRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- capturedRequest{
			method: r.Method, path: r.URL.Path,
			auth: r.Header.Get("Authorization"), contentType: r.Header.Get("Content-Type"), body: body,
		}
		writeBody(t, w, `{"count":1,"results":[{"key":"time_entries","id":"55"}],"time_entries":{"55":{"id":"55"}}}`)
	}))
	t.Cleanup(srv.Close)

	billable := false
	e := application.TimeEntry{
		EntryID: "ts-1/2026-10-05/GCM-1", WorkspaceID: 7, StoryID: 70, RoleID: 5, Billable: &billable,
		Day: monday(t), Minutes: 45, Notes: "Work on GCM-1",
	}
	id, err := bearerClient(srv, "tok").Create(t.Context(), e)
	if err != nil {
		t.Fatalf("Create(%+v) error = %v, want nil", e, err)
	}
	if id != "55" {
		t.Errorf("Create() = %q, want %q", id, "55")
	}

	got := <-seen
	if got.method != http.MethodPost || got.path != "/api/v1/time_entries.json" {
		t.Errorf("request = %s %s, want POST /api/v1/time_entries.json", got.method, got.path)
	}
	if got.auth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", got.auth, "Bearer tok")
	}
	if got.contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got.contentType)
	}
	var body createBody
	dec := json.NewDecoder(strings.NewReader(string(got.body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		t.Fatalf("decode request body %s: %v", got.body, err)
	}
	te := body.TimeEntry
	if te.WorkspaceID != 7 || te.DatePerformed != "2026-10-05" || te.TimeInMinutes != 45 || te.StoryID != 70 ||
		te.RoleID != 5 || te.Notes != "Work on GCM-1" || te.Billable == nil || *te.Billable {
		t.Errorf("time_entry = %+v, want project 7, 2026-10-05, 45m, task 70, role 5, not billable", te)
	}
	ref := te.ExternalReference
	if ref.ServiceName != mavenlink.ServiceName || ref.ServiceModel != mavenlink.ServiceModel || ref.ServiceModelRef != e.EntryID {
		t.Errorf("external_reference = %+v, want %s/%s/%s", ref, mavenlink.ServiceName, mavenlink.ServiceModel, e.EntryID)
	}
}

func TestCreateOmitsUnsetOptionalFields(t *testing.T) {
	t.Parallel()
	seen := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- body
		writeBody(t, w, `{"results":[{"key":"time_entries","id":"56"}]}`)
	}))
	t.Cleanup(srv.Close)

	if _, err := bearerClient(srv, "tok").Create(t.Context(), application.TimeEntry{WorkspaceID: 7, Day: monday(t), Minutes: 15}); err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	var raw struct {
		TimeEntry map[string]json.RawMessage `json:"time_entry"`
	}
	if err := json.Unmarshal(<-seen, &raw); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	for _, k := range []string{"story_id", "role_id", "billable", "notes"} {
		if _, ok := raw.TimeEntry[k]; ok {
			t.Errorf("time_entry has %q, want it omitted when unset", k)
		}
	}
}

func TestCreateFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "422 with a body", status: http.StatusUnprocessableEntity, body: `{"errors":[{"message":"time is locked"}]}`},
		{name: "404", status: http.StatusNotFound, body: `{}`},
		{name: "no time entry in results", status: http.StatusOK, body: `{"results":[]}`},
		{name: "malformed JSON", status: http.StatusOK, body: `{"results":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				writeBody(t, w, tt.body)
			}))
			t.Cleanup(srv.Close)

			id, err := bearerClient(srv, "tok").Create(t.Context(), application.TimeEntry{Day: monday(t)})
			if err == nil || id != "" {
				t.Errorf("Create() = %q, %v, want \"\" and an error", id, err)
			}
		})
	}
}

func TestErrorNamesReasonNotToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		// A server that echoes the request back must not leak the token through the error.
		writeBody(t, w, `{"errors":[{"message":"time is locked"}],"echo":"`+r.Header.Get("Authorization")+`"}`)
	}))
	t.Cleanup(srv.Close)

	_, err := bearerClient(srv, "tok-secret").Create(t.Context(), application.TimeEntry{Day: monday(t)})
	if err == nil {
		t.Fatal("Create() error = nil, want a 422 error")
	}
	if !strings.Contains(err.Error(), "time is locked") {
		t.Errorf("Create() error = %q, want it to include Mavenlink's reason", err)
	}
	if strings.Contains(err.Error(), "tok-secret") {
		t.Errorf("Create() error = %q, want it to omit the token", err)
	}
}

func TestBearerRejectedIsUnauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	_, err := bearerClient(srv, "tok").Created(t.Context(), week)
	if !errors.Is(err, application.ErrUnauthorized) {
		t.Errorf("Created() error = %v, want ErrUnauthorized", err)
	}
}

// Mavenlink stores the entry but answers 502. The production client
// (httpx.NewClient) must not post it again; the next run adopts it.
func TestCreateDoesNotResendOn5xx(t *testing.T) {
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

	_, err := mavenlink.NewClient(srv.URL, mavenlink.NewBearer("tok"), hc).Create(t.Context(), application.TimeEntry{WorkspaceID: 7, Day: monday(t), Minutes: 45})
	if err == nil {
		t.Errorf("Create() error = nil, want the 502 surfaced")
	}
	if got := posts.Load(); got != 1 {
		t.Errorf("POST /time_entries.json sent %d times, want 1 (never blindly re-sent)", got)
	}
}

func TestCreatedListsThisToolsEntriesAcrossPages(t *testing.T) {
	t.Parallel()
	pages := map[string]string{
		"1": `{"meta":{"page_count":2},
			"results":[{"key":"time_entries","id":"55"},{"key":"time_entries","id":"56"}],
			"time_entries":{
				"55":{"workspace_id":"7","active_submission_id":null,"external_reference_ids":["900"]},
				"56":{"workspace_id":"8","active_submission_id":"31","external_reference_ids":["901","902"]}},
			"external_references":{
				"900":{"service_name":"timesheet-bot","service_model":"timesheet_entry","service_model_ref":"ts-1/2026-10-05/GCM-1"},
				"901":{"service_name":"jira","service_model":"issue","service_model_ref":"GCM-2"},
				"902":{"service_name":"timesheet-bot","service_model":"timesheet_entry","service_model_ref":"ts-1/2026-10-05/GCM-2"}}}`,
		"2": `{"meta":{"page_count":2},
			"results":[{"key":"time_entries","id":57}],
			"time_entries":{"57":{"workspace_id":7,"external_reference_ids":["903"]}},
			"external_references":{
				"903":{"service_name":"timesheet-bot","service_model":"timesheet_entry","service_model_ref":"ts-1/2026-10-06/GCM-1"}}}`,
	}
	queries := make(chan map[string]string, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		queries <- map[string]string{
			"path": r.URL.Path, "between": q.Get("date_performed_between"),
			"service": q.Get("external_reference_service_name"), "include": q.Get("include"), "per_page": q.Get("per_page"),
		}
		writeBody(t, w, pages[q.Get("page")])
	}))
	t.Cleanup(srv.Close)

	got, err := bearerClient(srv, "tok").Created(t.Context(), week)
	if err != nil {
		t.Fatalf("Created(%v) error = %v, want nil", week, err)
	}
	want := map[string]application.RemoteTimeEntry{
		"ts-1/2026-10-05/GCM-1": {ID: "55", WorkspaceID: 7},
		"ts-1/2026-10-05/GCM-2": {ID: "56", WorkspaceID: 8, InSubmission: true},
		"ts-1/2026-10-06/GCM-1": {ID: "57", WorkspaceID: 7},
	}
	if len(got) != len(want) {
		t.Errorf("Created(%v) = %+v, want %+v", week, got, want)
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("Created(%v)[%s] = %+v, want %+v", week, k, got[k], w)
		}
	}
	q := <-queries
	if q["path"] != "/api/v1/time_entries.json" || q["between"] != "2026-10-05:2026-10-09" || q["service"] != mavenlink.ServiceName ||
		q["include"] != "external_references,workspace,active_submission" || q["per_page"] != "200" {
		t.Errorf("list query = %v, want week 2026-10-05:2026-10-09, this tool's service, the includes, 200 a page", q)
	}
}

func TestSubmitForApprovalPostsSubmission(t *testing.T) {
	t.Parallel()
	seen := make(chan capturedRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- capturedRequest{method: r.Method, path: r.URL.Path, body: body}
		writeBody(t, w, `{"results":[{"key":"timesheet_submissions","id":"31"}]}`)
	}))
	t.Cleanup(srv.Close)

	if err := bearerClient(srv, "tok").SubmitForApproval(t.Context(), 7, "Week 2026-W41", []string{"55", "57"}); err != nil {
		t.Fatalf("SubmitForApproval() error = %v, want nil", err)
	}
	got := <-seen
	if got.method != http.MethodPost || got.path != "/api/v1/timesheet_submissions.json" {
		t.Errorf("request = %s %s, want POST /api/v1/timesheet_submissions.json", got.method, got.path)
	}
	want := `{"timesheet_submission":{"workspace_id":7,"line_item_ids":[55,57],"title":"Week 2026-W41"}}`
	if string(got.body) != want {
		t.Errorf("request body = %s, want %s", got.body, want)
	}
}

func TestSubmitForApprovalRejectsNonNumericID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Errorf("request reached Mavenlink, want none for a bad id")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := bearerClient(srv, "tok").SubmitForApproval(t.Context(), 7, "Week 2026-W41", []string{"m-1"}); err == nil {
		t.Errorf("SubmitForApproval(id m-1) error = nil, want an error")
	}
}

func TestCreateHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Errorf("request reached Mavenlink, want none after cancel")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := bearerClient(srv, "tok").Create(ctx, application.TimeEntry{Day: monday(t)})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Create(canceled ctx) error = %v, want context.Canceled", err)
	}
}
