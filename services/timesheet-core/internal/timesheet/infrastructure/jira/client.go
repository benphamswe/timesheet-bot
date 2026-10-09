// Package jira resolves issue keys through the Jira Cloud REST API v3.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"timesheet-bot/internal/shared/kernel"
	"timesheet-bot/internal/timesheet/application"
)

// Client is an [application.IssueDirectory] backed by Jira Cloud. It caches
// resolved issues for its lifetime and is safe for concurrent use.
type Client struct {
	baseURL string
	email   string
	token   string
	http    *http.Client

	mu    sync.Mutex
	cache map[kernel.IssueKey]application.Issue
}

// NewClient returns a Client for the Jira site at baseURL that authenticates
// with email and API token over HTTP basic auth and sends requests through hc.
func NewClient(baseURL, email, token string, hc *http.Client) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"), email: email, token: token, http: hc,
		cache: map[kernel.IssueKey]application.Issue{},
	}
}

var _ application.IssueDirectory = (*Client)(nil)

// Resolve returns the issue for key, or application.ErrIssueNotFound.
func (c *Client) Resolve(ctx context.Context, key kernel.IssueKey) (application.Issue, error) {
	c.mu.Lock()
	if i, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return i, nil
	}
	c.mu.Unlock()

	u := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=summary", c.baseURL, url.PathEscape(key.String()))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return application.Issue{}, err
	}
	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return application.Issue{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return application.Issue{}, fmt.Errorf("%w: %s", application.ErrIssueNotFound, key)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return application.Issue{}, fmt.Errorf("jira GET issue %s: %s: %s", key, resp.Status, body)
	}

	var body struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return application.Issue{}, fmt.Errorf("decode jira issue %s: %w", key, err)
	}
	id, err := strconv.ParseInt(body.ID, 10, 64)
	if err != nil {
		return application.Issue{}, fmt.Errorf("jira issue %s id %q: %w", key, body.ID, err)
	}
	// Jira may return the current key if the issue was moved; keep the key we asked for.
	issue := application.Issue{ID: id, Key: key, Summary: body.Fields.Summary}

	c.mu.Lock()
	c.cache[key] = issue
	c.mu.Unlock()
	return issue, nil
}
