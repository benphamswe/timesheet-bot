// Package mavenlink writes time entries and timesheet submissions through the
// Mavenlink (Kantata OX) REST API v1.
package mavenlink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"timesheet-bot/internal/shared/httpx"
	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
)

// DefaultBaseURL is the endpoint for [Bearer] auth. [SessionCookie] auth needs
// the account's own host instead, such as https://<account>.mavenlink.com/api/v1.
const DefaultBaseURL = "https://api.mavenlink.com/api/v1"

// The external reference every created time entry carries. Created finds a
// run's entries by it, and service_model_ref holds the timesheet entry ID.
const (
	ServiceName  = "timesheet-bot"
	ServiceModel = "timesheet_entry"
)

// maxPages bounds a listing, at 200 time entries a page.
const maxPages = 20

// Client is an [application.TimeEntryGateway] backed by the Mavenlink API.
type Client struct {
	baseURL string
	auth    Auth
	http    *http.Client
}

// NewClient returns a Client that sends requests to baseURL, or to
// [DefaultBaseURL] if baseURL is empty, through a copy of hc that never
// follows redirects, signed in with auth.
func NewClient(baseURL string, auth Auth, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), auth: auth, http: withoutRedirects(hc)}
}

var _ application.TimeEntryGateway = (*Client)(nil)

// id is a Mavenlink id. The API sends ids as JSON strings; numbers are
// accepted too.
type id string

func (i *id) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*i = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*i = id(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("mavenlink id %s: %w", b, err)
	}
	*i = id(n.String())
	return nil
}

type result struct {
	Key string `json:"key"`
	ID  id     `json:"id"`
}

type listResponse struct {
	Meta struct {
		PageCount int `json:"page_count"`
	} `json:"meta"`
	Results     []result `json:"results"`
	TimeEntries map[string]struct {
		WorkspaceID          id   `json:"workspace_id"`
		ActiveSubmissionID   id   `json:"active_submission_id"`
		ExternalReferenceIDs []id `json:"external_reference_ids"`
	} `json:"time_entries"`
	ExternalReferences map[string]externalReference `json:"external_references"`
}

type externalReference struct {
	ServiceName     string `json:"service_name"`
	ServiceModel    string `json:"service_model"`
	ServiceModelRef string `json:"service_model_ref"`
}

// Created lists the time entries on week's days that carry this tool's
// external reference, keyed by the timesheet entry ID it holds.
func (c *Client) Created(ctx context.Context, week kernel.WeekID) (map[string]application.RemoteTimeEntry, error) {
	days := week.Days()
	q := url.Values{
		"date_performed_between":          {days[0].String() + ":" + days[len(days)-1].String()},
		"external_reference_service_name": {ServiceName},
		"include":                         {"external_references,workspace,active_submission"},
		"per_page":                        {"200"},
	}
	out := map[string]application.RemoteTimeEntry{}
	for page := 1; page <= maxPages; page++ {
		q.Set("page", strconv.Itoa(page))
		var r listResponse
		if err := c.do(ctx, http.MethodGet, "/time_entries.json", q, nil, &r); err != nil {
			return nil, err
		}
		for _, res := range r.Results {
			te, ok := r.TimeEntries[string(res.ID)]
			if res.Key != "time_entries" || !ok {
				continue
			}
			ws, err := strconv.ParseInt(string(te.WorkspaceID), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("mavenlink time entry %s: workspace_id %q is not a number", res.ID, te.WorkspaceID)
			}
			for _, refID := range te.ExternalReferenceIDs {
				ref := r.ExternalReferences[string(refID)]
				if ref.ServiceName == ServiceName && ref.ServiceModel == ServiceModel && ref.ServiceModelRef != "" {
					out[ref.ServiceModelRef] = application.RemoteTimeEntry{
						ID: string(res.ID), WorkspaceID: ws, InSubmission: te.ActiveSubmissionID != "",
					}
				}
			}
		}
		if page >= r.Meta.PageCount {
			return out, nil
		}
	}
	return nil, fmt.Errorf("mavenlink list time entries: more than %d pages", maxPages)
}

type createRequest struct {
	TimeEntry timeEntryBody `json:"time_entry"`
}

type timeEntryBody struct {
	WorkspaceID       int64             `json:"workspace_id"`
	DatePerformed     string            `json:"date_performed"`
	TimeInMinutes     int               `json:"time_in_minutes"`
	StoryID           int64             `json:"story_id,omitempty"`
	RoleID            int64             `json:"role_id,omitempty"`
	Notes             string            `json:"notes,omitempty"`
	Billable          *bool             `json:"billable,omitempty"`
	ExternalReference externalReference `json:"external_reference"`
}

type createResponse struct {
	Results []result `json:"results"`
}

// Create posts one time entry, tagged with e.EntryID as its external
// reference, and returns its id. It is never re-sent: after a 429, a 5xx or
// a timeout the next submit run finds it by the reference if it was stored.
func (c *Client) Create(ctx context.Context, e application.TimeEntry) (string, error) {
	body := createRequest{TimeEntry: timeEntryBody{
		WorkspaceID:   e.WorkspaceID,
		DatePerformed: e.Day.String(),
		TimeInMinutes: e.Minutes,
		StoryID:       e.StoryID,
		RoleID:        e.RoleID,
		Notes:         e.Notes,
		Billable:      e.Billable,
		ExternalReference: externalReference{
			ServiceName: ServiceName, ServiceModel: ServiceModel, ServiceModelRef: e.EntryID,
		},
	}}
	var out createResponse
	if err := c.do(ctx, http.MethodPost, "/time_entries.json", nil, body, &out); err != nil {
		return "", err
	}
	for _, r := range out.Results {
		if r.Key == "time_entries" && r.ID != "" {
			return string(r.ID), nil
		}
	}
	return "", errors.New("mavenlink response has no time entry id")
}

type submissionRequest struct {
	TimesheetSubmission submissionBody `json:"timesheet_submission"`
}

type submissionBody struct {
	WorkspaceID int64   `json:"workspace_id"`
	LineItemIDs []int64 `json:"line_item_ids"`
	Title       string  `json:"title"`
}

// SubmitForApproval posts one timesheet submission for workspaceID holding
// timeEntryIDs. Like Create, it is never re-sent.
func (c *Client) SubmitForApproval(ctx context.Context, workspaceID int64, title string, timeEntryIDs []string) error {
	ids := make([]int64, len(timeEntryIDs))
	for i, s := range timeEntryIDs {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("mavenlink time entry id %q is not a number", s)
		}
		ids[i] = n
	}
	body := submissionRequest{TimesheetSubmission: submissionBody{WorkspaceID: workspaceID, LineItemIDs: ids, Title: title}}
	var out createResponse
	if err := c.do(ctx, http.MethodPost, "/timesheet_submissions.json", nil, body, &out); err != nil {
		return err
	}
	for _, r := range out.Results {
		if r.Key == "timesheet_submissions" && r.ID != "" {
			return nil
		}
	}
	return errors.New("mavenlink response has no timesheet submission id")
}

// do sends one request with in as the JSON body (nil for none) and decodes
// the response into out.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, in, out any) error {
	target := c.baseURL + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
		// Without GetBody the retry transport cannot send the request a
		// second time (coding-standards R3: a create may have been stored).
		req.GetBody = nil
	}
	if err := c.auth.Authorize(ctx, req); err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if errors.Is(err, httpx.ErrNoRewind) {
		return fmt.Errorf("mavenlink %s %s: got 429 or 5xx and did not re-send, because it may have been stored; run submit again", method, path)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := c.auth.Rejected(req, resp); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("mavenlink %s %s: %s: %s", method, path, resp.Status, c.redact(string(snippet)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode mavenlink response: %w", err)
	}
	return nil
}

func (c *Client) redact(s string) string {
	for _, secret := range c.auth.Secrets() {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "[redacted]")
		}
	}
	return s
}
