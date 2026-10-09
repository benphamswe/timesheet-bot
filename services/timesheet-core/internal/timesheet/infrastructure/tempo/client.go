// Package tempo writes worklogs through the Tempo Cloud REST API v4.
package tempo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"timesheet-bot/internal/timesheet/application"
)

// DefaultBaseURL is the universal endpoint. Regional ones are
// https://api.eu.tempo.io/4 and https://api.us.tempo.io/4.
const DefaultBaseURL = "https://api.tempo.io/4"

// Client is an [application.WorklogGateway] backed by the Tempo Cloud REST API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient returns a Client that sends requests to baseURL, or to
// [DefaultBaseURL] if baseURL is empty, through hc with token as the bearer
// token.
func NewClient(baseURL, token string, hc *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: hc}
}

var _ application.WorklogGateway = (*Client)(nil)

type createRequest struct {
	AuthorAccountID  string `json:"authorAccountId"`
	IssueID          int64  `json:"issueId"`
	StartDate        string `json:"startDate"`
	StartTime        string `json:"startTime,omitempty"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	Description      string `json:"description,omitempty"`
}

// Create posts one worklog and returns its tempoWorklogId.
func (c *Client) Create(ctx context.Context, w application.Worklog) (string, error) {
	payload, err := json.Marshal(createRequest{
		AuthorAccountID:  w.AuthorAccountID,
		IssueID:          w.IssueID,
		StartDate:        w.Day.String(),
		StartTime:        w.StartTime,
		TimeSpentSeconds: w.Seconds,
		Description:      w.Description,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/worklogs", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("tempo create worklog: %s: %s", resp.Status, body)
	}
	var out struct {
		TempoWorklogID int64 `json:"tempoWorklogId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode tempo response: %w", err)
	}
	if out.TempoWorklogID == 0 {
		return "", fmt.Errorf("tempo response has no tempoWorklogId")
	}
	return strconv.FormatInt(out.TempoWorklogID, 10), nil
}
